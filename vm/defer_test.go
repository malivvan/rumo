package vm_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/parser"
)

// TestDefer_Basic verifies that a deferred call executes after the
// function returns.
func TestDefer_Basic(t *testing.T) {
	expectRun(t, `
out = func() {
	a := []
	defer func() { a = append(a, 3) }()
	a = append(a, 1)
	a = append(a, 2)
	return a
}()
`, Opts().Skip2ndPass(), ARR{1, 2})

	// Verify deferred function ran by observing side effect on global
	expectRun(t, `
x := 0
f := func() {
	defer func() { x = 42 }()
	return 1
}
f()
out = x
`, Opts().Skip2ndPass(), 42)
}

// TestDefer_LIFOOrder verifies that multiple deferred calls execute
// in last-in-first-out order.
func TestDefer_LIFOOrder(t *testing.T) {
	expectRun(t, `
result := []
f := func() {
	defer func() { result = append(result, 1) }()
	defer func() { result = append(result, 2) }()
	defer func() { result = append(result, 3) }()
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{3, 2, 1})
}

// TestDefer_ArgsEvaluatedAtDeferTime verifies that arguments to
// deferred calls are evaluated when the defer statement executes,
// not when the deferred function runs.
func TestDefer_ArgsEvaluatedAtDeferTime(t *testing.T) {
	expectRun(t, `
result := []
log := func(v) { result = append(result, v) }
f := func() {
	x := 0
	defer log(x)
	x = 1
	defer log(x)
	x = 2
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{1, 0})
}

// TestDefer_ReturnValueUnaffected verifies that the return value is
// determined before deferred calls execute.
func TestDefer_ReturnValueUnaffected(t *testing.T) {
	expectRun(t, `
f := func() {
	x := 10
	defer func() { x = 99 }()
	return x
}
out = f()
`, Opts().Skip2ndPass(), 10)
}

// TestDefer_InLoop verifies that defer inside a loop accumulates
// calls that all execute on function return.
func TestDefer_InLoop(t *testing.T) {
	expectRun(t, `
result := []
f := func() {
	for i := 0; i < 5; i++ {
		defer func(n) { result = append(result, n) }(i)
	}
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{4, 3, 2, 1, 0})
}

// TestDefer_WithClosures verifies that deferred closures capture
// variables correctly.
func TestDefer_WithClosures(t *testing.T) {
	expectRun(t, `
out = func() {
	result := []
	for i := 0; i < 3; i++ {
		v := i * 10
		defer func() { result = append(result, v) }()
	}
	return result
}()
`, Opts().Skip2ndPass(), ARR{})

	// After return, deferred closures run. Check via global.
	expectRun(t, `
result := []
f := func() {
	for i := 0; i < 3; i++ {
		v := i * 10
		defer func() { result = append(result, v) }()
	}
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{20, 10, 0})
}

// TestDefer_NestedFunctions verifies that defer works correctly
// in nested function calls.
func TestDefer_NestedFunctions(t *testing.T) {
	expectRun(t, `
result := []
inner := func() {
	defer func() { result = append(result, "inner") }()
	result = append(result, "inner-body")
}
outer := func() {
	defer func() { result = append(result, "outer") }()
	inner()
	result = append(result, "outer-body")
}
outer()
out = result
`, Opts().Skip2ndPass(), ARR{"inner-body", "inner", "outer-body", "outer"})
}

// TestDefer_DeferInDeferredCall verifies that a deferred function
// that itself uses defer works correctly.
func TestDefer_DeferInDeferredCall(t *testing.T) {
	expectRun(t, `
result := []
f := func() {
	defer func() {
		defer func() { result = append(result, "nested-defer") }()
		result = append(result, "defer-body")
	}()
	result = append(result, "main-body")
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{"main-body", "defer-body", "nested-defer"})
}

// TestDefer_WithReturn verifies that defer runs even when the
// function has an explicit return in the middle.
func TestDefer_WithReturn(t *testing.T) {
	expectRun(t, `
result := []
f := func(x) {
	defer func() { result = append(result, "deferred") }()
	if x > 0 {
		return x
	}
	result = append(result, "unreachable")
	return 0
}
v := f(5)
out = [v, result]
`, Opts().Skip2ndPass(), ARR{5, ARR{"deferred"}})
}

// TestDefer_NoReturn verifies that defer runs when a function
// exits without an explicit return statement.
func TestDefer_NoReturn(t *testing.T) {
	expectRun(t, `
result := []
f := func() {
	defer func() { result = append(result, "deferred") }()
	result = append(result, "body")
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{"body", "deferred"})
}

// TestDefer_BuiltinFunction verifies that deferring a builtin
// function works correctly.
func TestDefer_BuiltinFunction(t *testing.T) {
	expectRun(t, `
arr := [1, 2, 3]
f := func() {
	defer append(arr, 4)
	return len(arr)
}
v := f()
out = v
`, Opts().Skip2ndPass(), 3)
}

// TestDefer_NonCompiledCallable verifies that deferring a
// user-provided non-compiled callable works correctly.
func TestDefer_NonCompiledCallable(t *testing.T) {
	var captured int64
	captureFn := &vm.BuiltinFunction{
		Name: "capture_fn",
		Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			captured = args[0].(*vm.Int).Value
			return nil, nil
		},
	}

	expectRun(t, `
f := func() {
	defer capture_fn(42)
	return "ok"
}
out = f()
`, Opts().Skip2ndPass().Symbol("capture_fn", captureFn), "ok")

	if captured != 42 {
		t.Fatalf("expected captured=42, got %d", captured)
	}
}

// TestDefer_MultipleReturns verifies that defer fires exactly once
// regardless of which return path is taken.
func TestDefer_MultipleReturns(t *testing.T) {
	expectRun(t, `
count := 0
f := func(x) {
	defer func() { count += 1 }()
	if x == 1 { return "one" }
	if x == 2 { return "two" }
	return "other"
}
r1 := f(1)
r2 := f(2)
r3 := f(3)
out = [r1, r2, r3, count]
`, Opts().Skip2ndPass(), ARR{"one", "two", "other", 3})
}

// TestDefer_WithRoutine verifies that defer works correctly inside
// routines started with go.
func TestDefer_WithRoutine(t *testing.T) {
	expectRun(t, `
ch := chan(1)
f := func() {
	defer func() { ch.send("deferred") }()
	return 42
}
r := start f()
v := r.result()
msg := ch.recv()
out = [v, msg]
`, Opts().Skip2ndPass(), ARR{42, "deferred"})
}

// TestDefer_CompileError verifies that defer outside a function
// produces a compile error.
func TestDefer_CompileError(t *testing.T) {
	expectError(t, `defer func(){}()`, nil, "defer not allowed outside function")
}

// TestDefer_NotACallError verifies that defer with a non-call
// expression produces a parse error.
func TestDefer_NotACallError(t *testing.T) {
	input := `f := func() { defer 42 }`
	testFileSet := parser.NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))
	p := parser.NewParser(testFile, []byte(input), nil)
	_, err := p.ParseFile()
	if err == nil {
		t.Fatal("expected parse error for 'defer 42'")
	}
	if !strings.Contains(err.Error(), "function call") {
		t.Fatalf("expected error about function call, got: %s", err.Error())
	}
}

