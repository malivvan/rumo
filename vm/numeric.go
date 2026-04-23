package vm

import (
	"strconv"
	"unsafe"

	"github.com/malivvan/rumo/vm/token"
)

// Double is an alias for Float64 (64-bit IEEE 754).
type Double = Float64

// Int64 is an alias for Int; Int is stored as int64 so Int64 is identical.
type Int64 = Int

// Rune is an alias for Char (Unicode code point).
type Rune = Char

// signedIntBinaryOp implements binary operators for signed integer types by
// upcasting both operands to int64. wrap converts the int64 result back into
// the appropriate concrete type.
func signedIntBinaryOp(op token.Token, lv, rv int64, wrap func(int64) Object) (Object, error) {
	switch op {
	case token.Add:
		return wrap(lv + rv), nil
	case token.Sub:
		return wrap(lv - rv), nil
	case token.Mul:
		return wrap(lv * rv), nil
	case token.Quo:
		if rv == 0 {
			return nil, ErrDivisionByZero
		}
		return wrap(lv / rv), nil
	case token.Rem:
		if rv == 0 {
			return nil, ErrDivisionByZero
		}
		return wrap(lv % rv), nil
	case token.And:
		return wrap(lv & rv), nil
	case token.Or:
		return wrap(lv | rv), nil
	case token.Xor:
		return wrap(lv ^ rv), nil
	case token.AndNot:
		return wrap(lv &^ rv), nil
	case token.Shl:
		return wrap(lv << uint64(rv)), nil
	case token.Shr:
		return wrap(lv >> uint64(rv)), nil
	case token.Less:
		return boolObject(lv < rv), nil
	case token.Greater:
		return boolObject(lv > rv), nil
	case token.LessEq:
		return boolObject(lv <= rv), nil
	case token.GreaterEq:
		return boolObject(lv >= rv), nil
	}
	return nil, ErrInvalidOperator
}

// unsignedIntBinaryOp implements binary operators for unsigned integer types
// by upcasting both operands to uint64.
func unsignedIntBinaryOp(op token.Token, lv, rv uint64, wrap func(uint64) Object) (Object, error) {
	switch op {
	case token.Add:
		return wrap(lv + rv), nil
	case token.Sub:
		return wrap(lv - rv), nil
	case token.Mul:
		return wrap(lv * rv), nil
	case token.Quo:
		if rv == 0 {
			return nil, ErrDivisionByZero
		}
		return wrap(lv / rv), nil
	case token.Rem:
		if rv == 0 {
			return nil, ErrDivisionByZero
		}
		return wrap(lv % rv), nil
	case token.And:
		return wrap(lv & rv), nil
	case token.Or:
		return wrap(lv | rv), nil
	case token.Xor:
		return wrap(lv ^ rv), nil
	case token.AndNot:
		return wrap(lv &^ rv), nil
	case token.Shl:
		return wrap(lv << rv), nil
	case token.Shr:
		return wrap(lv >> rv), nil
	case token.Less:
		return boolObject(lv < rv), nil
	case token.Greater:
		return boolObject(lv > rv), nil
	case token.LessEq:
		return boolObject(lv <= rv), nil
	case token.GreaterEq:
		return boolObject(lv >= rv), nil
	}
	return nil, ErrInvalidOperator
}

// Byte represents a byte value (signed 8-bit in rumo semantics; stored as Go byte).
type Byte struct {
	ObjectImpl
	Value byte
}

func (o *Byte) String() string       { return strconv.FormatInt(int64(int8(o.Value)), 10) }
func (o *Byte) TypeName() string     { return "byte" }
func (o *Byte) Copy() Object         { return &Byte{Value: o.Value} }
func (o *Byte) IsFalsy() bool        { return o.Value == 0 }
func (o *Byte) Equals(x Object) bool { t, ok := x.(*Byte); return ok && o.Value == t.Value }
func (o *Byte) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToInt64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return signedIntBinaryOp(op, int64(int8(o.Value)), rv, func(r int64) Object { return &Byte{Value: byte(r)} })
}

// Int8 represents a signed 8-bit integer.
type Int8 struct {
	ObjectImpl
	Value int8
}

func (o *Int8) String() string       { return strconv.FormatInt(int64(o.Value), 10) }
func (o *Int8) TypeName() string     { return "int8" }
func (o *Int8) Copy() Object         { return &Int8{Value: o.Value} }
func (o *Int8) IsFalsy() bool        { return o.Value == 0 }
func (o *Int8) Equals(x Object) bool { t, ok := x.(*Int8); return ok && o.Value == t.Value }
func (o *Int8) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToInt64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return signedIntBinaryOp(op, int64(o.Value), rv, func(r int64) Object { return &Int8{Value: int8(r)} })
}

// Uint8 represents an unsigned 8-bit integer.
type Uint8 struct {
	ObjectImpl
	Value uint8
}

