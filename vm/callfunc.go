package vm

import (
	"context"
	"fmt"
)

// CallFunc invokes a rumo callback (compiled or builtin). For
// CompiledFunction values it extracts the parent VM from ctx, creates a
// shallow clone, and runs the function on the clone. For every other
// callable Object it falls back to Object.Call().
func CallFunc(ctx context.Context, fn Object, args ...Object) (Object, error) {
	if fn == nil {
		return nil, fmt.Errorf("cannot call nil function")
	}
	if cfn, ok := fn.(*CompiledFunction); ok {
		parentVM, ok := VMFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("no VM in context to run compiled function")
		}
		clone := parentVM.ShallowClone()
		// Register the clone with the parent so that parentVM.Abort()
		// reaches this callback too (mirrors the routineVM pattern in
		// routinevm.go).
		if _, err := parentVM.addChild(clone, nil); err != nil {
			return nil, err
		}
		defer parentVM.delChild(clone, 0)
		return clone.RunCompiled(cfn, args...)
	}
	return fn.Call(ctx, args...)
}