// makeDeferTestBuiltins returns a pair of builtin functions and a reader for
// recorded events. `block` blocks until its context is cancelled; `record`
// appends to a shared event log protected by a mutex.
func makeDeferTestBuiltins() (*vm.BuiltinFunction, *vm.BuiltinFunction, func() []string) {
	var mu sync.Mutex
	var events []string
	blockFn := &vm.BuiltinFunction{
		Name: "block",
		Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			<-ctx.Done()
			return vm.UndefinedValue, nil
		},
	}
	recordFn := &vm.BuiltinFunction{
		Name: "record",
		Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			mu.Lock()
			events = append(events, args[0].(*vm.String).Value)
			mu.Unlock()
			return vm.UndefinedValue, nil
		},
	}
	read := func() []string {
		mu.Lock()
		defer mu.Unlock()
		cp := make([]string, len(events))
		copy(cp, events)
		return cp
	}
	return blockFn, recordFn, read
}

// runDeferStopScript runs a rumo script that registers defers in a routine,
// synchronises with the main VM via a startCh channel, then cancels the
// routine. Returns the recorded events.
func runDeferStopScript(t *testing.T, body string) []string {
	t.Helper()
	blockFn, recordFn, read := makeDeferTestBuiltins()

	// The script pattern: the routine registers its defers, sends on
	// startCh to signal readiness, then calls block() which waits for
	// routine to stop. The main VM receives on startCh, stops the
	// routine, and waits for it to finish. This avoids the race where
	// stop fires before defers are registered.
	script := `
startCh := chan()
r := start func() {
` + body + `
}()
startCh.recv()
r.stop()
r.wait()
out = "ok"
`
	done := make(chan struct{})
	go func() {
		defer close(done)
		expectRun(t, script, Opts().Skip2ndPass().
			Symbol("block", blockFn).
			Symbol("record", recordFn), "ok")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for script to finish")
	}
	return read()
}

