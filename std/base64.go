package std

import (
	"encoding/base64"

	"github.com/malivvan/rumo/vm"
)

var base64Module = map[string]vm.Object{
	"encode":         &vm.BuiltinFunction{Value: FuncAYRS(base64.StdEncoding.EncodeToString)},    // encode(bytes) => string/error
	"decode":         &vm.BuiltinFunction{Value: FuncASRYE(base64.StdEncoding.DecodeString)},     // decode(string) => bytes/error
	"raw_encode":     &vm.BuiltinFunction{Value: FuncAYRS(base64.RawStdEncoding.EncodeToString)}, // raw_encode(bytes) => string/error
	"raw_decode":     &vm.BuiltinFunction{Value: FuncASRYE(base64.RawStdEncoding.DecodeString)},  // raw_decode(string) => bytes/error
	"url_encode":     &vm.BuiltinFunction{Value: FuncAYRS(base64.URLEncoding.EncodeToString)},    // url_encode(bytes) => string/error
	"url_decode":     &vm.BuiltinFunction{Value: FuncASRYE(base64.URLEncoding.DecodeString)},     // url_decode(string) => bytes/error
	"raw_url_encode": &vm.BuiltinFunction{Value: FuncAYRS(base64.RawURLEncoding.EncodeToString)}, // raw_url_encode(bytes) => string/error
	"raw_url_decode": &vm.BuiltinFunction{Value: FuncASRYE(base64.RawURLEncoding.DecodeString)},  // raw_url_decode(string) => bytes/error
}
