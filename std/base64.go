package std

import (
	"encoding/base64"

	"github.com/malivvan/vv/vm"
)

var base64Module = map[string]vm.Object{
	"encode": &vm.BuiltinFunction{
		Value: FuncAYRS(base64.StdEncoding.EncodeToString),
	},
	"decode": &vm.BuiltinFunction{
		Value: FuncASRYE(base64.StdEncoding.DecodeString),
	},
	"raw_encode": &vm.BuiltinFunction{
		Value: FuncAYRS(base64.RawStdEncoding.EncodeToString),
	},
	"raw_decode": &vm.BuiltinFunction{
		Value: FuncASRYE(base64.RawStdEncoding.DecodeString),
	},
	"url_encode": &vm.BuiltinFunction{
		Value: FuncAYRS(base64.URLEncoding.EncodeToString),
	},
	"url_decode": &vm.BuiltinFunction{
		Value: FuncASRYE(base64.URLEncoding.DecodeString),
	},
	"raw_url_encode": &vm.BuiltinFunction{
		Value: FuncAYRS(base64.RawURLEncoding.EncodeToString),
	},
	"raw_url_decode": &vm.BuiltinFunction{
		Value: FuncASRYE(base64.RawURLEncoding.DecodeString),
	},
}
