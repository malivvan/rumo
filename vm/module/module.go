package module

import (
	"github.com/malivvan/rumo/vm"
)

func wrapError(err error) vm.Object {
	if err == nil {
		return vm.TrueValue
	}
	return &vm.Error{Value: &vm.String{Value: err.Error()}}
}
