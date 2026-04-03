package std

import (
	"encoding/hex"
	"github.com/malivvan/vv/vm"
)

var hexModule = map[string]vm.Object{
	"encode": &vm.BuiltinFunction{Value: FuncAYRS(hex.EncodeToString)},
	"decode": &vm.BuiltinFunction{Value: FuncASRYE(hex.DecodeString)},
}
