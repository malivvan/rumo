package stdlib

import (
	"context"
	"os"

	"github.com/malivvan/vv/vvm"
)

func makeOSFile(file *os.File) *vvm.ImmutableMap {
	return &vvm.ImmutableMap{
		Value: map[string]vvm.Object{
			// chdir() => true/error
			"chdir": &vvm.BuiltinFunction{
				Name:  "chdir",
				Value: FuncARE(file.Chdir),
			}, //
			// chown(uid int, gid int) => true/error
			//"chown": &vvm.BuiltinFunction{
			//	Name:  "chown",
			//	Value: FuncAIIRE(file.Chown),
			//}, //
			// close() => error
			"close": &vvm.BuiltinFunction{
				Name:  "close",
				Value: FuncARE(file.Close),
			}, //
			// name() => string
			"name": &vvm.BuiltinFunction{
				Name:  "name",
				Value: FuncARS(file.Name),
			}, //
			// readdirnames(n int) => array(string)/error
			"readdirnames": &vvm.BuiltinFunction{
				Name:  "readdirnames",
				Value: FuncAIRSsE(file.Readdirnames),
			}, //
			// sync() => error
			"sync": &vvm.BuiltinFunction{
				Name:  "sync",
				Value: FuncARE(file.Sync),
			}, //
			// write(bytes) => int/error
			"write": &vvm.BuiltinFunction{
				Name:  "write",
				Value: FuncAYRIE(file.Write),
			}, //
			// write(string) => int/error
			"write_string": &vvm.BuiltinFunction{
				Name:  "write_string",
				Value: FuncASRIE(file.WriteString),
			}, //
			// read(bytes) => int/error
			"read": &vvm.BuiltinFunction{
				Name:  "read",
				Value: FuncAYRIE(file.Read),
			}, //
			// chmod(mode int) => error
			"chmod": &vvm.BuiltinFunction{
				Name: "chmod",
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
					return wrapError(file.Chmod(os.FileMode(i1))), nil
				},
			},
			// seek(offset int, whence int) => int/error
			"seek": &vvm.BuiltinFunction{
				Name: "seek",
				Value: func(ctx context.Context, args ...vvm.Object) (vvm.Object, error) {
					if len(args) != 2 {
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
					i2, ok := vvm.ToInt(args[1])
					if !ok {
						return nil, vvm.ErrInvalidArgumentType{
							Name:     "second",
							Expected: "int(compatible)",
							Found:    args[1].TypeName(),
						}
					}
					res, err := file.Seek(i1, i2)
					if err != nil {
						return wrapError(err), nil
					}
					return &vvm.Int{Value: res}, nil
				},
			},
			// stat() => imap(fileinfo)/error
			"stat": &vvm.BuiltinFunction{
				Name: "stat",
				Value: func(ctx context.Context, args ...vvm.Object) (vvm.Object, error) {
					if len(args) != 0 {
						return nil, vvm.ErrWrongNumArguments
					}
					return osStat(ctx, &vvm.String{Value: file.Name()})
				},
			},
		},
	}
}
