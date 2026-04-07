package vm

import (
	"errors"
	"github.com/malivvan/rumo/vm/encoding"
	"github.com/malivvan/rumo/vm/parser"
	"time"
)

const (
	_undefined        byte = 1
	_bool             byte = 2
	_bytes            byte = 3
	_char             byte = 4
	_int              byte = 5
	_float            byte = 6
	_string           byte = 7
	_time             byte = 8
	_array            byte = 9
	_map              byte = 10
	_immutableArray   byte = 11
	_immutableMap     byte = 12
	_objectPtr        byte = 13
	_compiledFunction byte = 14
	_error            byte = 15
	_arrayIterator    byte = 100
	_mapIterator      byte = 101
	_stringIterator   byte = 102
	_bytesIterator    byte = 103
	_builtinFunction  byte = 104
)

var _typeMap = map[byte]func() Object{
	_undefined:        func() Object { return &Undefined{} },
	_bool:             func() Object { return &Bool{} },
	_bytes:            func() Object { return &Bytes{} },
	_char:             func() Object { return &Char{} },
	_int:              func() Object { return &Int{} },
	_float:            func() Object { return &Float{} },
	_string:           func() Object { return &String{} },
	_time:             func() Object { return &Time{} },
	_array:            func() Object { return &Array{} },
	_map:              func() Object { return &Map{Value: make(map[string]Object)} },
	_immutableArray:   func() Object { return &ImmutableArray{} },
	_immutableMap:     func() Object { return &ImmutableMap{Value: make(map[string]Object)} },
	_objectPtr:        func() Object { return &ObjectPtr{} },
	_compiledFunction: func() Object { return &CompiledFunction{SourceMap: make(map[int]parser.Pos)} },
	_builtinFunction:  func() Object { return &BuiltinFunction{} },
	_error:            func() Object { return &Error{} },
}

// MakeObject creates a new object based on the given type code.
func MakeObject(code byte) Object {
	fn, exists := _typeMap[code]
	if !exists {
		panic("unknown type code: " + string(code))
	}
	return fn()
}

// SizeOfObjectPtr returns the size of the given object pointer.
func SizeOfObjectPtr(op *ObjectPtr) int {
	return SizeOfObject(*op.Value)
}

// TypeOfObject returns the type code of the given object.
func TypeOfObject(o Object) byte {
	switch o.(type) {
	case *Undefined:
		return _undefined
	case *Bool:
		return _bool
	case *Bytes:
		return _bytes
	case *Char:
		return _char
	case *Int:
		return _int
	case *Float:
		return _float
	case *String:
		return _string
	case *Time:
		return _time
	case *Array:
		return _array
	case *Map:
		return _map
	case *ImmutableArray:
		return _immutableArray
	case *ImmutableMap:
		return _immutableMap
	case *ObjectPtr:
		return _objectPtr
	case *CompiledFunction:
		return _compiledFunction
	case *BuiltinFunction:
		return _builtinFunction
	case *Error:
		return _error
	default:
		return 0
	}
}

