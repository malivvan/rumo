package std

import (
	"context"
	"os"
	"syscall"

	"github.com/malivvan/vv/vm"
)

func makeOSProcessState(state *os.ProcessState) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"exited": &vm.BuiltinFunction{
				Name:  "exited",
				Value: FuncARB(state.Exited),
			},
			//"pid": &vm.BuiltinFunction{
			//	Name:  "pid",
			//	Value: FuncARI(state.Pid),
			//},
			"string": &vm.BuiltinFunction{
				Name:  "string",
				Value: FuncARS(state.String),
			},
			"success": &vm.BuiltinFunction{
				Name:  "success",
				Value: FuncARB(state.Success),
			},
		},
	}
}

func makeOSProcess(proc *os.Process) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"kill": &vm.BuiltinFunction{
				Name:  "kill",
				Value: FuncARE(proc.Kill),
			},
			"release": &vm.BuiltinFunction{
				Name:  "release",
				Value: FuncARE(proc.Release),
			},
			"signal": &vm.BuiltinFunction{
				Name: "signal",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					i1, ok := vm.ToInt64(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					return wrapError(proc.Signal(syscall.Signal(i1))), nil
				},
			},
			"wait": &vm.BuiltinFunction{
				Name: "wait",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 0 {
						return nil, vm.ErrWrongNumArguments
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
