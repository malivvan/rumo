package std

import (
	"context"
	"os"

	"github.com/malivvan/rumo/vm"
)

func makeOSFile(file *os.File) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"chdir":        &vm.BuiltinFunction{Name: "chdir", Value: FuncARE(file.Chdir)},                  // chdir() => true/error
			"chown":        &vm.BuiltinFunction{Name: "chown", Value: FuncAIIRE(file.Chown)},                // chown(uid int, gid int) => true/error
			"close":        &vm.BuiltinFunction{Name: "close", Value: FuncARE(file.Close)},                  // close() => error
			"name":         &vm.BuiltinFunction{Name: "name", Value: FuncARS(file.Name)},                    // name() => string
			"readdirnames": &vm.BuiltinFunction{Name: "readdirnames", Value: FuncAIRSsE(file.Readdirnames)}, // readdirnames(n int) => array(string)/error
			"sync":         &vm.BuiltinFunction{Name: "sync", Value: FuncARE(file.Sync)},                    // sync() => error
			"write":        &vm.BuiltinFunction{Name: "write", Value: FuncAYRIE(file.Write)},                // write(bytes) => int/error
			"write_string": &vm.BuiltinFunction{Name: "write_string", Value: FuncASRIE(file.WriteString)},   // write(string) => int/error
			"read":         &vm.BuiltinFunction{Name: "read", Value: FuncAYRIE(file.Read)},                  // read(bytes) => int/error
			// chmod(mode int) => error
			"chmod": &vm.BuiltinFunction{
				Name: "chmod",
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
					return wrapError(file.Chmod(os.FileMode(i1))), nil
				},
			},
			// seek(offset int, whence int) => int/error
			"seek": &vm.BuiltinFunction{
				Name: "seek",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 2 {
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
					i2, ok := vm.ToInt(args[1])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "second",
							Expected: "int(compatible)",
							Found:    args[1].TypeName(),
						}
					}
					res, err := file.Seek(i1, i2)
					if err != nil {
						return wrapError(err), nil
					}
					return &vm.Int{Value: res}, nil
				},
			},
			// stat() => imap(fileinfo)/error
			"stat": &vm.BuiltinFunction{
				Name: "stat",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 0 {
						return nil, vm.ErrWrongNumArguments
					}
					return osStat(ctx, &vm.String{Value: file.Name()})
				},
			},
		},
	}
}
