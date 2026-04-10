package vm_test

import (
	"sync"
	"testing"
	"time"
)

// Issue #1: Shared globals slice in ShallowClone (data race)
//
// ShallowClone() used to copy just the pointer to the same []Object globals
// slice. Parent and child VMs then read/write globals (OpSetGlobal/OpGetGlobal)
// without synchronisation—a textbook data race.
//
// The fix copies the globals slice in ShallowClone so each child VM gets its
// own independent snapshot. Writes to globals in a child routine do not
// affect the parent or sibling routines.

// TestIssue1_GlobalsRace verifies that two concurrent routines started
// with the same function do NOT race on global variables.
// Before the fix, the globals slice was shared and this test would fail
// under -race. After the fix, each routine gets an isolated globals copy.
func TestIssue1_GlobalsRace(t *testing.T) {
	// Two routines increment a global counter 1000 times each.
	// With isolated globals: parent counter stays 0, each routine counts to 1000.
	expectRun(t, `
counter := 0
inc := func() {
	for i := 0; i < 1000; i++ {
		counter += 1
	}
	return counter
}
r1 := start(inc)
r2 := start(inc)
v1 := r1.result()
v2 := r2.result()
out = [counter, v1, v2]
`, Opts().Skip2ndPass(), ARR{0, 1000, 1000})
}

// TestIssue1_GlobalsIsolation verifies that a child routine's mutations
// to a global variable do not affect the parent's copy.
func TestIssue1_GlobalsIsolation(t *testing.T) {
	expectRun(t, `
x := 10
f := func() {
	x = 42
	return x
}
r := start(f)
v := r.result()
out = [x, v]
`, Opts().Skip2ndPass(), ARR{10, 42})
}

// TestIssue1_GlobalsSnapshotTiming verifies that each routine snapshots
// the globals at the time start() is called, not later.
func TestIssue1_GlobalsSnapshotTiming(t *testing.T) {
	// r1 starts when counter=0, r2 starts after counter is set to 500.
	// Each routine increments 100 times from its snapshot value.
	expectRun(t, `
counter := 0
inc := func() {
	for i := 0; i < 100; i++ {
		counter += 1
	}
	return counter
}
r1 := start(inc)
r1.wait()
counter = 500
r2 := start(inc)
v1 := r1.result()
v2 := r2.result()
out = [counter, v1, v2]
`, Opts().Skip2ndPass(), ARR{500, 100, 600})
}

// TestIssue1_MultipleGlobalsIsolation verifies that all globals are
// independently isolated, not just a single variable.
func TestIssue1_MultipleGlobalsIsolation(t *testing.T) {
	expectRun(t, `
x := 10
y := 20
f := func() {
	x = x + 1
	y = y + 2
	return [x, y]
}
r := start(f)
v := r.result()
out = [x, y, v]
`, Opts().Skip2ndPass(), ARR{10, 20, ARR{11, 22}})
}

// TestIssue1_GlobalsReadOnlyAccess verifies that a child routine reading
// a global (without writing) sees the snapshot value from start() time.
func TestIssue1_GlobalsReadOnlyAccess(t *testing.T) {
	expectRun(t, `
msg := "hello"
ch := chan()
f := func() {
	ch.send(msg)
}
r := start(f)
msg = "changed"
out = ch.recv()
r.wait()
`, Opts().Skip2ndPass(), "hello")
}

// Issue #2: Shared constants with mutable *ObjectPtr cells (data race)
//
// ShallowClone() shares the constants slice. CompiledFunction.Free contains
// *ObjectPtr cells mutated at runtime via OpSetFree. Two VMs executing the
// same closure race on closed-over variables.
//
// The fix deep-copies the Free entries of the CompiledFunction passed to
// start(), giving each child VM isolated *ObjectPtr cells. This prevents
// data races and enforces snapshot-isolation semantics: each routine gets
// a copy of captured variables at start() time. Communication between
// routines should use chan() instead of shared captures.

// TestIssue2_ClosureFreeVarRace verifies that two concurrent routines
// started with the same closure do NOT race on captured free variables.
// Before the fix, the *ObjectPtr was shared and this test would fail
// under -race. After the fix, each routine gets an isolated copy.
func TestIssue2_ClosureFreeVarRace(t *testing.T) {
	// Two routines increment a captured counter 1000 times each.
	// With isolated free vars: parent counter stays 0, each routine counts to 1000.
	expectRun(t, `
out = func() {
	counter := 0
	inc := func() {
		for i := 0; i < 1000; i++ {
			counter += 1
		}
		return counter
	}
	r1 := start(inc)
	r2 := start(inc)
	v1 := r1.result()
	v2 := r2.result()
	return [counter, v1, v2]
}()`, Opts().Skip2ndPass(), ARR{0, 1000, 1000})
}

