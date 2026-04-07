package std

import (
	"context"
	"os/exec"

	"github.com/malivvan/rumo/vm"
)

func makeOSExecCommand(cmd *exec.Cmd) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			// combined_output() => bytes/error
			"combined_output": &vm.BuiltinFunction{
				Name:  "combined_output",
				Value: FuncARYE(cmd.CombinedOutput),
			},
			// output() => bytes/error
			"output": &vm.BuiltinFunction{
				Name:  "output",
				Value: FuncARYE(cmd.Output),
			}, //
			// run() => error
			"run": &vm.BuiltinFunction{
				Name:  "run",
				Value: FuncARE(cmd.Run),
			}, //
			// start() => error
			"start": &vm.BuiltinFunction{
				Name:  "start",
				Value: FuncARE(cmd.Start),
			}, //
			// wait() => error
			"wait": &vm.BuiltinFunction{
				Name:  "wait",
				Value: FuncARE(cmd.Wait),
			}, //
			// set_path(path string)
			"set_path": &vm.BuiltinFunction{
				Name: "set_path",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					s1, ok := vm.ToString(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					cmd.Path = s1
					return vm.UndefinedValue, nil
				},
			},
			// set_dir(dir string)
			"set_dir": &vm.BuiltinFunction{
				Name: "set_dir",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					s1, ok := vm.ToString(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					cmd.Dir = s1
					return vm.UndefinedValue, nil
				},
			},
			// set_env(env array(string))
			"set_env": &vm.BuiltinFunction{
				Name: "set_env",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}

					var env []string
					var err error
					switch arg0 := args[0].(type) {
					case *vm.Array:
						env, err = stringArray(arg0.Value, "first")
						if err != nil {
							return nil, err
						}
					case *vm.ImmutableArray:
						env, err = stringArray(arg0.Value, "first")
						if err != nil {
							return nil, err
						}
					default:
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "array",
							Found:    arg0.TypeName(),
						}
					}
					cmd.Env = env
					return vm.UndefinedValue, nil
				},
			},
			// process() => imap(process)
			"process": &vm.BuiltinFunction{
				Name: "process",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 0 {
						return nil, vm.ErrWrongNumArguments
					}
					if cmd.Process == nil {
						return vm.UndefinedValue, nil
					}
					return makeOSProcess(cmd.Process), nil
				},
			},
		},
	}
}
