//go:build !native

package vm

import (
	"context"
	"fmt"
)

// NativeSupported reports whether the current build includes FFI (native)
// support.  It returns false in non-native builds and true when compiled with
// -tags native.
func NativeSupported() bool { return false }

// ResolveNativePath always returns an empty string in non-native builds.
// Native FFI is not compiled in, so no library path can be resolved.
func ResolveNativePath(_ string) string { return "" }

// Native is a stub type used in non-native builds so that the bytecode
// deduplication switch in bytecode.go can still compile.  It can never be
// instantiated at runtime because compileNative (compiler_native_stub.go)
// always returns an error before creating one.  The Path and Funcs fields are
// populated during bytecode decoding so that Stat() can still inspect what
// native libraries a bytecode file would require.
type Native struct {
	ObjectImpl
	Path  string
	Funcs []NativeFuncSpec
}

// TypeName returns the name of the type.
func (o *Native) TypeName() string { return "native-loader" }

// String returns a string representation.
func (o *Native) String() string { return "<native-loader (disabled)>" }

// NativePath returns the shared library path embedded in this loader constant.
func (o *Native) NativePath() string { return o.Path }

// CanCall returns false; the stub is never callable.
func (o *Native) CanCall() bool { return false }

// Call always returns an error; native support is not compiled in.
func (o *Native) Call(_ context.Context, _ ...Object) (Object, error) {
	return nil, fmt.Errorf("native: not supported (rebuild with -tags native)")
}