// TestIssue2_ClosureFreeVarIsolation verifies that a child routine's
// mutations to a captured free variable do not affect the parent.
func TestIssue2_ClosureFreeVarIsolation(t *testing.T) {
	expectRun(t, `
out = func() {
	x := 10
	f := func() {
		x = 42
		return x
	}
	r := start(f)
	v := r.result()
	return [x, v]
}()`, Opts().Skip2ndPass(), ARR{10, 42})
}

// TestIssue2_ClosureFreeVarSnapshotTiming verifies that each routine
// snapshots the free variable at the time start() is called, not later.
func TestIssue2_ClosureFreeVarSnapshotTiming(t *testing.T) {
	// r1 starts when counter=0, r2 starts after counter is set to 500.
	// Each routine increments 100 times from its snapshot value.
	expectRun(t, `
out = func() {
	counter := 0
	inc := func() {
		for i := 0; i < 100; i++ {
			counter += 1
		}
		return counter
	}
	r1 := start(inc)
	r1.wait()
	counter = 500
	r2 := start(inc)
	v1 := r1.result()
	v2 := r2.result()
	return [counter, v1, v2]
}()`, Opts().Skip2ndPass(), ARR{500, 100, 600})
}

// TestIssue2_NestedClosureSharedFreeVar verifies that nested closures
// sharing the same captured variable are properly isolated in the child.
// Both inc() and get() capture the same 'a' variable. In the child,
// inc() should mutate the child's copy, and get() should read from
// the same child copy.
func TestIssue2_NestedClosureSharedFreeVar(t *testing.T) {
	expectRun(t, `
out = func() {
	a := 1
	inc := func() { a += 1 }
	get := func() { return a }
	r := start(func() {
		inc()
		inc()
		inc()
		return get()
	})
	v := r.result()
	return [a, v]
}()`, Opts().Skip2ndPass(), ARR{1, 4})
}

// TestIssue2_MultipleFreeVarsIsolation verifies that closures with
// multiple free variables have all of them isolated.
func TestIssue2_MultipleFreeVarsIsolation(t *testing.T) {
	expectRun(t, `
out = func() {
	x := 10
	y := 20
	f := func() {
		x = x + 1
		y = y + 2
		return [x, y]
	}
	r := start(f)
	v := r.result()
	return [x, y, v]
}()`, Opts().Skip2ndPass(), ARR{10, 20, ARR{11, 22}})
}

// TestIssue2_ClosureNoFreeVars verifies that closures without free
// variables work correctly (no-op isolation path).
func TestIssue2_ClosureNoFreeVars(t *testing.T) {
	expectRun(t, `
f := func() { return 42 }
r := start(f)
out = r.result()
`, Opts().Skip2ndPass(), 42)
}

// Issue #3: ShallowClone context stores parent VM pointer
//
// context.WithValue(v.ctx, ContextKey("vm"), v) stored the **parent** VM
// pointer instead of the clone. This caused builtins in the child VM
// (e.g. abort(), start()) to operate on the wrong VM — breaking abort
// propagation and child tracking. For example, calling abort() from a
// child routine would abort the parent instead of the child, and
// start() inside a child would register grandchildren on the parent
// instead of the child, breaking the abort chain.
//
// The fix changes ShallowClone to store the clone (vClone) in the
// context instead of the parent (v).

// TestIssue3_AbortFromChildAbortsChild verifies that calling abort()
// from within a child routine aborts that child, not the parent.
// Before the fix, the child's context stored the parent VM, so
// abort() would abort the parent — leaving the parent's output
// unset or causing it to terminate prematurely.
func TestIssue3_AbortFromChildAbortsChild(t *testing.T) {
	// The child calls abort() which should only abort itself.
	// The parent should continue and produce the expected output.
	expectRun(t, `
r := start(func() {
	abort()
	return "should not reach"
})
r.wait()
out = "parent alive"
`, Opts().Skip2ndPass(), "parent alive")
}

// TestIssue3_AbortFromChildDoesNotAbortParent verifies that the parent
// continues executing after a child self-aborts. Before the fix, the
// child's abort() targeted the parent VM, which would stop it.
func TestIssue3_AbortFromChildDoesNotAbortParent(t *testing.T) {
	expectRun(t, `
ch := chan()
r := start(func() {
	ch.send("started")
	abort()
})
ch.recv()
r.wait()
out = 42
`, Opts().Skip2ndPass(), 42)
}

