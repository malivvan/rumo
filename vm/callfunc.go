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
		if vmVal := ctx.Value(ContextKey("vm")); vmVal != nil {
			parentVM := vmVal.(*VM)
			clone := parentVM.ShallowClone()
			return clone.RunCompiled(cfn, args...)
		}
		return nil, fmt.Errorf("no VM in context to run compiled function")
	}
	return fn.Call(ctx, args...)
}