// SizeOfObject returns the size of the given object.
func SizeOfObject(o Object) int {
	if o == nil {
		return encoding.SizeByte()
	}
	switch TypeOfObject(o) {
	case _undefined:
		return encoding.SizeByte()
	case _bool:
		return encoding.SizeByte() + encoding.SizeBool()
	case _bytes:
		return encoding.SizeByte() + encoding.SizeBytes(o.(*Bytes).Value)
	case _char:
		return encoding.SizeByte() + encoding.SizeInt32()
	case _int:
		return encoding.SizeByte() + encoding.SizeInt64()
	case _float:
		return encoding.SizeByte() + encoding.SizeFloat64()
	case _string:
		return encoding.SizeByte() + encoding.SizeString(o.(*String).Value)
	case _time:
		return encoding.SizeByte() + encoding.SizeInt64()
	case _array:
		return encoding.SizeByte() + encoding.SizeSlice(o.(*Array).Value, SizeOfObject)
	case _map:
		return encoding.SizeByte() + encoding.SizeMap(o.(*Map).Value, encoding.SizeString, SizeOfObject)
	case _immutableArray:
		return encoding.SizeByte() + encoding.SizeSlice(o.(*ImmutableArray).Value, SizeOfObject)
	case _immutableMap:
		return encoding.SizeByte() + encoding.SizeMap(o.(*ImmutableMap).Value, encoding.SizeString, SizeOfObject)
	case _objectPtr:
		v := o.(*ObjectPtr)
		if v.Value != nil {
			return encoding.SizeByte() + SizeOfObject(*v.Value)
		}
		return encoding.SizeByte() + SizeOfObject(nil)
	case _compiledFunction:
		s := encoding.SizeBytes(o.(*CompiledFunction).Instructions)
		s += encoding.SizeInt(o.(*CompiledFunction).NumLocals)
		s += encoding.SizeInt(o.(*CompiledFunction).NumParameters)
		s += encoding.SizeBool()
		s += encoding.SizeMap(o.(*CompiledFunction).SourceMap, encoding.SizeInt, parser.SizePos)
		s += encoding.SizeSlice[*ObjectPtr](o.(*CompiledFunction).Free, SizeOfObjectPtr)
		return encoding.SizeByte() + s
	case _builtinFunction:
		s := encoding.SizeString(o.(*BuiltinFunction).Name)
		return encoding.SizeByte() + s
	case _error:
		return encoding.SizeByte() + SizeOfObject(o.(*Error).Value)
	default:
		panic("sizeof: unsupported type: " + o.TypeName())
	}
}

// MarshalObject marshals the given object into a byte slice.
func MarshalObject(n int, b []byte, o Object) int {
	if o == nil {
		return encoding.MarshalByte(n, b, 0)
	}
	switch TypeOfObject(o) {
	case _undefined:
		n = encoding.MarshalByte(n, b, _undefined)
	case _bool:
		n = encoding.MarshalByte(n, b, _bool)
		n = encoding.MarshalBool(n, b, o.(*Bool).value)
	case _bytes:
		n = encoding.MarshalByte(n, b, _bytes)
		n = encoding.MarshalBytes(n, b, o.(*Bytes).Value)
	case _char:
		n = encoding.MarshalByte(n, b, _char)
		n = encoding.MarshalInt32(n, b, o.(*Char).Value)
	case _int:
		n = encoding.MarshalByte(n, b, _int)
		n = encoding.MarshalInt64(n, b, o.(*Int).Value)
	case _float:
		n = encoding.MarshalByte(n, b, _float)
		n = encoding.MarshalFloat64(n, b, o.(*Float).Value)
	case _string:
		n = encoding.MarshalByte(n, b, _string)
		n = encoding.MarshalString(n, b, o.(*String).Value)
	case _time:
		n = encoding.MarshalByte(n, b, _time)
		n = encoding.MarshalInt64(n, b, o.(*Time).Value.UnixNano())
	case _array:
		n = encoding.MarshalByte(n, b, _array)
		n = encoding.MarshalSlice(n, b, o.(*Array).Value, MarshalObject)
	case _map:
		n = encoding.MarshalByte(n, b, _map)
		n = encoding.MarshalMap(n, b, o.(*Map).Value, encoding.MarshalString, MarshalObject)
	case _immutableArray:
		n = encoding.MarshalByte(n, b, _immutableArray)
		n = encoding.MarshalSlice(n, b, o.(*ImmutableArray).Value, MarshalObject)
	case _immutableMap:
		n = encoding.MarshalByte(n, b, _immutableMap)
		n = encoding.MarshalMap(n, b, o.(*ImmutableMap).Value, encoding.MarshalString, MarshalObject)
	case _objectPtr:
		n = encoding.MarshalByte(n, b, _objectPtr)
		if o.(*ObjectPtr).Value != nil {
			n = MarshalObject(n, b, *o.(*ObjectPtr).Value)
		} else {
			n = MarshalObject(n, b, nil)
		}
	case _compiledFunction:
		v := o.(*CompiledFunction)
		n = encoding.MarshalByte(n, b, _compiledFunction)
		n = encoding.MarshalBytes(n, b, v.Instructions)
		n = encoding.MarshalInt(n, b, v.NumLocals)
		n = encoding.MarshalInt(n, b, v.NumParameters)
		n = encoding.MarshalBool(n, b, v.VarArgs)
		n = encoding.MarshalMap(n, b, v.SourceMap, encoding.MarshalInt, parser.MarshalPos)
		n = encoding.MarshalSlice(n, b, v.Free, func(n int, b []byte, o *ObjectPtr) int { return MarshalObject(n, b, o) })
	case _builtinFunction:
		n = encoding.MarshalByte(n, b, _builtinFunction)
		n = encoding.MarshalString(n, b, o.(*BuiltinFunction).Name)
	case _error:
		n = encoding.MarshalByte(n, b, _error)
		n = MarshalObject(n, b, o.(*Error).Value)
	default:
		panic("marshal: unsupported type: " + o.TypeName())
	}
	return n
}

