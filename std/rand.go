package std

import (
	"context"
	"math/rand"

	"github.com/malivvan/vv/vm"
)

var randModule = map[string]vm.Object{
	"int": &vm.BuiltinFunction{
		Name:  "int",
		Value: FuncARI64(rand.Int63),
	},
	"float": &vm.BuiltinFunction{
		Name:  "float",
		Value: FuncARF(rand.Float64),
	},
	"intn": &vm.BuiltinFunction{
		Name:  "intn",
		Value: FuncAI64RI64(rand.Int63n),
	},
	"exp_float": &vm.BuiltinFunction{
		Name:  "exp_float",
		Value: FuncARF(rand.ExpFloat64),
	},
	"norm_float": &vm.BuiltinFunction{
		Name:  "norm_float",
		Value: FuncARF(rand.NormFloat64),
	},
	"perm": &vm.BuiltinFunction{
		Name:  "perm",
		Value: FuncAIRIs(rand.Perm),
	},
	"seed": &vm.BuiltinFunction{
		Name:  "seed",
		Value: FuncAI64R(rand.Seed),
	},
	"read": &vm.BuiltinFunction{
		Name: "read",
		Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			y1, ok := args[0].(*vm.Bytes)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{
					Name:     "first",
					Expected: "bytes",
					Found:    args[0].TypeName(),
				}
			}
			res, err := rand.Read(y1.Value)
			if err != nil {
				ret = wrapError(err)
				return
			}
			return &vm.Int{Value: int64(res)}, nil
		},
	},
	"rand": &vm.BuiltinFunction{
		Name: "rand",
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
			src := rand.NewSource(i1)
			return randRand(rand.New(src)), nil
		},
	},
}

func randRand(r *rand.Rand) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"int": &vm.BuiltinFunction{
				Name:  "int",
				Value: FuncARI64(r.Int63),
			},
			"float": &vm.BuiltinFunction{
				Name:  "float",
				Value: FuncARF(r.Float64),
			},
			"intn": &vm.BuiltinFunction{
				Name:  "intn",
				Value: FuncAI64RI64(r.Int63n),
			},
			"exp_float": &vm.BuiltinFunction{
				Name:  "exp_float",
				Value: FuncARF(r.ExpFloat64),
			},
			"norm_float": &vm.BuiltinFunction{
				Name:  "norm_float",
				Value: FuncARF(r.NormFloat64),
			},
			"perm": &vm.BuiltinFunction{
				Name:  "perm",
				Value: FuncAIRIs(r.Perm),
			},
			"seed": &vm.BuiltinFunction{
				Name:  "seed",
				Value: FuncAI64R(r.Seed),
			},
			"read": &vm.BuiltinFunction{
				Name: "read",
				Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					y1, ok := args[0].(*vm.Bytes)
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "bytes",
							Found:    args[0].TypeName(),
						}
					}
					res, err := r.Read(y1.Value)
					if err != nil {
						ret = wrapError(err)
						return
					}
					return &vm.Int{Value: int64(res)}, nil
				},
			},
		},
	}
}
