package std

import (
	"context"
	"os"

	"github.com/malivvan/rumo/vm"
)

func makeOSFile(file *os.File) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			// chdir() => true/error
			"chdir": &vm.BuiltinFunction{
				Name:  "chdir",
				Value: FuncARE(file.Chdir),
			}, //
			// chown(uid int, gid int) => true/error
			//"chown": &vm.BuiltinFunction{
			//	Name:  "chown",
			//	Value: FuncAIIRE(file.Chown),
			//}, //
			// close() => error
			"close": &vm.BuiltinFunction{
				Name:  "close",
				Value: FuncARE(file.Close),
			}, //
			// name() => string
			"name": &vm.BuiltinFunction{
				Name:  "name",
				Value: FuncARS(file.Name),
			}, //
			// readdirnames(n int) => array(string)/error
			"readdirnames": &vm.BuiltinFunction{
				Name:  "readdirnames",
				Value: FuncAIRSsE(file.Readdirnames),
			}, //
			// sync() => error
			"sync": &vm.BuiltinFunction{
				Name:  "sync",
				Value: FuncARE(file.Sync),
			}, //
			// write(bytes) => int/error
			"write": &vm.BuiltinFunction{
				Name:  "write",
				Value: FuncAYRIE(file.Write),
			}, //
			// write(string) => int/error
			"write_string": &vm.BuiltinFunction{
				Name:  "write_string",
				Value: FuncASRIE(file.WriteString),
			}, //
			// read(bytes) => int/error
			"read": &vm.BuiltinFunction{
				Name:  "read",
				Value: FuncAYRIE(file.Read),
			}, //
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
