package std

import (
	"bytes"
	"context"
	gojson "encoding/json"

	"github.com/malivvan/vv/std/json"
	"github.com/malivvan/vv/vm"
)

var jsonModule = map[string]vm.Object{
	"decode": &vm.BuiltinFunction{
		Name:  "decode",
		Value: jsonDecode,
	},
	"encode": &vm.BuiltinFunction{
		Name:  "encode",
		Value: jsonEncode,
	},
	"indent": &vm.BuiltinFunction{
		Name:  "encode",
		Value: jsonIndent,
	},
	"html_escape": &vm.BuiltinFunction{
		Name:  "html_escape",
		Value: jsonHTMLEscape,
	},
}

func jsonDecode(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}

	switch o := args[0].(type) {
	case *vm.Bytes:
		v, err := json.Decode(o.Value)
		if err != nil {
			return &vm.Error{
				Value: &vm.String{Value: err.Error()},
			}, nil
		}
		return v, nil
	case *vm.String:
		v, err := json.Decode([]byte(o.Value))
		if err != nil {
			return &vm.Error{
				Value: &vm.String{Value: err.Error()},
			}, nil
		}
		return v, nil
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bytes/string",
			Found:    args[0].TypeName(),
		}
	}
}

func jsonEncode(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}

	b, err := json.Encode(args[0])
	if err != nil {
		return &vm.Error{Value: &vm.String{Value: err.Error()}}, nil
	}

	return &vm.Bytes{Value: b}, nil
}

func jsonIndent(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 3 {
		return nil, vm.ErrWrongNumArguments
	}

	prefix, ok := vm.ToString(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "prefix",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}

	indent, ok := vm.ToString(args[2])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "indent",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
	}

	switch o := args[0].(type) {
	case *vm.Bytes:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, o.Value, prefix, indent)
		if err != nil {
			return &vm.Error{
				Value: &vm.String{Value: err.Error()},
			}, nil
		}
		return &vm.Bytes{Value: dst.Bytes()}, nil
	case *vm.String:
		var dst bytes.Buffer
		err := gojson.Indent(&dst, []byte(o.Value), prefix, indent)
		if err != nil {
			return &vm.Error{
				Value: &vm.String{Value: err.Error()},
			}, nil
		}
		return &vm.Bytes{Value: dst.Bytes()}, nil
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bytes/string",
			Found:    args[0].TypeName(),
		}
	}
}

func jsonHTMLEscape(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}

	switch o := args[0].(type) {
	case *vm.Bytes:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, o.Value)
		return &vm.Bytes{Value: dst.Bytes()}, nil
	case *vm.String:
		var dst bytes.Buffer
		gojson.HTMLEscape(&dst, []byte(o.Value))
		return &vm.Bytes{Value: dst.Bytes()}, nil
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bytes/string",
			Found:    args[0].TypeName(),
		}
	}
}
