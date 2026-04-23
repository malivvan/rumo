package vm

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

import (
	"strings"
	"testing"

	"github.com/malivvan/rumo/vm/parser"
)

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

