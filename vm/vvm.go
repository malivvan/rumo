package vm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unsafe"
)



// Config holds the VM limits. Pass a *Config to NewVM to override the
// defaults. Passing nil selects DefaultConfig. Zero values in a non-nil
// Config are filled in from DefaultConfig by the eval method.
type Config struct {
	// GlobalsSize is the maximum number of global variables for a VM.
	GlobalsSize int

	// StackSize is the maximum stack size for a VM.
	StackSize int

	// MaxFrames is the maximum number of function frames for a VM.
	MaxFrames int

	// MaxAllocs is the maximum number of object allocations (-1 = unlimited).
	MaxAllocs int64

	// MaxStringLen is the maximum byte-length for string values.
	MaxStringLen int

	// MaxBytesLen is the maximum length for bytes values.
	MaxBytesLen int
}

// DefaultConfig is the Config used when nil is passed to NewVM.
var DefaultConfig = &Config{
	GlobalsSize:  1024,
	StackSize:    2048,
	MaxFrames:    1024,
	MaxAllocs:    -1,
	MaxStringLen: 2147483647,
	MaxBytesLen:  2147483647,
}

// eval replaces every zero-valued field with the corresponding value from
// DefaultConfig. Called internally by NewVM on a private copy of the caller's
// Config so that the original is never mutated.
func (c *Config) eval() {
	if c.GlobalsSize == 0 {
		c.GlobalsSize = DefaultConfig.GlobalsSize
	}
	if c.StackSize == 0 {
		c.StackSize = DefaultConfig.StackSize
	}
	if c.MaxFrames == 0 {
		c.MaxFrames = DefaultConfig.MaxFrames
	}
	if c.MaxAllocs == 0 {
		c.MaxAllocs = DefaultConfig.MaxAllocs
	}
	if c.MaxStringLen == 0 {
		c.MaxStringLen = DefaultConfig.MaxStringLen
	}
	if c.MaxBytesLen == 0 {
		c.MaxBytesLen = DefaultConfig.MaxBytesLen
	}
}

// CallableFunc is a function signature for the callable functions.
type CallableFunc = func(ctx context.Context, args ...Object) (ret Object, err error)

// CountObjects returns the number of objects that a given object o contains.
// For scalar value types, it will always be 1. For compound value types,
// this will include its elements and all of their elements recursively.
func CountObjects(o Object) (c int) {
	c = 1
	switch o := o.(type) {
	case *Array:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *ImmutableArray:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *Map:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *ImmutableMap:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *Error:
		c += CountObjects(o.Value)
	}
	return
}

// ToString will try to convert object o to string value.
func ToString(o Object) (v string, ok bool) {
	if o == UndefinedValue {
		return
	}
	ok = true
	if str, isStr := o.(*String); isStr {
		v = str.Value
	} else {
		v = o.String()
	}
	return
}

// ToInt will try to convert object o to int value.
func ToInt(o Object) (v int, ok bool) {
	n, ok := ToInt64(o)
	if ok {
		v = int(n)
	}
	return
}

// ToInt64 will try to convert object o to int64 value.
func ToInt64(o Object) (v int64, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = o.Value
		ok = true
	case *Int8:
		v = int64(o.Value)
		ok = true
	case *Int16:
		v = int64(o.Value)
		ok = true
	case *Byte:
		v = int64(int8(o.Value))
		ok = true
	case *Uint8:
		v = int64(o.Value)
		ok = true
	case *Uint16:
		v = int64(o.Value)
		ok = true
	case *Uint:
		v = int64(o.Value)
		ok = true
	case *Uint64:
		v = int64(o.Value)
		ok = true
	case *Float32:
		v = int64(o.Value)
		ok = true
	case *Float64:
		v = int64(o.Value)
		ok = true
	case *Char:
		v = int64(o.Value)
		ok = true
	case *Bool:
		if o == TrueValue {
			v = 1
		}
		ok = true
	case *String:
		c, err := strconv.ParseInt(o.Value, 10, 64)
		if err == nil {
			v = c
			ok = true
		}
	}
	return
}

