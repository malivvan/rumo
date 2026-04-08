package module

import "github.com/malivvan/rumo/vm"

// WrapError converts a Go error into a Rumo error object. If the error is nil, it returns the Rumo true value.
func WrapError(err error) vm.Object {
	if err == nil {
		return vm.TrueValue
	}
	return &vm.Error{Value: &vm.String{Value: err.Error()}}
}
