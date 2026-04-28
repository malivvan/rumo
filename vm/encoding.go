package vm

import (
	"errors"
	"fmt"

	"github.com/malivvan/rumo/vm/codec"
	"github.com/malivvan/rumo/vm/parser"
)

const (
	_undefined byte = 1
	_bool      byte = 2
	_bytes     byte = 3
	_char      byte = 4
	_int       byte = 5
	_float     byte = 6  // legacy alias for _float64
	_float64   byte = 6  // float64 / C double
	_float32   byte = 16 // float32 / C float
	_string    byte = 7
	//	_time             byte = 8 // reserved (formerly *Time); do not reuse to preserve bytecode compatibility
	_array byte = 9
	_map   byte = 10
	//	_immutableArray   byte = 11
	//	_immutableMap     byte = 12
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
	_int32            byte = 25
	_arrayIterator    byte = 100
	_mapIterator      byte = 101
	_stringIterator   byte = 102
	_bytesIterator    byte = 103
	_builtinFunction  byte = 104
	_rangeObject      byte = 105
	_userType         byte = 106
	_nativeLoader     byte = 107
	_chan             byte = 108
)

var _typeMap = map[byte]func() Object{
	_undefined:        func() Object { return &Undefined{} },
	_bool:             func() Object { return &Bool{} },
	_bytes:            func() Object { return &Bytes{} },
	_char:             func() Object { return &Char{} },
	_int:              func() Object { return &Int{} },
	_int8:             func() Object { return &Int8{} },
	_int16:            func() Object { return &Int16{} },
	_int32:            func() Object { return &Int32{} },
	_byte:             func() Object { return &Byte{} },
	_uint8:            func() Object { return &Uint8{} },
	_uint16:           func() Object { return &Uint16{} },
	_uint:             func() Object { return &Uint{} },
	_uint64:           func() Object { return &Uint64{} },
	_ptr:              func() Object { return &Ptr{} },
	_float:            func() Object { return &Float64{} },
	_float32:          func() Object { return &Float32{} },
	_string:           func() Object { return &String{} },
	_array:            func() Object { return &Array{} },
	_map:              func() Object { return &Map{Value: make(map[string]Object)} },
	_objectPtr:        func() Object { return &ObjectPtr{} },
	_compiledFunction: func() Object { return &CompiledFunction{SourceMap: make(map[int]parser.Pos)} },
	_builtinFunction:  func() Object { return &BuiltinFunction{} },
	_error:            func() Object { return &Error{} },
	_rangeObject:      func() Object { return &RangeObject{} },
	_userType:         func() Object { return &UserType{} },
	_nativeLoader:     func() Object { return &Native{} },
	_chan:             func() Object { return &Chan{} },
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
//
// Must mirror the marshaler used in the _compiledFunction Free-slice
// path, which calls MarshalObject(*ObjectPtr) — that dispatches to the
// _objectPtr case which writes a 1-byte type tag plus the inner value.
// Returning only SizeOfObject(*op.Value) would under-count by one byte
// per free variable and cause MarshalSlice's trailing sentinel write to
// panic with "slice bounds out of range" once the slice is non-empty.
func SizeOfObjectPtr(op *ObjectPtr) int {
	if op == nil || op.Value == nil {
		return codec.SizeByte() + SizeOfObject(nil)
	}
	return codec.SizeByte() + SizeOfObject(*op.Value)
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
	case *Int32:
		return _int32
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
	case *Array:
		return _array
	case *Map:
		return _map
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
	case *UserType:
		return _userType
	case *Native:
		return _nativeLoader
	case *Chan:
		return _chan
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
	case _int32:
		return codec.SizeByte() + codec.SizeInt32()
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
	case _array:
		arr := o.(*Array)
		arr.mu.RLock()
		snap := append([]Object(nil), arr.Value...)
		arr.mu.RUnlock()
		return codec.SizeByte() + codec.SizeBool() + codec.SizeSlice(snap, SizeOfObject)
	case _map:
		m := o.(*Map)
		m.mu.RLock()
		snapM := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			snapM[k] = v
		}
		modName := m.moduleName
		m.mu.RUnlock()
		return codec.SizeByte() + codec.SizeBool() + codec.SizeString(modName) + codec.SizeMap(snapM, codec.SizeString, SizeOfObject)
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
	case _userType:
		ut := o.(*UserType)
		s := codec.SizeString(ut.Name)
		s += codec.SizeByte() // Kind
		s += codec.SizeSlice[string](ut.Fields, codec.SizeString)
		s += codec.SizeSlice[string](ut.FieldTypes, codec.SizeString)
		s += codec.SizeSlice[string](ut.Params, codec.SizeString)
		s += codec.SizeSlice[string](ut.ParamTypes, codec.SizeString)
		s += codec.SizeBool()            // VarArgs
		s += codec.SizeInt(ut.NumParams) // NumParams
		s += codec.SizeString(ut.Result)
		s += codec.SizeString(ut.Underlying)
		return codec.SizeByte() + s
	case _nativeLoader:
		nl := o.(*Native)
		s := codec.SizeString(nl.Path)
		s += codec.SizeInt(len(nl.Structs))
		for _, st := range nl.Structs {
			s += codec.SizeString(st.Name)
			s += codec.SizeInt(len(st.Fields))
			for _, f := range st.Fields {
				s += codec.SizeString(f.Name)
				s += codec.SizeByte()             // Kind
				s += codec.SizeInt(f.StructIdx)   // StructIdx
				s += codec.SizeBool()             // Pointer
			}
		}
		s += codec.SizeInt(len(nl.Funcs))
		for _, f := range nl.Funcs {
			s += codec.SizeString(f.Name)
			s += codec.SizeByte() // Return kind
			s += codec.SizeInt(f.ReturnStructIdx)
			s += codec.SizeBool()
			s += codec.SizeInt(len(f.Params))
			s += len(f.Params) * codec.SizeByte()
			s += codec.SizeInt(len(f.ParamStructIdx))
			for _, idx := range f.ParamStructIdx {
				s += codec.SizeInt(idx)
			}
			s += codec.SizeInt(len(f.ParamPointer))
			s += len(f.ParamPointer) * codec.SizeBool()
		}
		return codec.SizeByte() + s
	case _chan:
		// channels travel as just their int64 id; the receiver upgrades the
		// core via ResolveChans (see chan.go) before any send/recv call.
		return codec.SizeByte() + codec.SizeInt64()
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
	case _int32:
		n = codec.MarshalByte(n, b, _int32)
		n = codec.MarshalInt32(n, b, o.(*Int32).Value)
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
	case _array:
		arr := o.(*Array)
		arr.mu.RLock()
		snap := append([]Object(nil), arr.Value...)
		frozen := arr.Frozen
		arr.mu.RUnlock()
		n = codec.MarshalByte(n, b, _array)
		n = codec.MarshalBool(n, b, frozen)
		n = codec.MarshalSlice(n, b, snap, MarshalObject)
	case _map:
		m := o.(*Map)
		m.mu.RLock()
		snapM := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			snapM[k] = v
		}
		frozen := m.Frozen
		modName := m.moduleName
		m.mu.RUnlock()
		n = codec.MarshalByte(n, b, _map)
		n = codec.MarshalBool(n, b, frozen)
		n = codec.MarshalString(n, b, modName)
		n = codec.MarshalMap(n, b, snapM, codec.MarshalString, MarshalObject)
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
	case _userType:
		ut := o.(*UserType)
		n = codec.MarshalByte(n, b, _userType)
		n = codec.MarshalString(n, b, ut.Name)
		n = codec.MarshalByte(n, b, byte(ut.Kind))
		n = codec.MarshalSlice[string](n, b, ut.Fields, codec.MarshalString)
		n = codec.MarshalSlice[string](n, b, ut.FieldTypes, codec.MarshalString)
		n = codec.MarshalSlice[string](n, b, ut.Params, codec.MarshalString)
		n = codec.MarshalSlice[string](n, b, ut.ParamTypes, codec.MarshalString)
		n = codec.MarshalBool(n, b, ut.VarArgs)
		n = codec.MarshalInt(n, b, ut.NumParams)
		n = codec.MarshalString(n, b, ut.Result)
		n = codec.MarshalString(n, b, ut.Underlying)
	case _nativeLoader:
		nl := o.(*Native)
		n = codec.MarshalByte(n, b, _nativeLoader)
		n = codec.MarshalString(n, b, nl.Path)
		n = codec.MarshalInt(n, b, len(nl.Structs))
		for _, st := range nl.Structs {
			n = codec.MarshalString(n, b, st.Name)
			n = codec.MarshalInt(n, b, len(st.Fields))
			for _, f := range st.Fields {
				n = codec.MarshalString(n, b, f.Name)
				n = codec.MarshalByte(n, b, byte(f.Kind))
				n = codec.MarshalInt(n, b, f.StructIdx)
				n = codec.MarshalBool(n, b, f.Pointer)
			}
		}
		n = codec.MarshalInt(n, b, len(nl.Funcs))
		for _, f := range nl.Funcs {
			n = codec.MarshalString(n, b, f.Name)
			n = codec.MarshalByte(n, b, byte(f.Return))
			n = codec.MarshalInt(n, b, f.ReturnStructIdx)
			n = codec.MarshalBool(n, b, f.ReturnPointer)
			n = codec.MarshalInt(n, b, len(f.Params))
			for _, p := range f.Params {
				n = codec.MarshalByte(n, b, byte(p))
			}
			n = codec.MarshalInt(n, b, len(f.ParamStructIdx))
			for _, idx := range f.ParamStructIdx {
				n = codec.MarshalInt(n, b, idx)
			}
			n = codec.MarshalInt(n, b, len(f.ParamPointer))
			for _, p := range f.ParamPointer {
				n = codec.MarshalBool(n, b, p)
			}
		}
	case _chan:
		c := o.(*Chan)
		n = codec.MarshalByte(n, b, _chan)
		n = codec.MarshalInt64(n, b, c.id)
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
	case _int32:
		n, o.(*Int32).Value, err = codec.UnmarshalInt32(n, b)
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
	case _array:
		arr := o.(*Array)
		n, arr.Frozen, err = codec.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, arr.Value, err = codec.UnmarshalSlice[Object](n, b, UnmarshalObject)
		if err != nil {
			return nn, nil, err
		}
		return n, o, nil
	case _map:
		m := o.(*Map)
		n, m.Frozen, err = codec.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, m.moduleName, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, m.Value, err = codec.UnmarshalMap[string, Object](n, b, codec.UnmarshalString, UnmarshalObject)
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
	case _userType:
		ut := o.(*UserType)
		n, ut.Name, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		var kindByte byte
		n, kindByte, err = codec.UnmarshalByte(n, b)
		if err != nil {
			return nn, nil, err
		}
		ut.Kind = UserTypeKind(kindByte)
		n, ut.Fields, err = codec.UnmarshalSlice[string](n, b, codec.UnmarshalString)
		if err != nil {
			return nn, nil, err
		}
		n, ut.FieldTypes, err = codec.UnmarshalSlice[string](n, b, codec.UnmarshalString)
		if err != nil {
			return nn, nil, err
		}
		n, ut.Params, err = codec.UnmarshalSlice[string](n, b, codec.UnmarshalString)
		if err != nil {
			return nn, nil, err
		}
		n, ut.ParamTypes, err = codec.UnmarshalSlice[string](n, b, codec.UnmarshalString)
		if err != nil {
			return nn, nil, err
		}
		n, ut.VarArgs, err = codec.UnmarshalBool(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, ut.NumParams, err = codec.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, ut.Result, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		n, ut.Underlying, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		return n, ut, nil
	case _nativeLoader:
		nl := o.(*Native)
		n, nl.Path, err = codec.UnmarshalString(n, b)
		if err != nil {
			return nn, nil, err
		}
		var numStructs int
		n, numStructs, err = codec.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		nl.Structs = make([]NativeStructSpec, numStructs)
		for i := range nl.Structs {
			n, nl.Structs[i].Name, err = codec.UnmarshalString(n, b)
			if err != nil {
				return nn, nil, err
			}
			var numFields int
			n, numFields, err = codec.UnmarshalInt(n, b)
			if err != nil {
				return nn, nil, err
			}
			nl.Structs[i].Fields = make([]NativeStructFieldSpec, numFields)
			for j := range nl.Structs[i].Fields {
				n, nl.Structs[i].Fields[j].Name, err = codec.UnmarshalString(n, b)
				if err != nil {
					return nn, nil, err
				}
				var kb byte
				n, kb, err = codec.UnmarshalByte(n, b)
				if err != nil {
					return nn, nil, err
				}
				nl.Structs[i].Fields[j].Kind = NativeKind(kb)
				n, nl.Structs[i].Fields[j].StructIdx, err = codec.UnmarshalInt(n, b)
				if err != nil {
					return nn, nil, err
				}
				n, nl.Structs[i].Fields[j].Pointer, err = codec.UnmarshalBool(n, b)
				if err != nil {
					return nn, nil, err
				}
			}
		}
		var numFuncs int
		n, numFuncs, err = codec.UnmarshalInt(n, b)
		if err != nil {
			return nn, nil, err
		}
		nl.Funcs = make([]NativeFuncSpec, numFuncs)
		for i := range nl.Funcs {
			n, nl.Funcs[i].Name, err = codec.UnmarshalString(n, b)
			if err != nil {
				return nn, nil, err
			}
			var retByte byte
			n, retByte, err = codec.UnmarshalByte(n, b)
			if err != nil {
				return nn, nil, err
			}
			nl.Funcs[i].Return = NativeKind(retByte)
			n, nl.Funcs[i].ReturnStructIdx, err = codec.UnmarshalInt(n, b)
			if err != nil {
				return nn, nil, err
			}
			n, nl.Funcs[i].ReturnPointer, err = codec.UnmarshalBool(n, b)
			if err != nil {
				return nn, nil, err
			}
			var numParams int
			n, numParams, err = codec.UnmarshalInt(n, b)
			if err != nil {
				return nn, nil, err
			}
			nl.Funcs[i].Params = make([]NativeKind, numParams)
			for j := range nl.Funcs[i].Params {
				var pb byte
				n, pb, err = codec.UnmarshalByte(n, b)
				if err != nil {
					return nn, nil, err
				}
				nl.Funcs[i].Params[j] = NativeKind(pb)
			}
			var numIdx int
			n, numIdx, err = codec.UnmarshalInt(n, b)
			if err != nil {
				return nn, nil, err
			}
			nl.Funcs[i].ParamStructIdx = make([]int, numIdx)
			for j := range nl.Funcs[i].ParamStructIdx {
				n, nl.Funcs[i].ParamStructIdx[j], err = codec.UnmarshalInt(n, b)
				if err != nil {
					return nn, nil, err
				}
			}
			var numPtr int
			n, numPtr, err = codec.UnmarshalInt(n, b)
			if err != nil {
				return nn, nil, err
			}
			nl.Funcs[i].ParamPointer = make([]bool, numPtr)
			for j := range nl.Funcs[i].ParamPointer {
				n, nl.Funcs[i].ParamPointer[j], err = codec.UnmarshalBool(n, b)
				if err != nil {
					return nn, nil, err
				}
			}
		}
		return n, nl, nil
	case _chan:
		c := o.(*Chan)
		n, c.id, err = codec.UnmarshalInt64(n, b)
		if err != nil {
			return nn, nil, err
		}
		// core stays nil here — caller (live codec / runtime) must run
		// ResolveChans() to bind it to a LocalChanCore (if owner) or
		// RemoteChanCore (otherwise) before any send/recv call.
		return n, c, nil
	}
	return nn, nil, errors.New("unmarshal: unsupported type: " + o.TypeName())
}
