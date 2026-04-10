package vm_test

import (
	"testing"
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
