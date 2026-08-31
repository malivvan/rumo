// Package crand is the cryptographic random module: it is backed by
// crypto/rand and is safe for session tokens, secrets, IDs and anything else
// security-sensitive. The plain `rand` module is a fast deterministic PRNG
// (math/rand) and must NOT be used for security purposes.
package crand

import (
	"context"
	cryptorand "crypto/rand"
	"math/big"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

// maxInt63 is crypto/rand.Int's documented maximum (2^63 - 1).
var maxInt63 = big.NewInt(1<<63 - 1)

var Module = module.NewBuiltin().
	Func("read(b bytes) (n int, err error)	fills b with cryptographically secure random bytes", crandRead).
	Func("int() (v int)						returns a cryptographically secure random int in [0, 2^63)", crandInt).
	Func("intn(n int) (v int)					returns a cryptographically secure random int in [0, n)", crandIntn)

func crandRead(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	b, ok := args[0].(*vm.Bytes)
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bytes",
			Found:    args[0].TypeName(),
		}
	}
	res, err := cryptorand.Read(b.Value)
	if err != nil {
		return module.WrapError(err), nil
	}
	return &vm.Int{Value: int64(res)}, nil
}

func crandInt(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 0 {
		return nil, vm.ErrWrongNumArguments
	}
	n, err := cryptorand.Int(cryptorand.Reader, maxInt63)
	if err != nil {
		return module.WrapError(err), nil
	}
	return &vm.Int{Value: n.Int64()}, nil
}

func crandIntn(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	i1, ok := vm.ToInt64(args[0])
	if !ok || i1 <= 0 {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(> 0)",
			Found:    args[0].TypeName(),
		}
	}
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(i1))
	if err != nil {
		return module.WrapError(err), nil
	}
	return &vm.Int{Value: n.Int64()}, nil
}