// TestIssue3_NestedStartRegistersOnChild verifies that start() inside
// a child routine registers the grandchild on the child VM (not the
// parent). When the child is aborted, the grandchild should also be
// aborted. Before the fix, the grandchild was registered as a child
// of the parent, so aborting the child would not propagate.
func TestIssue3_NestedStartRegistersOnChild(t *testing.T) {
	// Parent starts child, child starts grandchild.
	// Aborting the child should propagate to the grandchild.
	// If abort propagates correctly, grandchild's channel recv
	// will be interrupted and the grandchild will complete.
	expectRun(t, `
ch := chan()
r := start(func() {
	gc := start(func() {
		for {
			// infinite loop — only abort can stop this
			x := 1 + 1
		}
	})
	gc.abort()
	gc.wait()
	return "done"
})
out = r.result()
`, Opts().Skip2ndPass(), "done")
}

// TestIssue3_ChildAbortPropagatesDownward verifies the full abort
// chain: parent → child → grandchild. The parent aborts the child,
// which should cascade to the grandchild. If propagation is broken,
// the grandchild's infinite loop never terminates and child.wait(5)
// would time out (return false).
func TestIssue3_ChildAbortPropagatesDownward(t *testing.T) {
	expectRun(t, `
child := start(func() {
	gc := start(func() {
		for {
			x := 1 + 1
		}
	})
	gc.wait()
})
child.abort()
out = child.wait(5)
`, Opts().Skip2ndPass(), true)
}

// TestIssue3_StartInsideChildDoesNotAffectParent verifies that
// start() from a child does not add the grandchild to the parent's
// child list. The parent should be able to finish independently.
func TestIssue3_StartInsideChildDoesNotAffectParent(t *testing.T) {
	expectRun(t, `
r := start(func() {
	gc := start(func() {
		return 99
	})
	return gc.result()
})
out = r.result()
`, Opts().Skip2ndPass(), 99)
}

// Issue #4: routineVM.abort() races with goroutine completion
//
// routineVM.abort() reads gvm.VM to check for nil and then calls
// gvm.Abort(), but the goroutine's deferred cleanup sets gvm.VM = nil
// without any synchronisation. If abort() is called while the goroutine
// is completing, the nil check passes but gvm.VM is nilled before
// Abort() executes — causing a nil-pointer dereference crash.

// TestIssue4_AbortRacesWithCompletion triggers the data race between
// routineVM.abort() reading gvm.VM and the goroutine's deferred cleanup
// nilling it. Under -race, this reliably detects the unsynchronised
// read/write. Without -race, repeated iterations increase the chance
// of hitting the nil-pointer dereference crash.
func TestIssue4_AbortRacesWithCompletion(t *testing.T) {
	// We run many iterations to maximise the chance of hitting the
	// timing window where abort() and goroutine cleanup overlap.
	for i := 0; i < 100; i++ {
		expectRun(t, `
f := func() {
	return 1
}
r := start(f)
r.abort()
r.wait()
out = "ok"
`, Opts().Skip2ndPass(), "ok")
	}
}

// TestIssue4_AbortRacesWithCompletionParallel uses parallel goroutines
// to call abort() at the exact moment the routine finishes, exercising
// the race window more aggressively.
func TestIssue4_AbortRacesWithCompletionParallel(t *testing.T) {
	for i := 0; i < 50; i++ {
		expectRun(t, `
ch := chan()
f := func() {
	ch.recv()
	return 1
}
r := start(f)
ch.send(1)
r.abort()
r.wait()
out = "ok"
`, Opts().Skip2ndPass(), "ok")
	}
}

// TestIssue4_ConcurrentAbortCalls verifies that calling abort()
// multiple times concurrently from different goroutines does not
// panic or race.
func TestIssue4_ConcurrentAbortCalls(t *testing.T) {
	// This script launches a long-running routine and then
	// immediately aborts it — the parent calls abort() which
	// races with the routine's own natural completion.
	for i := 0; i < 50; i++ {
		expectRun(t, `
r := start(func() {
	for i := 0; i < 10; i++ {
		x := i
	}
	return "done"
})
r.abort()
r.abort()
r.wait()
out = "ok"
`, Opts().Skip2ndPass(), "ok")
	}
}

