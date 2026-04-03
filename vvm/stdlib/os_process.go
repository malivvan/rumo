package stdlib

import (
	"context"
	"os"
	"syscall"

	"github.com/malivvan/vv/vvm"
)

func makeOSProcessState(state *os.ProcessState) *vvm.ImmutableMap {
	return &vvm.ImmutableMap{
		Value: map[string]vvm.Object{
			"exited": &vvm.BuiltinFunction{
				Name:  "exited",
				Value: FuncARB(state.Exited),
			},
			//"pid": &vvm.BuiltinFunction{
			//	Name:  "pid",
			//	Value: FuncARI(state.Pid),
			//},
			"string": &vvm.BuiltinFunction{
				Name:  "string",
				Value: FuncARS(state.String),
			},
			"success": &vvm.BuiltinFunction{
				Name:  "success",
				Value: FuncARB(state.Success),
			},
		},
	}
}

func makeOSProcess(proc *os.Process) *vvm.ImmutableMap {
	return &vvm.ImmutableMap{
		Value: map[string]vvm.Object{
			"kill": &vvm.BuiltinFunction{
				Name:  "kill",
				Value: FuncARE(proc.Kill),
			},
			"release": &vvm.BuiltinFunction{
				Name:  "release",
				Value: FuncARE(proc.Release),
			},
			"signal": &vvm.BuiltinFunction{
				Name: "signal",
				Value: func(ctx context.Context, args ...vvm.Object) (vvm.Object, error) {
					if len(args) != 1 {
						return nil, vvm.ErrWrongNumArguments
					}
					i1, ok := vvm.ToInt64(args[0])
					if !ok {
						return nil, vvm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					return wrapError(proc.Signal(syscall.Signal(i1))), nil
				},
			},
			"wait": &vvm.BuiltinFunction{
				Name: "wait",
				Value: func(ctx context.Context, args ...vvm.Object) (vvm.Object, error) {
					if len(args) != 0 {
						return nil, vvm.ErrWrongNumArguments
					}
					state, err := proc.Wait()
					if err != nil {
						return wrapError(err), nil
					}
					return makeOSProcessState(state), nil
				},
			},
		},
	}
}
