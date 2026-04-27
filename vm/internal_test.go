package vm

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/malivvan/rumo/vm/parser"
)

// addChild with nil VM — non-compiled routines unkillable
//
// Non-compiled routines (e.g. BuiltinFunction passed to start()) call
// addChild(nil), which increments the WaitGroup but stores nothing in
// vmMap. When Abort() iterates vmMap to kill children, these routines
// are invisible. Although context cascade from v.cancel() currently
// masks this when callCtx derives from the parent, the routines are
// not explicitly tracked and Abort() cannot cancel them independently
// of the context tree.
//
// These tests use an independent context (context.Background()) for the
// simulated non-compiled child to bypass cascade and isolate the
// explicit tracking mechanism that Abort() should provide.

func makeTestVM() *VM {
	fs := parser.NewFileSet()
	fs.AddFile("test", -1, 1)
	bytecode := &Bytecode{
		MainFunction: emptyEntry,
		FileSet:      fs,
	}
	return NewVM(context.Background(), bytecode, make([]Object, DefaultConfig.GlobalsSize), nil)
}

func TestAbortDoesNotCancelNonCompiledRoutine(t *testing.T) {
	v := makeTestVM()

	// Create an independent context for a non-compiled routine.
	// This is NOT derived from the parent VM's context, so context
	// cascade from v.cancel() will NOT reach it. Only explicit
	// cancel tracking in Abort() can cancel it.
	routineCtx, routineCancel := context.WithCancel(context.Background())
	defer routineCancel() // safety cleanup

	var cancelled int64
	routineDone := make(chan struct{})

	// Register the non-compiled child with the parent, passing the
	// routine's cancel function so Abort() can explicitly cancel it.
	if err := v.addChild(nil, routineCancel); err != nil {
		t.Fatal(err)
	}

	go func() {
		defer func() {
			v.delChild(nil, routineCancel)
			close(routineDone)
		}()
		<-routineCtx.Done()
		atomic.StoreInt64(&cancelled, 1)
	}()

	// Abort the parent VM. This should explicitly cancel the
	// non-compiled routine via the tracked cancel function.
	v.Abort()

	select {
	case <-routineDone:
		// The routine was cancelled — Abort() handled it correctly.
	case <-time.After(3 * time.Second):
		routineCancel() // unblock the goroutine for cleanup
		<-routineDone
		t.Fatal("Issue #8: Abort() did not cancel non-compiled routine — " +
			"it is unkillable because its cancel function is not tracked in vmChildCtl")
	}

	if atomic.LoadInt64(&cancelled) != 1 {
		t.Fatal("Issue #8: non-compiled routine's context was not cancelled by Abort()")
	}
}

func TestAbortCancelsMultipleNonCompiledRoutines(t *testing.T) {
	v := makeTestVM()

	const numRoutines = 5
	var cancelledCount int64
	routineDone := make(chan struct{}, numRoutines)

	cancels := make([]context.CancelFunc, numRoutines)
	for i := 0; i < numRoutines; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancels[i] = cancel
		if err := v.addChild(nil, cancel); err != nil {
			t.Fatal(err)
		}
		go func(c context.Context, cancelFn context.CancelFunc) {
			defer func() {
				v.delChild(nil, cancelFn)
				routineDone <- struct{}{}
			}()
			<-c.Done()
			atomic.AddInt64(&cancelledCount, 1)
		}(ctx, cancel)
	}
	defer func() {
		for _, c := range cancels {
			c()
		}
	}()

	v.Abort()

	for i := 0; i < numRoutines; i++ {
		select {
		case <-routineDone:
		case <-time.After(3 * time.Second):
			for _, c := range cancels {
				c()
			}
			t.Fatalf("Issue #8: only %d/%d non-compiled routines cancelled by Abort()",
				atomic.LoadInt64(&cancelledCount), numRoutines)
		}
	}

	if got := atomic.LoadInt64(&cancelledCount); got != numRoutines {
		t.Fatalf("Issue #8: expected %d cancellations, got %d", numRoutines, got)
	}
}

// TestDelChildCleansUpCancelFn verifies that delChild calls the cancel
// function to release context resources promptly, and that a subsequent Abort()
// (which re-calls idempotent cancel) does not panic.
func TestDelChildCleansUpCancelFn(t *testing.T) {
	v := makeTestVM()

	_, routineCancel := context.WithCancel(context.Background())

	if err := v.addChild(nil, routineCancel); err != nil {
		t.Fatal(err)
	}

	// delChild should call routineCancel for prompt cleanup.
	v.delChild(nil, routineCancel)

	// Abort after delChild must not panic even though routineCancel was
	// already called — cancel functions are idempotent.
	v.Abort()
}

