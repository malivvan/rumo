package cui

import (
	"context"
	"errors"
	"testing"

	"github.com/malivvan/rumo/vm"
)

// ---------------------------------------------------------------------------
// Issue #20: cui: Stale context capture in widget callbacks
// Callbacks capture ctx at registration time. When the registering routine
// exits/aborts, the context is cancelled and the VM may be torn down, but
// the tview event loop still invokes the callback — use-after-free style
// corruption.
// ---------------------------------------------------------------------------

func TestIssue20_SafeCallFuncWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before callback invocation

	called := false
	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
			called = true
			return vm.UndefinedValue, nil
		},
	}

	result, err := safeCallFunc(ctx, cb)

	if called {
		t.Fatal("callback should not be called when context is cancelled")
	}
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if result != vm.UndefinedValue {
		t.Fatal("expected UndefinedValue when context is cancelled")
	}
}

func TestIssue20_SafeCallFuncWithActiveContext(t *testing.T) {
	ctx := context.Background()

	called := false
	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
			called = true
			return &vm.String{Value: "ok"}, nil
		},
	}

	// Suppress error handler for clean test
	origHandler := CallbackErrorHandler
	CallbackErrorHandler = nil
	defer func() { CallbackErrorHandler = origHandler }()

	result, err := safeCallFunc(ctx, cb)

	if !called {
		t.Fatal("callback should be called when context is active")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != `"ok"` {
		t.Fatalf("expected 'ok', got %v", result)
	}
}

func TestIssue20_SafeCallFuncWithArgsAndCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, args ...vm.Object) (vm.Object, error) {
			t.Fatal("should not be called")
			return vm.UndefinedValue, nil
		},
	}

	_, err := safeCallFunc(ctx, cb, &vm.String{Value: "arg1"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---------------------------------------------------------------------------
// Issue #21: cui: Callback errors silently discarded
// All 19 callFunc call sites in cui.go ignore both return values. VM clone
// failures, panics, and rumo errors vanish silently.
// ---------------------------------------------------------------------------

func TestIssue21_SafeCallFuncReportsErrors(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New("callback failed")
	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
			return nil, expectedErr
		},
	}

	var capturedErr error
	origHandler := CallbackErrorHandler
	CallbackErrorHandler = func(err error) {
		capturedErr = err
	}
	defer func() { CallbackErrorHandler = origHandler }()

	_, err := safeCallFunc(ctx, cb)

	if err == nil {
		t.Fatal("expected error from callback")
	}
	if capturedErr == nil {
		t.Fatal("CallbackErrorHandler should have been called")
	}
	if capturedErr.Error() != expectedErr.Error() {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, capturedErr)
	}
}

func TestIssue21_SafeCallFuncErrorHandlerNil(t *testing.T) {
	ctx := context.Background()

	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
			return nil, errors.New("some error")
		},
	}

	origHandler := CallbackErrorHandler
	CallbackErrorHandler = nil
	defer func() { CallbackErrorHandler = origHandler }()

	// Should not panic even with nil handler
	_, err := safeCallFunc(ctx, cb)
	if err == nil {
		t.Fatal("expected error from callback")
	}
}

func TestIssue21_SafeCallFuncSuccessNoErrorHandler(t *testing.T) {
	ctx := context.Background()

	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
			return &vm.Int{Value: 42}, nil
		},
	}

	var handlerCalled bool
	origHandler := CallbackErrorHandler
	CallbackErrorHandler = func(err error) {
		handlerCalled = true
	}
	defer func() { CallbackErrorHandler = origHandler }()

	result, err := safeCallFunc(ctx, cb)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handlerCalled {
		t.Fatal("error handler should not be called on success")
	}
	if result.(*vm.Int).Value != 42 {
		t.Fatalf("expected 42, got %v", result)
	}
}

// Regression: verify safeCallFunc passes args through
func TestIssue20_SafeCallFuncPassesArgs(t *testing.T) {
	ctx := context.Background()

	var gotArgs []vm.Object
	cb := &vm.BuiltinFunction{
		Name: "test_cb",
		Value: func(_ context.Context, args ...vm.Object) (vm.Object, error) {
			gotArgs = args
			return vm.UndefinedValue, nil
		},
	}

	origHandler := CallbackErrorHandler
	CallbackErrorHandler = nil
	defer func() { CallbackErrorHandler = origHandler }()

	arg1 := &vm.String{Value: "hello"}
	arg2 := &vm.Int{Value: 99}
	_, err := safeCallFunc(ctx, cb, arg1, arg2)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotArgs) != 2 {
		t.Fatalf("expected 2 args, got %d", len(gotArgs))
	}
	if gotArgs[0].(*vm.String).Value != "hello" {
		t.Fatalf("expected 'hello', got %v", gotArgs[0])
	}
	if gotArgs[1].(*vm.Int).Value != 99 {
		t.Fatalf("expected 99, got %v", gotArgs[1])
	}
}