func (o *Uint8) String() string       { return strconv.FormatUint(uint64(o.Value), 10) }
func (o *Uint8) TypeName() string     { return "uint8" }
func (o *Uint8) Copy() Object         { return &Uint8{Value: o.Value} }
func (o *Uint8) IsFalsy() bool        { return o.Value == 0 }
func (o *Uint8) Equals(x Object) bool { t, ok := x.(*Uint8); return ok && o.Value == t.Value }
func (o *Uint8) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToUint64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return unsignedIntBinaryOp(op, uint64(o.Value), rv, func(r uint64) Object { return &Uint8{Value: uint8(r)} })
}

// Int16 represents a signed 16-bit integer.
type Int16 struct {
	ObjectImpl
	Value int16
}

func (o *Int16) String() string       { return strconv.FormatInt(int64(o.Value), 10) }
func (o *Int16) TypeName() string     { return "int16" }
func (o *Int16) Copy() Object         { return &Int16{Value: o.Value} }
func (o *Int16) IsFalsy() bool        { return o.Value == 0 }
func (o *Int16) Equals(x Object) bool { t, ok := x.(*Int16); return ok && o.Value == t.Value }
func (o *Int16) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToInt64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return signedIntBinaryOp(op, int64(o.Value), rv, func(r int64) Object { return &Int16{Value: int16(r)} })
}

// Uint16 represents an unsigned 16-bit integer.
type Uint16 struct {
	ObjectImpl
	Value uint16
}

func (o *Uint16) String() string       { return strconv.FormatUint(uint64(o.Value), 10) }
func (o *Uint16) TypeName() string     { return "uint16" }
func (o *Uint16) Copy() Object         { return &Uint16{Value: o.Value} }
func (o *Uint16) IsFalsy() bool        { return o.Value == 0 }
func (o *Uint16) Equals(x Object) bool { t, ok := x.(*Uint16); return ok && o.Value == t.Value }
func (o *Uint16) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToUint64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return unsignedIntBinaryOp(op, uint64(o.Value), rv, func(r uint64) Object { return &Uint16{Value: uint16(r)} })
}

// Uint represents an unsigned 32-bit integer (per rumo Type Mapping).
type Uint struct {
	ObjectImpl
	Value uint32
}

func (o *Uint) String() string       { return strconv.FormatUint(uint64(o.Value), 10) }
func (o *Uint) TypeName() string     { return "uint" }
func (o *Uint) Copy() Object         { return &Uint{Value: o.Value} }
func (o *Uint) IsFalsy() bool        { return o.Value == 0 }
func (o *Uint) Equals(x Object) bool { t, ok := x.(*Uint); return ok && o.Value == t.Value }
func (o *Uint) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToUint64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return unsignedIntBinaryOp(op, uint64(o.Value), rv, func(r uint64) Object { return &Uint{Value: uint32(r)} })
}

// Uint64 represents an unsigned 64-bit integer.
type Uint64 struct {
	ObjectImpl
	Value uint64
}

func (o *Uint64) String() string       { return strconv.FormatUint(o.Value, 10) }
func (o *Uint64) TypeName() string     { return "uint64" }
func (o *Uint64) Copy() Object         { return &Uint64{Value: o.Value} }
func (o *Uint64) IsFalsy() bool        { return o.Value == 0 }
func (o *Uint64) Equals(x Object) bool { t, ok := x.(*Uint64); return ok && o.Value == t.Value }
func (o *Uint64) BinaryOp(op token.Token, rhs Object) (Object, error) {
	rv, ok := ToUint64(rhs)
	if !ok {
		return nil, ErrInvalidOperator
	}
	return unsignedIntBinaryOp(op, o.Value, rv, func(r uint64) Object { return &Uint64{Value: r} })
}

// Ptr represents an untyped pointer (unsafe.Pointer in Go, void* in C).
type Ptr struct {
	ObjectImpl
	Value unsafe.Pointer
}

func (o *Ptr) String() string {
	return "0x" + strconv.FormatUint(uint64(uintptr(o.Value)), 16)
}
func (o *Ptr) TypeName() string     { return "ptr" }
func (o *Ptr) Copy() Object         { return &Ptr{Value: o.Value} }
func (o *Ptr) IsFalsy() bool        { return o.Value == nil }
func (o *Ptr) Equals(x Object) bool { t, ok := x.(*Ptr); return ok && o.Value == t.Value }
func (o *Ptr) BinaryOp(op token.Token, rhs Object) (Object, error) {
	t, ok := rhs.(*Ptr)
	if !ok {
		return nil, ErrInvalidOperator
	}
	switch op {
	case token.Less:
		return boolObject(uintptr(o.Value) < uintptr(t.Value)), nil
	case token.Greater:
		return boolObject(uintptr(o.Value) > uintptr(t.Value)), nil
	case token.LessEq:
		return boolObject(uintptr(o.Value) <= uintptr(t.Value)), nil
	case token.GreaterEq:
		return boolObject(uintptr(o.Value) >= uintptr(t.Value)), nil
	}
	return nil, ErrInvalidOperator
}