// TestMixedCompiledAndNonCompiledAbort verifies that Abort() handles
// both compiled (VM-tracked) and non-compiled (cancelFn-tracked) children.
func TestMixedCompiledAndNonCompiledAbort(t *testing.T) {
	parent := makeTestVM()

	// Simulate a compiled child by creating a real child VM.
	child := makeTestVM()
	if err := parent.addChild(child); err != nil {
		t.Fatal(err)
	}

	// Simulate a non-compiled child with a tracked cancel function.
	routineCtx, routineCancel := context.WithCancel(context.Background())
	defer routineCancel()

	if err := parent.addChild(nil, routineCancel); err != nil {
		t.Fatal(err)
	}

	var nonCompiledDone int64
	doneCh := make(chan struct{})
	go func() {
		<-routineCtx.Done()
		atomic.StoreInt64(&nonCompiledDone, 1)
		close(doneCh)
	}()

	parent.Abort()

	// The compiled child should be aborted.
	if atomic.LoadInt64(&child.aborting) == 0 {
		t.Fatal("Issue #8: compiled child was not aborted")
	}

	// The non-compiled child should be cancelled via its cancel function.
	select {
	case <-doneCh:
	case <-time.After(3 * time.Second):
		t.Fatal("Issue #8: non-compiled routine was not cancelled in mixed scenario")
	}

	if atomic.LoadInt64(&nonCompiledDone) != 1 {
		t.Fatal("Issue #8: non-compiled routine context was not cancelled")
	}

	// Cleanup
	parent.delChild(child)
	parent.delChild(nil, routineCancel)
}

// TestAddChildAfterAbortReturnsError verifies that addChild with a
// cancel function correctly returns ErrVMAborted when the parent is aborting.
func TestAddChildAfterAbortReturnsError(t *testing.T) {
	v := makeTestVM()
	v.Abort()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := v.addChild(nil, cancel)
	if err != ErrVMAborted {
		t.Fatalf("Issue #8: expected ErrVMAborted after Abort(), got %v", err)
	}
}

// Abort() TOCTOU between atomic.Load and Lock
//
// Abort() first performs an atomic.Load check:
//
//     if atomic.LoadInt64(&v.aborting) != 0 { return }
//     v.childCtl.Lock()
//     atomic.StoreInt64(&v.aborting, 1)
//     v.cancel()
//     ...
//     v.childCtl.Unlock()
//
// Two concurrent Abort() callers can both observe aborting==0 at the atomic
// check and both proceed toward the lock. The first goroutine acquires the
// lock, stores aborting=1, calls cancel(), iterates children, then unlocks.
// The second goroutine has already passed the check with aborting==0, so it
// acquires the lock next and repeats the entire body — calling cancel() and
// every child Abort() a second time.
//
// The fix replaces the load+check+store sequence with an atomic.CompareAndSwap,
// guaranteeing that exactly one goroutine transitions aborting 0→1. All other
// goroutines that lose the CAS exit immediately.

// TestAbortConcurrentTOCTOU reproduces the TOCTOU bug deterministically.
//
// We hold childCtl.Lock() before launching two Abort() goroutines. Both
// goroutines execute the atomic check (aborting==0, passes) and then block
// on Lock(). After a brief sleep — during which both goroutines are provably
// blocked on Lock() — we release the lock. Both goroutines now run the abort
// body in sequence: with the bug, the tracked cancelFn is called twice
// (callCount==2); with the CAS fix, only the first goroutine succeeds and the
// cancelFn is called exactly once (callCount==1).
func TestAbortConcurrentTOCTOU(t *testing.T) {
	var callCount int64
	cancelFn := func() {
		atomic.AddInt64(&callCount, 1)
	}

	v := makeTestVM()
	if err := v.addChild(nil, cancelFn); err != nil {
		t.Fatal(err)
	}

	// Hold the mutex BEFORE launching the goroutines.
	// Both goroutines will execute the atomic.Load check (seeing aborting==0),
	// pass the check, and then block at v.childCtl.Lock() — because we hold it.
	v.childCtl.Lock()

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			v.Abort()
		}()
	}

	// Give goroutines time to be scheduled, execute the atomic check, and
	// block at v.childCtl.Lock(). 20 ms is generous for any real system.
	time.Sleep(20 * time.Millisecond)

	// Release the lock. Both goroutines have already passed the atomic check.
	// Without the fix: goroutine 1 runs the body (callCount=1), unlocks;
	// goroutine 2 runs the body again (callCount=2), unlocks.
	// With the fix (CAS): goroutine 1 sets aborting=1 and calls cancelFn;
	// goroutine 2's CAS fails, it returns immediately — callCount stays 1.
	v.childCtl.Unlock()
	wg.Wait()

	got := atomic.LoadInt64(&callCount)
	if got != 1 {
		t.Fatalf("TOCTOU bug: Abort() called cancelFn %d times, expected exactly 1 — "+
			"both goroutines passed the atomic.Load check before the lock was released, "+
			"then both ran the abort body sequentially", got)
	}
}

