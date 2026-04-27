package vm

import (
	"errors"
	"fmt"
	"time"

	"github.com/malivvan/rumo/vm/codec"
	"github.com/malivvan/rumo/vm/parser"
)

const (
	_undefined        byte = 1
	_bool             byte = 2
	_bytes            byte = 3
	_char             byte = 4
	_int              byte = 5
	_float            byte = 6  // legacy alias for _float64
	_float64          byte = 6  // float64 / C double
	_float32          byte = 16 // float32 / C float
	_string           byte = 7
	_time             byte = 8
	_array            byte = 9
	_map              byte = 10
	_immutableArray   byte = 11
	_immutableMap     byte = 12
	_objectPtr        byte = 13
	_compiledFunction byte = 14
	_error            byte = 15
	_byte             byte = 17
	_int8             byte = 18
	_uint8            byte = 19
	_int16            byte = 20
	_uint16           byte = 21
	_uint             byte = 22
	_uint64           byte = 23
	_ptr              byte = 24
	_arrayIterator    byte = 100
	_mapIterator      byte = 101
	_stringIterator   byte = 102
	_bytesIterator    byte = 103
	_builtinFunction  byte = 104
	_rangeObject      byte = 105
)

var _typeMap = map[byte]func() Object{
	_undefined:        func() Object { return &Undefined{} },
	_bool:             func() Object { return &Bool{} },
	_bytes:            func() Object { return &Bytes{} },
	_char:             func() Object { return &Char{} },
	_int:              func() Object { return &Int{} },
	_int8:             func() Object { return &Int8{} },
	_int16:            func() Object { return &Int16{} },
	_byte:             func() Object { return &Byte{} },
	_uint8:            func() Object { return &Uint8{} },
	_uint16:           func() Object { return &Uint16{} },
	_uint:             func() Object { return &Uint{} },
	_uint64:           func() Object { return &Uint64{} },
	_ptr:              func() Object { return &Ptr{} },
	_float:            func() Object { return &Float64{} },
	_float32:          func() Object { return &Float32{} },
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
	_rangeObject:      func() Object { return &RangeObject{} },
}

// MakeObject creates a new object based on the given type code.
// Returns nil if the type code is not recognised; callers should check
// for nil and convert it to an error rather than letting it propagate.
func MakeObject(code byte) Object {
	fn, exists := _typeMap[code]
	if !exists {
		return nil
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
	case *Int8:
		return _int8
	case *Int16:
		return _int16
	case *Byte:
		return _byte
	case *Uint8:
		return _uint8
	case *Uint16:
		return _uint16
	case *Uint:
		return _uint
	case *Uint64:
		return _uint64
	case *Ptr:
		return _ptr
	case *Float64:
		return _float64
	case *Float32:
		return _float32
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
	case *RangeObject:
		return _rangeObject
	default:
		return 0
	}
}