// TestDeferOnStopSimple verifies that a deferred function runs when
// the routine is stopped while its body is blocked.
func TestDeferOnStopSimple(t *testing.T) {
	events := runDeferStopScript(t, `
	defer record("deferred")
	startCh.send(true)
	block()
	record("after-block")
`)
	foundDeferred := false
	for _, e := range events {
		if e == "deferred" {
			foundDeferred = true
		}
		if e == "after-block" {
			t.Fatalf("body continued after block returned from stop, got %v", events)
		}
	}
	if !foundDeferred {
		t.Fatalf("deferred call was not executed, got %v", events)
	}
}

// TestDeferOnStopLIFO verifies that multiple defers in a stopped
// routine still execute in LIFO order.
func TestDeferOnStopLIFO(t *testing.T) {
	events := runDeferStopScript(t, `
	defer record("d1")
	defer record("d2")
	defer record("d3")
	startCh.send(true)
	block()
`)
	want := []string{"d3", "d2", "d1"}
	if len(events) != 3 {
		t.Fatalf("expected %v, got %v", want, events)
	}
	for i, w := range want {
		if events[i] != w {
			t.Fatalf("expected %v, got %v", want, events)
		}
	}
}

// TestDeferOnStopNestedFrames verifies that defers across multiple
// nested frames all run when the deepest frame is stopped.
func TestDeferOnStopNestedFrames(t *testing.T) {
	events := runDeferStopScript(t, `
	inner := func(sig) {
		defer record("inner-defer")
		sig.send(true)
		block()
	}
	defer record("outer-defer")
	inner(startCh)
`)
	want := []string{"inner-defer", "outer-defer"}
	if len(events) != 2 {
		t.Fatalf("expected %v, got %v", want, events)
	}
	for i, w := range want {
		if events[i] != w {
			t.Fatalf("expected %v, got %v", want, events)
		}
	}
}

// TestDeferOnStopCompiledDefer verifies that defers running compiled
// functions (bytecode, not just builtins) execute correctly during stop.
func TestDeferOnStopCompiledDefer(t *testing.T) {
	// The deferred function body contains multiple statements to force
	// the compiled-function deferred-call path (not a single builtin).
	events := runDeferStopScript(t, `
	defer func() {
		msg := "compiled-defer"
		record(msg)
	}()
	startCh.send(true)
	block()
`)
	if len(events) != 1 || events[0] != "compiled-defer" {
		t.Fatalf("expected compiled deferred function to run, got %v", events)
	}
}

// TestDeferOnStopDeferWithDeferredDefer verifies a defer inside a
// deferred function also runs when the routine is cancelled.
func TestDeferOnStopDeferWithDeferredDefer(t *testing.T) {
	events := runDeferStopScript(t, `
	defer func() {
		defer record("nested-defer")
		record("outer-defer-body")
	}()
	startCh.send(true)
	block()
`)
	want := []string{"outer-defer-body", "nested-defer"}
	if len(events) != 2 {
		t.Fatalf("expected %v, got %v", want, events)
	}
	for i, w := range want {
		if events[i] != w {
			t.Fatalf("expected %v, got %v", want, events)
		}
	}
}

// TestDeferOnStopBuiltin verifies that defers run when the stop()
// builtin is called from within the routine itself (self-cancellation).
func TestDeferOnStopBuiltin(t *testing.T) {
	_, recordFn, read := makeDeferTestBuiltins()

	done := make(chan struct{})
	go func() {
		defer close(done)
		expectRun(t, `
r := start func() {
	defer record("deferred")
	record("before-stop")
	stop()
	record("unreachable")
}()
r.wait()
out = "ok"
`, Opts().Skip2ndPass().Symbol("record", recordFn), "ok")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("TestDeferOnStopBuiltin: timed out")
	}

	events := read()
	foundDeferred := false
	foundBefore := false
	for _, e := range events {
		if e == "deferred" {
			foundDeferred = true
		}
		if e == "before-stop" {
			foundBefore = true
		}
		if e == "unreachable" {
			t.Fatalf("stop() should have prevented 'unreachable', got %v", events)
		}
	}
	if !foundBefore {
		t.Fatalf("expected 'before-stop' to have run, got %v", events)
	}
	if !foundDeferred {
		t.Fatalf("defer was not executed when stop() fired, got %v", events)
	}
}

// TestDefer_Spread verifies that defer works with spread arguments.
func TestDefer_Spread(t *testing.T) {
	expectRun(t, `
result := []
log := func(...args) {
	for v in args {
		result = append(result, v)
	}
}
f := func() {
	defer log([1, 2, 3]...)
}
f()
out = result
`, Opts().Skip2ndPass(), ARR{1, 2, 3})
}