// TestAbortIdempotentSingleCaller verifies that calling Abort() multiple times
// from a single goroutine does not call the tracked cancel function more than once.
func TestAbortIdempotentSingleCaller(t *testing.T) {
	var callCount int64
	cancelFn := func() {
		atomic.AddInt64(&callCount, 1)
	}

	v := makeTestVM()
	if err := v.addChild(nil, cancelFn); err != nil {
		t.Fatal(err)
	}

	v.Abort()
	v.Abort()
	v.Abort()

	got := atomic.LoadInt64(&callCount)
	if got != 1 {
		t.Fatalf("Abort() called cancelFn %d times, expected exactly 1", got)
	}
}

// TestAbortConcurrentManyGoroutines stress-tests that many concurrent Abort()
// calls produce exactly one invocation of each tracked cancel function.
func TestAbortConcurrentManyGoroutines(t *testing.T) {
	const N = 50

	for iter := 0; iter < 20; iter++ {
		var callCount int64
		cancelFn := func() {
			atomic.AddInt64(&callCount, 1)
		}

		v := makeTestVM()
		if err := v.addChild(nil, cancelFn); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func() {
				defer wg.Done()
				<-start
				v.Abort()
			}()
		}
		close(start)
		wg.Wait()

		got := atomic.LoadInt64(&callCount)
		if got != 1 {
			t.Fatalf("iter %d: Abort() called cancelFn %d times with %d concurrent callers, expected exactly 1",
				iter, got, N)
		}
	}
}

// TestAbortChildAbortedExactlyOnce verifies that a child VM's aborting flag
// is only set once even when the parent Abort() is called concurrently.
func TestAbortChildAbortedExactlyOnce(t *testing.T) {
	parent := makeTestVM()
	child := makeTestVM()

	if err := parent.addChild(child); err != nil {
		t.Fatal(err)
	}

	// Hold the lock to force both goroutines past the atomic check simultaneously.
	parent.childCtl.Lock()

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			parent.Abort()
		}()
	}

	time.Sleep(20 * time.Millisecond)
	parent.childCtl.Unlock()
	wg.Wait()

	// cleanup child from parent's WaitGroup so makeTestVM can be GC'd
	parent.delChild(child)

	// The parent flag should be exactly 1.
	if got := atomic.LoadInt64(&parent.aborting); got != 1 {
		t.Fatalf("parent.aborting = %d, expected 1", got)
	}
	// The child flag should be exactly 1.
	if got := atomic.LoadInt64(&child.aborting); got != 1 {
		t.Fatalf("child.aborting = %d, expected 1", got)
	}
}

// checkGrowStack sets v.err but RunCompiled continues to access v.stack
//
// When a CompiledFunction with a very large NumLocals is called via
// RunCompiled, OpCall sets:
//
//   v.sp = v.sp - numArgs + callee.NumLocals
//
// If the jump lands at or beyond StackSize the end-of-loop
// checkGrowStack(0) correctly sets v.err = ErrStackOverflow and returns
// false, causing run() to return.  However the deferred body of
// RunCompiled then contains:
//
//   if fn != nil && atomic.LoadInt64(&v.aborting) == 0 {
//       val = v.stack[v.sp-1]   // ← sp may be >> len(v.stack)
//   }
//
// Because v.err is never consulted here, an OOB access panics when the
// stack was never grown to match the overflowed sp value.
//
// Fix: guard the return-value extraction with `v.err == nil` so that
// a stack overflow exits cleanly with ErrStackOverflow instead of a
// runtime panic.

// bigLocalsFn returns a CompiledFunction whose single call increases sp
// from its current position by exactly numLocals slots.  With
// numLocals == StackSize the very first call pushes sp to StackSize or
// beyond, hitting the overflow check before any stack growth can occur.
func bigLocalsFn(numLocals int) *CompiledFunction {
	return &CompiledFunction{
		// The body only needs to return; locals are never actually used.
		Instructions:  MakeInstruction(parser.OpReturn, 0),
		NumLocals:     numLocals,
		NumParameters: 0,
	}
}