// SizeOfObject returns the size of the given object.
func SizeOfObject(o Object) int {
	if o == nil {
		return codec.SizeByte()
	}
	switch TypeOfObject(o) {
	case _undefined:
		return codec.SizeByte()
	case _bool:
		return codec.SizeByte() + codec.SizeBool()
	case _bytes:
		return codec.SizeByte() + codec.SizeBytes(o.(*Bytes).Value)
	case _char:
		return codec.SizeByte() + codec.SizeInt32()
	case _int:
		return codec.SizeByte() + codec.SizeInt64()
	case _int8:
		return codec.SizeByte() + codec.SizeByte()
	case _int16:
		return codec.SizeByte() + codec.SizeInt16()
	case _byte, _uint8:
		return codec.SizeByte() + codec.SizeByte()
	case _uint16:
		return codec.SizeByte() + codec.SizeUint16()
	case _uint:
		return codec.SizeByte() + codec.SizeUint32()
	case _uint64:
		return codec.SizeByte() + codec.SizeUint64()
	case _ptr:
		// Ptr values must never be serialised; refuse at size-computation time.
		panic("sizeof: Ptr cannot be serialised into bytecode")
	case _float64:
		return codec.SizeByte() + codec.SizeFloat64()
	case _float32:
		return codec.SizeByte() + codec.SizeFloat32()
	case _string:
		return codec.SizeByte() + codec.SizeString(o.(*String).Value)
	case _time:
		return codec.SizeByte() + codec.SizeInt64()
	case _array:
		arr := o.(*Array)
		arr.mu.RLock()
		snap := append([]Object(nil), arr.Value...)
		arr.mu.RUnlock()
		return codec.SizeByte() + codec.SizeSlice(snap, SizeOfObject)
	case _map:
		m := o.(*Map)
		m.mu.RLock()
		snapM := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			snapM[k] = v
		}
		m.mu.RUnlock()
		return codec.SizeByte() + codec.SizeMap(snapM, codec.SizeString, SizeOfObject)
	case _immutableArray:
		return codec.SizeByte() + codec.SizeSlice(o.(*ImmutableArray).Value, SizeOfObject)
	case _immutableMap:
		im := o.(*ImmutableMap)
		return codec.SizeByte() + codec.SizeString(im.moduleName) + codec.SizeMap(im.Value, codec.SizeString, SizeOfObject)
	case _objectPtr:
		v := o.(*ObjectPtr)
		if v.Value != nil {
			return codec.SizeByte() + SizeOfObject(*v.Value)
		}
		return codec.SizeByte() + SizeOfObject(nil)
	case _compiledFunction:
		s := codec.SizeBytes(o.(*CompiledFunction).Instructions)
		s += codec.SizeInt(o.(*CompiledFunction).NumLocals)
		s += codec.SizeInt(o.(*CompiledFunction).NumParameters)
		s += codec.SizeBool()
		s += codec.SizeMap(o.(*CompiledFunction).SourceMap, codec.SizeInt, parser.SizePos)
		s += codec.SizeSlice[*ObjectPtr](o.(*CompiledFunction).Free, SizeOfObjectPtr)
		return codec.SizeByte() + s
	case _builtinFunction:
		s := codec.SizeString(o.(*BuiltinFunction).Name)
		return codec.SizeByte() + s
	case _error:
		return codec.SizeByte() + SizeOfObject(o.(*Error).Value)
	case _rangeObject:
		return codec.SizeByte() + codec.SizeInt64()*3
	default:
		panic("sizeof: unsupported type: " + o.TypeName())
	}
}

