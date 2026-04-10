package vm

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/malivvan/rumo/vm/parser"
)

// Issue #8: addChild with nil VM — non-compiled routines unkillable
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
	return NewVM(context.Background(), bytecode, make([]Object, GlobalsSize), -1)
}

func TestIssue8_AbortDoesNotCancelNonCompiledRoutine(t *testing.T) {
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

func TestIssue8_AbortCancelsMultipleNonCompiledRoutines(t *testing.T) {
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

// TestIssue8_DelChildCleansUpCancelFn verifies that delChild calls the cancel
// function to release context resources promptly, and that a subsequent Abort()
// (which re-calls idempotent cancel) does not panic.
func TestIssue8_DelChildCleansUpCancelFn(t *testing.T) {
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

// TestIssue8_MixedCompiledAndNonCompiledAbort verifies that Abort() handles
// both compiled (VM-tracked) and non-compiled (cancelFn-tracked) children.
func TestIssue8_MixedCompiledAndNonCompiledAbort(t *testing.T) {
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

// TestIssue8_AddChildAfterAbortReturnsError verifies that addChild with a
// cancel function correctly returns ErrVMAborted when the parent is aborting.
func TestIssue8_AddChildAfterAbortReturnsError(t *testing.T) {
	v := makeTestVM()
	v.Abort()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := v.addChild(nil, cancel)
	if err != ErrVMAborted {
		t.Fatalf("Issue #8: expected ErrVMAborted after Abort(), got %v", err)
	}
}

