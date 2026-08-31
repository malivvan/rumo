package module

import (
	"github.com/malivvan/rumo/vm"
)

// wrapError delegates to the exported WrapError so there is exactly one
// error-wrapping implementation. See WrapError for the nil-error convention.
func wrapError(err error) vm.Object {
	return WrapError(err)
}