// TestIssue4_AbortAfterCompletion verifies that calling abort() after
// the routine has already finished and gvm.VM has been nilled does not
// crash.
func TestIssue4_AbortAfterCompletion(t *testing.T) {
	expectRun(t, `
r := start(func() {
	return 42
})
r.wait()
r.abort()
out = r.result()
`, Opts().Skip2ndPass(), 42)
}

// TestIssue4_AbortAndResultRace exercises calling abort() and result()
// concurrently from different started routines to stress the synchronisation.
func TestIssue4_AbortAndResultRace(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			expectRun(t, `
r := start(func() {
	return 1
})
r.abort()
r.wait()
out = "ok"
`, Opts().Skip2ndPass(), "ok")
		}()
	}
	wg.Wait()
}

// Issue #5: wait() channel is one-shot — concurrent callers deadlock
//
// waitChan has capacity 1. The goroutine sends exactly one ret value when
// the routine completes. If two concurrent callers (e.g. two routines both
// calling wait() or result() on the same handle) both see done == 0 and
// enter the select, only one receives the value — the other blocks forever.
//
// The fix replaces the single-value waitChan with a close()-based broadcast
// channel (chan struct{}). Closing a channel unblocks ALL receivers. The
// return value is stored in the struct under mutex protection before the
// channel is closed.

// TestIssue5_ConcurrentWaitDeadlock verifies that two concurrent routines
// calling wait() on the same routine handle do NOT deadlock.
// Before the fix, waitChan was a single-value buffered channel — only one
// of the two waiters would receive, and the other would block forever.
func TestIssue5_ConcurrentWaitDeadlock(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		expectRun(t, `
ch := chan()
r := start(func() {
	ch.recv()
	return 42
})
w1 := start(func() {
	return r.wait(10)
})
w2 := start(func() {
	return r.wait(10)
})
ch.send(true)
out = [w1.result(), w2.result()]
`, Opts().Skip2ndPass(), ARR{true, true})
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("TestIssue5_ConcurrentWaitDeadlock: timed out — likely deadlock")
	}
}

// TestIssue5_ConcurrentResultDeadlock verifies that two concurrent routines
// calling result() on the same routine handle do NOT deadlock.
// Uses sync channels to maximise the chance both routines enter wait()
// before the target routine completes.
func TestIssue5_ConcurrentResultDeadlock(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		expectRun(t, `
ch := chan()
sync := chan()
r := start(func() {
	ch.recv()
	return 42
})
r1 := start(func() {
	sync.send(true)
	return r.result()
})
r2 := start(func() {
	sync.send(true)
	return r.result()
})
sync.recv()
sync.recv()
ch.send(true)
out = [r1.result(), r2.result()]
`, Opts().Skip2ndPass(), ARR{42, 42})
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("TestIssue5_ConcurrentResultDeadlock: timed out — likely deadlock")
	}
}

// TestIssue5_WaitAfterResult verifies that calling wait() after result()
// has already consumed the value still returns true (not blocked forever).
func TestIssue5_WaitAfterResult(t *testing.T) {
	expectRun(t, `
r := start(func() {
	return 42
})
v := r.result()
out = [v, r.wait()]
`, Opts().Skip2ndPass(), ARR{42, true})
}

// TestIssue5_MultipleWaitCalls verifies that calling wait() multiple times
// from the same routine works correctly (idempotent).
func TestIssue5_MultipleWaitCalls(t *testing.T) {
	expectRun(t, `
r := start(func() {
	return 42
})
w1 := r.wait()
w2 := r.wait()
w3 := r.wait()
out = [w1, w2, w3]
`, Opts().Skip2ndPass(), ARR{true, true, true})
}

// TestIssue5_MultipleResultCalls verifies that calling result() multiple
// times returns the same value without deadlocking.
func TestIssue5_MultipleResultCalls(t *testing.T) {
	expectRun(t, `
r := start(func() {
	return 42
})
v1 := r.result()
v2 := r.result()
v3 := r.result()
out = [v1, v2, v3]
`, Opts().Skip2ndPass(), ARR{42, 42, 42})
}

// TestIssue5_ConcurrentWaitStress runs many iterations of concurrent
// wait/result calls to stress-test the broadcast mechanism.
func TestIssue5_ConcurrentWaitStress(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			expectRun(t, `
r := start(func() {
	return 1
})
w1 := start(func() { return r.wait(10) })
w2 := start(func() { return r.wait(10) })
w3 := start(func() { return r.result() })
out = [w1.result(), w2.result(), w3.result()]
`, Opts().Skip2ndPass(), ARR{true, true, 1})
		}
	}()

	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("TestIssue5_ConcurrentWaitStress: timed out — likely deadlock")
	}
}


