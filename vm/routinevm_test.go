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