// MarshalObject marshals the given object into a byte slice.
func MarshalObject(n int, b []byte, o Object) int {
	if o == nil {
		return codec.MarshalByte(n, b, 0)
	}
	switch TypeOfObject(o) {
	case _undefined:
		n = codec.MarshalByte(n, b, _undefined)
	case _bool:
		n = codec.MarshalByte(n, b, _bool)
		n = codec.MarshalBool(n, b, o.(*Bool).value)
	case _bytes:
		n = codec.MarshalByte(n, b, _bytes)
		n = codec.MarshalBytes(n, b, o.(*Bytes).Value)
	case _char:
		n = codec.MarshalByte(n, b, _char)
		n = codec.MarshalInt32(n, b, o.(*Char).Value)
	case _int:
		n = codec.MarshalByte(n, b, _int)
		n = codec.MarshalInt64(n, b, o.(*Int).Value)
	case _int8:
		n = codec.MarshalByte(n, b, _int8)
		n = codec.MarshalByte(n, b, byte(o.(*Int8).Value))
	case _int16:
		n = codec.MarshalByte(n, b, _int16)
		n = codec.MarshalInt16(n, b, o.(*Int16).Value)
	case _byte:
		n = codec.MarshalByte(n, b, _byte)
		n = codec.MarshalByte(n, b, o.(*Byte).Value)
	case _uint8:
		n = codec.MarshalByte(n, b, _uint8)
		n = codec.MarshalByte(n, b, o.(*Uint8).Value)
	case _uint16:
		n = codec.MarshalByte(n, b, _uint16)
		n = codec.MarshalUint16(n, b, o.(*Uint16).Value)
	case _uint:
		n = codec.MarshalByte(n, b, _uint)
		n = codec.MarshalUint32(n, b, o.(*Uint).Value)
	case _uint64:
		n = codec.MarshalByte(n, b, _uint64)
		n = codec.MarshalUint64(n, b, o.(*Uint64).Value)
	case _ptr:
		// Ptr values are process-local and must never be serialised into
		// bytecode; doing so would let a crafted bytecode file inject
		// arbitrary pointer constants (see security issue: Ptr constructible
		// from integer).
		panic("marshal: Ptr cannot be serialised into bytecode")
	case _float64:
		n = codec.MarshalByte(n, b, _float64)
		n = codec.MarshalFloat64(n, b, o.(*Float64).Value)
	case _float32:
		n = codec.MarshalByte(n, b, _float32)
		n = codec.MarshalFloat32(n, b, o.(*Float32).Value)
	case _string:
		n = codec.MarshalByte(n, b, _string)
		n = codec.MarshalString(n, b, o.(*String).Value)
	case _time:
		n = codec.MarshalByte(n, b, _time)
		n = codec.MarshalInt64(n, b, o.(*Time).Value.UnixNano())
	case _array:
		arr := o.(*Array)
		arr.mu.RLock()
		snap := append([]Object(nil), arr.Value...)
		arr.mu.RUnlock()
		n = codec.MarshalByte(n, b, _array)
		n = codec.MarshalSlice(n, b, snap, MarshalObject)
	case _map:
		m := o.(*Map)
		m.mu.RLock()
		snapM := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			snapM[k] = v
		}
		m.mu.RUnlock()
		n = codec.MarshalByte(n, b, _map)
		n = codec.MarshalMap(n, b, snapM, codec.MarshalString, MarshalObject)
	case _immutableArray:
		n = codec.MarshalByte(n, b, _immutableArray)
		n = codec.MarshalSlice(n, b, o.(*ImmutableArray).Value, MarshalObject)
	case _immutableMap:
		im := o.(*ImmutableMap)
		n = codec.MarshalByte(n, b, _immutableMap)
		n = codec.MarshalString(n, b, im.moduleName)
		n = codec.MarshalMap(n, b, im.Value, codec.MarshalString, MarshalObject)
	case _objectPtr:
		n = codec.MarshalByte(n, b, _objectPtr)
		if o.(*ObjectPtr).Value != nil {
			n = MarshalObject(n, b, *o.(*ObjectPtr).Value)
		} else {
			n = MarshalObject(n, b, nil)
		}
	case _compiledFunction:
		v := o.(*CompiledFunction)
		n = codec.MarshalByte(n, b, _compiledFunction)
		n = codec.MarshalBytes(n, b, v.Instructions)
		n = codec.MarshalInt(n, b, v.NumLocals)
		n = codec.MarshalInt(n, b, v.NumParameters)
		n = codec.MarshalBool(n, b, v.VarArgs)
		n = codec.MarshalMap(n, b, v.SourceMap, codec.MarshalInt, parser.MarshalPos)
		n = codec.MarshalSlice(n, b, v.Free, func(n int, b []byte, o *ObjectPtr) int { return MarshalObject(n, b, o) })
	case _builtinFunction:
		n = codec.MarshalByte(n, b, _builtinFunction)
		n = codec.MarshalString(n, b, o.(*BuiltinFunction).Name)
	case _error:
		n = codec.MarshalByte(n, b, _error)
		n = MarshalObject(n, b, o.(*Error).Value)
	case _rangeObject:
		r := o.(*RangeObject)
		n = codec.MarshalByte(n, b, _rangeObject)
		n = codec.MarshalInt64(n, b, r.Start)
		n = codec.MarshalInt64(n, b, r.Stop)
		n = codec.MarshalInt64(n, b, r.Step)
	default:
		panic("marshal: unsupported type: " + o.TypeName())
	}
	return n
}