// ToUint64 will try to convert object o to uint64 value.
func ToUint64(o Object) (v uint64, ok bool) {
	switch o := o.(type) {
	case *Uint64:
		v = o.Value
		ok = true
	case *Uint:
		v = uint64(o.Value)
		ok = true
	case *Uint16:
		v = uint64(o.Value)
		ok = true
	case *Uint8:
		v = uint64(o.Value)
		ok = true
	case *Int:
		v = uint64(o.Value)
		ok = true
	case *Int8:
		v = uint64(o.Value)
		ok = true
	case *Int16:
		v = uint64(o.Value)
		ok = true
	case *Byte:
		v = uint64(o.Value)
		ok = true
	case *Float32:
		v = uint64(o.Value)
		ok = true
	case *Float64:
		v = uint64(o.Value)
		ok = true
	case *Char:
		v = uint64(o.Value)
		ok = true
	case *Bool:
		if o == TrueValue {
			v = 1
		}
		ok = true
	case *String:
		c, err := strconv.ParseUint(o.Value, 10, 64)
		if err == nil {
			v = c
			ok = true
		}
	}
	return
}

// ToFloat32 will try to convert object o to float32 value.
func ToFloat32(o Object) (v float32, ok bool) {
	f, ok := ToFloat64(o)
	if ok {
		v = float32(f)
	}
	return
}

// ToFloat64 will try to convert object o to float64 value.
func ToFloat64(o Object) (v float64, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = float64(o.Value)
		ok = true
	case *Int8:
		v = float64(o.Value)
		ok = true
	case *Int16:
		v = float64(o.Value)
		ok = true
	case *Byte:
		v = float64(int8(o.Value))
		ok = true
	case *Uint8:
		v = float64(o.Value)
		ok = true
	case *Uint16:
		v = float64(o.Value)
		ok = true
	case *Uint:
		v = float64(o.Value)
		ok = true
	case *Uint64:
		v = float64(o.Value)
		ok = true
	case *Float32:
		v = float64(o.Value)
		ok = true
	case *Float64:
		v = o.Value
		ok = true
	case *String:
		c, err := strconv.ParseFloat(o.Value, 64)
		if err == nil {
			v = c
			ok = true
		}
	}
	return
}

// ToPtr tries to convert object o to an unsafe.Pointer value. Only an
// existing Ptr or Undefined (nil) are accepted; integer-to-pointer coercion
// is intentionally rejected to prevent scripts from fabricating arbitrary
// memory addresses.
func ToPtr(o Object) (v unsafe.Pointer, ok bool) {
	switch o := o.(type) {
	case *Ptr:
		v = o.Value
		ok = true
	case *Undefined:
		v = nil
		ok = true
	}
	return
}

// ToBool will try to convert object o to bool value.
func ToBool(o Object) (v bool, ok bool) {
	ok = true
	v = !o.IsFalsy()
	return
}

// ToRune will try to convert object o to rune value.
func ToRune(o Object) (v rune, ok bool) {
	switch o := o.(type) {
	case *Int:
		v = rune(o.Value)
		ok = true
	case *Char:
		v = o.Value
		ok = true
	}
	return
}

// ToByteSlice will try to convert object o to []byte value.
func ToByteSlice(o Object) (v []byte, ok bool) {
	switch o := o.(type) {
	case *Bytes:
		v = o.Value
		ok = true
	case *String:
		v = []byte(o.Value)
		ok = true
	}
	return
}

// ToTime will try to convert object o to time.Time value.
func ToTime(o Object) (v time.Time, ok bool) {
	switch o := o.(type) {
	case *Time:
		v = o.Value
		ok = true
	case *Int:
		v = time.Unix(o.Value, 0)
		ok = true
	}
	return
}

