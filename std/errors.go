package std

import (
	"github.com/malivvan/vv/vm"
)

func wrapError(err error) vm.Object {
	if err == nil {
		return vm.TrueValue
	}
	return &vm.Error{Value: &vm.String{Value: err.Error()}}
}