// UnmarshalObject unmarshals the given byte slice into an object.
func UnmarshalObject(nn int, b []byte) (n int, o Object, err error) {
	if nn >= len(b) {
		return nn, nil, errors.New("unmarshal: buffer too small to read type code")
	}
	if b[nn] == 0 {
		return nn + 1, nil, nil
	}
	var t byte
	n, t, err = codec.UnmarshalByte(nn, b)
	if err != nil {
		return nn, nil, err
	}
	o = MakeObject(t)
	if o == nil {
		return nn, nil, fmt.Errorf("unmarshal: unknown object type code: %d", t)
	}
	switch t {
	case _undefined:
		return n, o, nil
	case _bool:
		n, o.(*Bool).value, err = codec.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _bytes:
		n, o.(*Bytes).Value, err = codec.UnmarshalBytes(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _char:
		n, o.(*Char).Value, err = codec.UnmarshalInt32(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _int:
		n, o.(*Int).Value, err = codec.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _int8:
		var u byte
		n, u, err = codec.UnmarshalByte(n, b)
		if err != nil {
			return nn, nil, err
		}
		o.(*Int8).Value = int8(u)
		return n, o, nil
	case _int16:
		n, o.(*Int16).Value, err = codec.UnmarshalInt16(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _byte:
		n, o.(*Byte).Value, err = codec.UnmarshalByte(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _uint8:
		n, o.(*Uint8).Value, err = codec.UnmarshalByte(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _uint16:
		n, o.(*Uint16).Value, err = codec.UnmarshalUint16(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _uint:
		n, o.(*Uint).Value, err = codec.UnmarshalUint32(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _uint64:
		n, o.(*Uint64).Value, err = codec.UnmarshalUint64(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _ptr:
		// Ptr values are process-local and must never be deserialised from
		// bytecode; a crafted bytecode file could otherwise inject arbitrary
		// pointer constants.
		return nn, nil, errors.New("unmarshal: Ptr cannot be deserialised from bytecode")
	case _float64:
		n, o.(*Float64).Value, err = codec.UnmarshalFloat64(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _float32:
		n, o.(*Float32).Value, err = codec.UnmarshalFloat32(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _string:
		n, o.(*String).Value, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _time:
		var v int64
		n, v, err = codec.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		o.(*Time).Value = time.Unix(0, v).In(time.UTC)
		return n, o, nil
	case _array:
		n, o.(*Array).Value, err = codec.UnmarshalSlice[Object](n, b, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _map:
		n, o.(*Map).Value, err = codec.UnmarshalMap[string, Object](n, b, codec.UnmarshalString, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _immutableArray:
		n, o.(*ImmutableArray).Value, err = codec.UnmarshalSlice[Object](n, b, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _immutableMap:
		im := o.(*ImmutableMap)
		n, im.moduleName, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, im.Value, err = codec.UnmarshalMap[string, Object](n, b, codec.UnmarshalString, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, im, nil
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
		n, o.(*CompiledFunction).Instructions, err = codec.UnmarshalBytes(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).NumLocals, err = codec.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).NumParameters, err = codec.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).VarArgs, err = codec.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).SourceMap, err = codec.UnmarshalMap[int, parser.Pos](n, b, codec.UnmarshalInt, parser.UnmarshalPos)
		if err != nil {
			return nn, nil, err
		}
		n, o.(*CompiledFunction).Free, err = codec.UnmarshalSlice[*ObjectPtr](n, b, func(nn int, b []byte) (n int, o *ObjectPtr, err error) {
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
		n, o.(*BuiltinFunction).Name, err = codec.UnmarshalString(n, b)
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
	case _rangeObject:
		r := o.(*RangeObject)
		n, r.Start, err = codec.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, r.Stop, err = codec.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, r.Step, err = codec.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, r, nil
	}
	return nn, nil, errors.New("unmarshal: unsupported type: " + o.TypeName())
}