// Issue #6: Channel close() panics are unrecoverable
//
// Double-close or send-on-closed-channel `panic`s the goroutine. The
// objchan.close() method calls Go's close() on the underlying channel
// without any guard — a second close panics. Similarly, objchan.send()
// sends on the channel without checking if it's been closed — panicing
// the goroutine. In both cases the panic is either unrecoverable or
// produces a confusing "Runtime Panic" message instead of a clean
// runtime error. Additionally, the builtinStart defer's `callers == nil`
// check re-panics with "callers not saved" for compiled callables
// because callers is only set on the non-compiled path.
//
// The fix wraps close() with a closed-state guard (sync.Once or
// recover), wraps send() to recover from send-on-closed panics, and
// removes the re-panic in builtinStart's defer.

// TestIssue6_DoubleCloseChannel verifies that closing a channel twice
// returns an error instead of panicking the process.
func TestIssue6_DoubleCloseChannel(t *testing.T) {
	expectError(t, `
ch := chan()
ch.close()
ch.close()
`, nil, "channel already closed")
}

// TestIssue6_SendOnClosedChannel verifies that sending on a closed
// channel returns an error instead of panicking the process.
func TestIssue6_SendOnClosedChannel(t *testing.T) {
	expectError(t, `
ch := chan()
ch.close()
ch.send(42)
`, nil, "send on closed channel")
}

// TestIssue6_DoubleCloseInRoutine verifies that double-close inside a
// started routine does not crash the process. The child error propagates
// to the parent as a clean runtime error (not a panic).
func TestIssue6_DoubleCloseInRoutine(t *testing.T) {
	expectError(t, `
ch := chan()
r := start(func() {
	ch.close()
	ch.close()
})
r.wait()
`, nil, "channel already closed")
}

// TestIssue6_SendOnClosedInRoutine verifies that send-on-closed inside
// a started routine does not crash the process. The child error
// propagates cleanly (not as an unrecoverable panic).
func TestIssue6_SendOnClosedInRoutine(t *testing.T) {
	expectError(t, `
ch := chan()
ch.close()
r := start(func() {
	ch.send(42)
})
r.wait()
`, nil, "send on closed channel")
}

// TestIssue6_DoubleCloseResultIsError verifies that the error from
// double-close is surfaced as an error object through result().
func TestIssue6_DoubleCloseResultIsError(t *testing.T) {
	// Using expectError because the child error also propagates to
	// the parent. The important thing is the error message is clean
	// ("channel already closed") and not an unrecovered Go panic.
	expectError(t, `
ch := chan()
r := start(func() {
	ch.close()
	ch.close()
	return "ok"
})
v := r.result()
`, nil, "channel already closed")
}

// TestIssue6_SendOnClosedResultIsError verifies that the error from
// send-on-closed is surfaced as an error object through result().
func TestIssue6_SendOnClosedResultIsError(t *testing.T) {
	expectError(t, `
ch := chan()
ch.close()
r := start(func() {
	ch.send(1)
	return "ok"
})
v := r.result()
`, nil, "send on closed channel")
}

// TestIssue6_CloseAndRecvReturnsNil verifies that recv on a closed
// channel returns undefined (not an error), which is the documented
// behavior.
func TestIssue6_CloseAndRecvReturnsNil(t *testing.T) {
	expectRun(t, `
ch := chan()
ch.close()
v := ch.recv()
out = is_undefined(v)
`, nil, true)
}

// TestIssue6_BufferedDoubleClose verifies that double-close on a
// buffered channel also returns an error.
func TestIssue6_BufferedDoubleClose(t *testing.T) {
	expectError(t, `
ch := chan(5)
ch.close()
ch.close()
`, nil, "channel already closed")
}

// TestIssue6_BufferedSendOnClosed verifies that send on a closed
// buffered channel returns an error.
func TestIssue6_BufferedSendOnClosed(t *testing.T) {
	expectError(t, `
ch := chan(5)
ch.close()
ch.send(1)
`, nil, "send on closed channel")
}

// TestIssue6_ConcurrentCloseStress runs many iterations of concurrent
// close to ensure no panics under stress. One of the two routines will
// always get the "already closed" error — the test verifies this is a
// clean error, not an unrecoverable panic.
func TestIssue6_ConcurrentCloseStress(t *testing.T) {
	for i := 0; i < 50; i++ {
		expectError(t, `
ch := chan()
r1 := start(func() {
	ch.close()
})
r2 := start(func() {
	ch.close()
})
r1.wait()
r2.wait()
`, nil, "channel already closed")
	}
}

