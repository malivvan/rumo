package rand

import (
	"context"
	"math/rand"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin("rand").
	Func("int() (v int)", rand.Int63).
	Func("float() (v float)", rand.Float64).
	Func("intn(n int) (v int)", rand.Int63n).
	Func("exp_float() (v float)", rand.ExpFloat64).
	Func("norm_float() (v float)", rand.NormFloat64).
	Func("perm(n int) (v []int)", rand.Perm).
	Func("seed(seed int)", rand.Seed).
	Func("read(b bytes) (n int, err error)", func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
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
			ret = module.WrapError(err)
			return
		}
		return &vm.Int{Value: int64(res)}, nil
	}).
	Func("rand(seed int) (rand *Rand)", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
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
	})

func randRand(r *rand.Rand) *vm.ImmutableMap {
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"int":        &vm.BuiltinFunction{Name: "int", Value: module.Func(r.Int63)},
		"float":      &vm.BuiltinFunction{Name: "float", Value: module.Func(r.Float64)},
		"intn":       &vm.BuiltinFunction{Name: "intn", Value: module.Func(r.Int63n)},
		"exp_float":  &vm.BuiltinFunction{Name: "exp_float", Value: module.Func(r.ExpFloat64)},
		"norm_float": &vm.BuiltinFunction{Name: "norm_float", Value: module.Func(r.NormFloat64)},
		"perm":       &vm.BuiltinFunction{Name: "perm", Value: module.Func(r.Perm)},
		"seed":       &vm.BuiltinFunction{Name: "seed", Value: module.Func(r.Seed)},
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
					ret = module.WrapError(err)
					return
				}
				return &vm.Int{Value: int64(res)}, nil
			},
		},
	},
	}
}
