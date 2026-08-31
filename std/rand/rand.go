// Package rand exposes Go's math/rand pseudo-random generator. The output is
// deterministic and fast, but it is NOT cryptographically secure: never use
// it for session tokens, secrets, keys or anything security-sensitive. Use
// the crand module (crypto/rand) for those purposes.
package rand

import (
	"context"
	"math/rand"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Func("int() (v int)						non-cryptographic random int in [0, 2^63); use crand for secrets", rand.Int63).
	Func("float() (v float)					non-cryptographic random float in [0.0, 1.0)", rand.Float64).
	Func("intn(n int) (v int)					non-cryptographic random int in [0, n); use crand for secrets", rand.Int63n).
	Func("exp_float() (v float)				non-cryptographic exponentially distributed float", rand.ExpFloat64).
	Func("norm_float() (v float)				non-cryptographic normally distributed float", rand.NormFloat64).
	Func("perm(n int) (v []int)				non-cryptographic random permutation of [0, n)", rand.Perm).
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
		res, err := rand.Read(y1.Value) //nolint:staticcheck // deprecated but acceptable here
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

func randRand(r *rand.Rand) *vm.Map {
	return &vm.Map{Frozen: true, Value: map[string]vm.Object{
		"int":        &vm.BuiltinFunction{Name: "int", Value: module.Func(r.Int63)},
		"float":      &vm.BuiltinFunction{Name: "float", Value: module.Func(r.Float64)},
		"intn":       &vm.BuiltinFunction{Name: "intn", Value: module.Func(r.Int63n)},
		"exp_float":  &vm.BuiltinFunction{Name: "exp_float", Value: module.Func(r.ExpFloat64)},
		"norm_float": &vm.BuiltinFunction{Name: "norm_float", Value: module.Func(r.NormFloat64)},
		"perm":       &vm.BuiltinFunction{Name: "perm", Value: module.Func(r.Perm)},
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