// UnmarshalObject unmarshals the given byte slice into an object.
func UnmarshalObject(nn int, b []byte) (n int, o Object, err error) {
	if b[nn] == 0 {
		return nn + 1, nil, nil
	}
	var t byte
	n, t, err = encoding.UnmarshalByte(nn, b)
	if err != nil {
		return nn, nil, err
	}
	o = MakeObject(t)
	switch t {
	case _undefined:
		return n, o, nil
	case _bool:
		n, o.(*Bool).value, err = encoding.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _bytes:
		n, o.(*Bytes).Value, err = encoding.UnmarshalBytes(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _char:
		n, o.(*Char).Value, err = encoding.UnmarshalInt32(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _int:
		n, o.(*Int).Value, err = encoding.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _float:
		n, o.(*Float).Value, err = encoding.UnmarshalFloat64(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _string:
		n, o.(*String).Value, err = encoding.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _time:
		var v int64
		n, v, err = encoding.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		o.(*Time).Value = time.Unix(0, v).In(time.UTC)
		return n, o, nil
	case _array:
		n, o.(*Array).Value, err = encoding.UnmarshalSlice[Object](n, b, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _map:
		n, o.(*Map).Value, err = encoding.UnmarshalMap[string, Object](n, b, encoding.UnmarshalString, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _immutableArray:
		n, o.(*ImmutableArray).Value, err = encoding.UnmarshalSlice[Object](n, b, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _immutableMap:
		n, o.(*ImmutableMap).Value, err = encoding.UnmarshalMap[string, Object](n, b, encoding.UnmarshalString, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _objectPtr:
		var v Object
		n, v, err = UnmarshalObject(n, b)
		if err != nil {
			return nn, nil, err
		}
		if _, isUndefined := v.(*Undefined); !isUndefined {
			o.(*ObjectPtr).Value = &v
		}
		return n, o, nil
	case _compiledFunction:
		n, o.(*CompiledFunction).Instructions, err = encoding.UnmarshalBytes(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).NumLocals, err = encoding.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).NumParameters, err = encoding.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).VarArgs, err = encoding.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).SourceMap, err = encoding.UnmarshalMap[int, parser.Pos](n, b, encoding.UnmarshalInt, parser.UnmarshalPos)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).Free, err = encoding.UnmarshalSlice[*ObjectPtr](n, b, func(nn int, b []byte) (n int, o *ObjectPtr, err error) {
			var v Object
			n, v, err = UnmarshalObject(nn, b)
			if err != nil {
				return nn, nil, err
			}
			var ok bool
			o, ok = v.(*ObjectPtr)
			if !ok {
				return nn, nil, errors.New("expected *ObjectPtr")
			}
			return n, o, nil
		})
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil

	case _builtinFunction:
		n, o.(*BuiltinFunction).Name, err = encoding.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _error:
		n, o.(*Error).Value, err = UnmarshalObject(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	}
	return nn, nil, errors.New("unmarshal: unsupported type: " + o.TypeName())
}