// TestStackOverflowFromLargeNumLocalsDoesNotPanic is the primary
// regression test.  Before the fix, RunCompiled panics with an index
// out of range because the stack was never grown to hold the overflowed
// sp value.  After the fix it must return a wrapped ErrStackOverflow
// error without panicking.
func TestStackOverflowFromLargeNumLocalsDoesNotPanic(t *testing.T) {
	fn := bigLocalsFn(DefaultConfig.StackSize) // one call: sp → ~StackSize+5 (overflow)

	v := makeTestVM()

	var (
		runErr    error
		didPanic  bool
		panicInfo interface{}
	)

	// Wrap the call so that a panic is treated as a test failure rather
	// than crashing the whole test binary.
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicInfo = r
			}
		}()
		_, runErr = v.RunCompiled(fn)
	}()

	if didPanic {
		t.Fatalf(
			"RunCompiled panicked with an index-out-of-range error instead of "+
				"returning ErrStackOverflow cleanly.\n"+
				"This reproduces the checkGrowStack-sets-v.err-but-execution-continues bug.\n"+
				"Panic value: %v",
			panicInfo,
		)
	}
	if runErr == nil {
		t.Fatal("expected a stack-overflow error, got nil")
	}
	if !strings.Contains(runErr.Error(), "stack overflow") {
		t.Fatalf("expected 'stack overflow' in error, got: %v", runErr)
	}
}

// TestStackOverflowFromLargeNumLocalsIsIdempotent verifies that calling
// RunCompiled a second time on the same (now-reset) VM also returns a
// clean error.  This guards against any stale sp value leaking between
// runs.
func TestStackOverflowFromLargeNumLocalsIsIdempotent(t *testing.T) {
	fn := bigLocalsFn(DefaultConfig.StackSize)
	v := makeTestVM()

	for i := 0; i < 3; i++ {
		var (
			runErr   error
			didPanic bool
		)
		func() {
			defer func() {
				if recover() != nil {
					didPanic = true
				}
			}()
			_, runErr = v.RunCompiled(fn)
		}()

		if didPanic {
			t.Fatalf("attempt %d: RunCompiled panicked (expected clean ErrStackOverflow)", i+1)
		}
		if runErr == nil {
			t.Fatalf("attempt %d: expected stack overflow error, got nil", i+1)
		}
		if !strings.Contains(runErr.Error(), "stack overflow") {
			t.Fatalf("attempt %d: unexpected error: %v", i+1, runErr)
		}
	}
}

// TestStackOverflowFromDeepRecursionDoesNotPanic is a regression test for
// the MaxFrames code path in run().  It stores the recursive function in
// globals[0] so the function can call itself, driving framesIndex past
// MaxFrames.  An OpNull between OpCall and OpReturn breaks the
// tail-call-optimisation pattern (which would otherwise infinite-loop
// without growing frames) so that each call genuinely pushes a new frame.
//
// Unlike the large-NumLocals tests above, this test does NOT panic before
// the fix — when MaxFrames fires, sp is still within len(v.stack).  It is
// included as a regression guard to ensure clean error handling for deep
// legitimate recursion.
func TestStackOverflowFromDeepRecursionDoesNotPanic(t *testing.T) {
	v := makeTestVM()

	// selfCall body (in pseudo-asm):
	//   GetGlobal 0      ; push self
	//   Call 0 0         ; call self() → result on stack
	//   Null             ; push undefined  (breaks tail-call pattern)
	//   Pop              ; discard it
	//   Return 1         ; return the nested call's result
	selfCall := &CompiledFunction{
		NumLocals:     1,
		NumParameters: 0,
	}
	selfCall.Instructions = concatInsts(
		MakeInstruction(parser.OpGetGlobal, 0),
		MakeInstruction(parser.OpCall, 0, 0),
		MakeInstruction(parser.OpNull), // prevents tail-call optimisation
		MakeInstruction(parser.OpPop),
		MakeInstruction(parser.OpReturn, 1),
	)
	v.globals[0] = selfCall

	var (
		runErr   error
		didPanic bool
		pval     interface{}
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				pval = r
			}
		}()
		_, runErr = v.RunCompiled(selfCall)
	}()

	if didPanic {
		t.Fatalf("RunCompiled panicked during deep recursion: %v", pval)
	}
	if runErr == nil {
		t.Fatal("expected a stack-overflow error from deep recursion, got nil")
	}
	if !strings.Contains(runErr.Error(), "stack overflow") {
		t.Fatalf("expected 'stack overflow' in error, got: %v", runErr)
	}
}
