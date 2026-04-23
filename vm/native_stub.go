//go:build !native

package vm

import (
	"context"
	"fmt"
)

// Native is a stub type used in non-native builds so that the bytecode
// deduplication switch in bytecode.go can still compile.  It can never be
// instantiated at runtime because compileNative (compiler_native_stub.go)
// always returns an error before creating one.
type Native struct {
	ObjectImpl
}

// TypeName returns the name of the type.
func (o *Native) TypeName() string { return "native-loader" }

// String returns a string representation.
func (o *Native) String() string { return "<native-loader (disabled)>" }

// CanCall returns false; the stub is never callable.
func (o *Native) CanCall() bool { return false }

// Call always returns an error; native support is not compiled in.
func (o *Native) Call(_ context.Context, _ ...Object) (Object, error) {
	return nil, fmt.Errorf("native: not supported (rebuild with -tags native)")
}

