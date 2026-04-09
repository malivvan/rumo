package base64

import (
	"encoding/base64"

	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Func("encode(b bytes) (s string)			returns the base64 encoding of src", base64.StdEncoding.EncodeToString).
	Func("decode(s string) (b bytes)			returns the bytes represented by the base64 string s", base64.StdEncoding.DecodeString).
	Func("raw_encode(b bytes) (s string)		returns the unpadded base64 encoding of src", base64.RawStdEncoding.EncodeToString).
	Func("raw_decode(s string) (b bytes)		returns the bytes represented by the unpadded base64 string s", base64.RawStdEncoding.DecodeString).
	Func("url_encode(b bytes) (s string)		returns the base64 encoding of src using URL and Filename safe alphabet", base64.URLEncoding.EncodeToString).
	Func("url_decode(s string) (b bytes)		returns the bytes represented by the base64 URL and Filename safe string s", base64.URLEncoding.DecodeString).
	Func("raw_url_encode(b bytes) (s string)	returns the unpadded base64 encoding of src using URL and Filename safe alphabet", base64.RawURLEncoding.EncodeToString).
	Func("raw_url_decode(s string) (b bytes)	returns the bytes represented by the unpadded base64 URL and Filename safe string s", base64.RawURLEncoding.DecodeString)