// ToInterface attempts to convert an object o to an interface{} value
func ToInterface(o Object) (res interface{}) {
	switch o := o.(type) {
	case *Int:
		res = o.Value
	case *Int8:
		res = o.Value
	case *Int16:
		res = o.Value
	case *Byte:
		res = o.Value
	case *Uint8:
		res = o.Value
	case *Uint16:
		res = o.Value
	case *Uint:
		res = o.Value
	case *Uint64:
		res = o.Value
	case *String:
		res = o.Value
	case *Float32:
		res = o.Value
	case *Float64:
		res = o.Value
	case *Bool:
		res = o == TrueValue
	case *Char:
		res = o.Value
	case *Bytes:
		res = o.Value
	case *Ptr:
		res = o.Value
	case *Array:
		res = make([]interface{}, len(o.Value))
		for i, val := range o.Value {
			res.([]interface{})[i] = ToInterface(val)
		}
	case *ImmutableArray:
		res = make([]interface{}, len(o.Value))
		for i, val := range o.Value {
			res.([]interface{})[i] = ToInterface(val)
		}
	case *Map:
		res = make(map[string]interface{})
		for key, v := range o.Value {
			res.(map[string]interface{})[key] = ToInterface(v)
		}
	case *ImmutableMap:
		res = make(map[string]interface{})
		for key, v := range o.Value {
			res.(map[string]interface{})[key] = ToInterface(v)
		}
	case *Time:
		res = o.Value
	case *Error:
		res = errors.New(o.String())
	case *Undefined:
		res = nil
	case Object:
		return o
	}
	return
}

// FromInterface will attempt to convert an interface{} v to a vm Object
func FromInterface(v interface{}) (Object, error) {
	switch v := v.(type) {
	case nil:
		return UndefinedValue, nil
	case string:
		if len(v) > DefaultConfig.MaxStringLen {
			return nil, ErrStringLimit
		}
		return &String{Value: v}, nil
	case int64:
		return &Int{Value: v}, nil
	case int:
		return &Int{Value: int64(v)}, nil
	case int8:
		return &Int8{Value: v}, nil
	case int16:
		return &Int16{Value: v}, nil
	case uint:
		return &Uint{Value: uint32(v)}, nil
	case uint8:
		return &Uint8{Value: v}, nil
	case uint16:
		return &Uint16{Value: v}, nil
	case uint32:
		return &Uint{Value: v}, nil
	case uint64:
		return &Uint64{Value: v}, nil
	case uintptr:
		return &Ptr{Value: unsafe.Pointer(v)}, nil
	case unsafe.Pointer:
		return &Ptr{Value: v}, nil
	case bool:
		if v {
			return TrueValue, nil
		}
		return FalseValue, nil
	case rune:
		return &Char{Value: v}, nil
	case float32:
		return &Float32{Value: v}, nil
	case float64:
		return &Float64{Value: v}, nil
	case []byte:
		if len(v) > DefaultConfig.MaxBytesLen {
			return nil, ErrBytesLimit
		}
		return &Bytes{Value: v}, nil
	case error:
		return &Error{Value: &String{Value: v.Error()}}, nil
	case map[string]Object:
		return &Map{Value: v}, nil
	case map[string]interface{}:
		kv := make(map[string]Object)
		for vk, vv := range v {
			vo, err := FromInterface(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &Map{Value: kv}, nil
	case []Object:
		return &Array{Value: v}, nil
	case []interface{}:
		arr := make([]Object, len(v))
		for i, e := range v {
			vo, err := FromInterface(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &Array{Value: arr}, nil
	case time.Time:
		return &Time{Value: v}, nil
	case Object:
		return v, nil
	case CallableFunc:
		return &BuiltinFunction{Value: v}, nil
	}
	return nil, fmt.Errorf("cannot convert to object: %T", v)
}
