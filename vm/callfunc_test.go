package vm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/malivvan/rumo/vm"
)

// TestCallFuncWithBuiltinFunction verifies that CallFunc correctly
// dispatches to a BuiltinFunction's Call method.
func TestCallFuncWithBuiltinFunction(t *testing.T) {
	called := false
	fn := &vm.BuiltinFunction{
		Name: "test",
		Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			called = true
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 arg, got %d", len(args))
			}
			return args[0], nil
		},
	}

	ctx := context.Background()
	result, err := vm.CallFunc(ctx, fn, &vm.Int{Value: 42})
	if err != nil {
		t.Fatalf("CallFunc with builtin: unexpected error: %v", err)
	}
	if !called {
		t.Fatal("CallFunc with builtin: function was not called")
	}
	intVal, ok := result.(*vm.Int)
	if !ok || intVal.Value != 42 {
		t.Fatalf("CallFunc with builtin: expected Int(42), got %v", result)
	}
}

// TestCallFuncWithCompiledFunctionNoVM verifies that CallFunc returns a
// clear error when asked to run a CompiledFunction but no VM is present
// in the context.
func TestCallFuncWithCompiledFunctionNoVM(t *testing.T) {
	fn := &vm.CompiledFunction{}
	ctx := context.Background()

	_, err := vm.CallFunc(ctx, fn, &vm.Int{Value: 1})
	if err == nil {
		t.Fatal("CallFunc with compiled fn and no VM: expected error, got nil")
	}
}

// TestCallFuncWithNilFunction verifies that CallFunc returns an error
// when called with a nil function object.
func TestCallFuncWithNilFunction(t *testing.T) {
	ctx := context.Background()
	_, err := vm.CallFunc(ctx, nil)
	if err == nil {
		t.Fatal("CallFunc with nil: expected error, got nil")
	}
}
