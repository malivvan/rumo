package hex

import (
	"encoding/hex"

	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Func("encode(b bytes) (s string)		returns the hexadecimal encoding of src", hex.EncodeToString).
	Func("decode(s string) (b bytes)		returns the bytes represented by the hexadecimal string s", hex.DecodeString)
