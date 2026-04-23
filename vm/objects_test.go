package vm_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
	"github.com/malivvan/rumo/vm/token"
)

func TestObject_TypeName(t *testing.T) {
	var o vm.Object = &vm.Int{}
	require.Equal(t, "int", o.TypeName())
	o = &vm.Float{}
	require.Equal(t, "float64", o.TypeName())
	o = &vm.Float32{}
	require.Equal(t, "float32", o.TypeName())
	o = &vm.Char{}
	require.Equal(t, "char", o.TypeName())
	o = &vm.String{}
	require.Equal(t, "string", o.TypeName())
	o = &vm.Bool{}
	require.Equal(t, "bool", o.TypeName())
	o = &vm.Array{}
	require.Equal(t, "array", o.TypeName())
	o = &vm.Map{}
	require.Equal(t, "map", o.TypeName())
	o = &vm.ArrayIterator{}
	require.Equal(t, "array-iterator", o.TypeName())
	o = &vm.StringIterator{}
	require.Equal(t, "string-iterator", o.TypeName())
	o = &vm.MapIterator{}
	require.Equal(t, "map-iterator", o.TypeName())
	o = &vm.BuiltinFunction{Name: "fn"}
	require.Equal(t, "builtin-function:fn", o.TypeName())
	o = &vm.CompiledFunction{}
	require.Equal(t, "compiled-function", o.TypeName())
	o = &vm.Undefined{}
	require.Equal(t, "undefined", o.TypeName())
	o = &vm.Error{}
	require.Equal(t, "error", o.TypeName())
	o = &vm.Bytes{}
	require.Equal(t, "bytes", o.TypeName())
}

func TestObject_IsFalsy(t *testing.T) {
	var o vm.Object = &vm.Int{Value: 0}
	require.True(t, o.IsFalsy())
	o = &vm.Int{Value: 1}
	require.False(t, o.IsFalsy())
	o = &vm.Float{Value: 0}
	require.False(t, o.IsFalsy())
	o = &vm.Float{Value: 1}
	require.False(t, o.IsFalsy())
	o = &vm.Char{Value: ' '}
	require.False(t, o.IsFalsy())
	o = &vm.Char{Value: 'T'}
	require.False(t, o.IsFalsy())
	o = &vm.String{Value: ""}
	require.True(t, o.IsFalsy())
	o = &vm.String{Value: " "}
	require.False(t, o.IsFalsy())
	o = &vm.Array{Value: nil}
	require.True(t, o.IsFalsy())
	o = &vm.Array{Value: []vm.Object{nil}} // nil is not valid but still count as 1 element
	require.False(t, o.IsFalsy())
	o = &vm.Map{Value: nil}
	require.True(t, o.IsFalsy())
	o = &vm.Map{Value: map[string]vm.Object{"a": nil}} // nil is not valid but still count as 1 element
	require.False(t, o.IsFalsy())
	o = &vm.StringIterator{}
	require.True(t, o.IsFalsy())
	o = &vm.ArrayIterator{}
	require.True(t, o.IsFalsy())
	o = &vm.MapIterator{}
	require.True(t, o.IsFalsy())
	o = &vm.BuiltinFunction{}
	require.False(t, o.IsFalsy())
	o = &vm.CompiledFunction{}
	require.False(t, o.IsFalsy())
	o = &vm.Undefined{}
	require.True(t, o.IsFalsy())
	o = &vm.Error{}
	require.True(t, o.IsFalsy())
	o = &vm.Bytes{}
	require.True(t, o.IsFalsy())
	o = &vm.Bytes{Value: []byte{1, 2}}
	require.False(t, o.IsFalsy())
}

func TestObject_String(t *testing.T) {
	var o vm.Object = &vm.Int{Value: 0}
	require.Equal(t, "0", o.String())
	o = &vm.Int{Value: 1}
	require.Equal(t, "1", o.String())
	o = &vm.Float{Value: 0}
	require.Equal(t, "0", o.String())
	o = &vm.Float{Value: 1}
	require.Equal(t, "1", o.String())
	o = &vm.Char{Value: ' '}
	require.Equal(t, " ", o.String())
	o = &vm.Char{Value: 'T'}
	require.Equal(t, "T", o.String())
	o = &vm.String{Value: ""}
	require.Equal(t, `""`, o.String())
	o = &vm.String{Value: " "}
	require.Equal(t, `" "`, o.String())
	o = &vm.Array{Value: nil}
	require.Equal(t, "[]", o.String())
	o = &vm.Map{Value: nil}
	require.Equal(t, "{}", o.String())
	o = &vm.Error{Value: nil}
	require.Equal(t, "error", o.String())
	o = &vm.Error{Value: &vm.String{Value: "error 1"}}
	require.Equal(t, `error: "error 1"`, o.String())
	o = &vm.StringIterator{}
	require.Equal(t, "<string-iterator>", o.String())
	o = &vm.ArrayIterator{}
	require.Equal(t, "<array-iterator>", o.String())
	o = &vm.MapIterator{}
	require.Equal(t, "<map-iterator>", o.String())
	o = &vm.Undefined{}
	require.Equal(t, "<undefined>", o.String())
	o = &vm.Bytes{}
	require.Equal(t, "", o.String())
	o = &vm.Bytes{Value: []byte("foo")}
	require.Equal(t, "foo", o.String())
}

func TestObject_BinaryOp(t *testing.T) {
	var o vm.Object = &vm.Char{}
	_, err := o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.Bool{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.Map{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.ArrayIterator{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.StringIterator{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.MapIterator{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.BuiltinFunction{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.CompiledFunction{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.Undefined{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
	o = &vm.Error{}
	_, err = o.BinaryOp(token.Add, vm.UndefinedValue)
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, &vm.Array{Value: nil}, token.Add,
		&vm.Array{Value: nil}, &vm.Array{Value: nil})
	testBinaryOp(t, &vm.Array{Value: nil}, token.Add,
		&vm.Array{Value: []vm.Object{}}, &vm.Array{Value: nil})
	testBinaryOp(t, &vm.Array{Value: []vm.Object{}}, token.Add,
		&vm.Array{Value: nil}, &vm.Array{Value: []vm.Object{}})
	testBinaryOp(t, &vm.Array{Value: []vm.Object{}}, token.Add,
		&vm.Array{Value: []vm.Object{}},
		&vm.Array{Value: []vm.Object{}})
	testBinaryOp(t, &vm.Array{Value: nil}, token.Add,
		&vm.Array{Value: []vm.Object{
			&vm.Int{Value: 1},
		}}, &vm.Array{Value: []vm.Object{
			&vm.Int{Value: 1},
		}})
	testBinaryOp(t, &vm.Array{Value: nil}, token.Add,
		&vm.Array{Value: []vm.Object{
			&vm.Int{Value: 1},
			&vm.Int{Value: 2},
			&vm.Int{Value: 3},
		}}, &vm.Array{Value: []vm.Object{
			&vm.Int{Value: 1},
			&vm.Int{Value: 2},
			&vm.Int{Value: 3},
		}})
	testBinaryOp(t, &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 1},
		&vm.Int{Value: 2},
		&vm.Int{Value: 3},
	}}, token.Add, &vm.Array{Value: nil},
		&vm.Array{Value: []vm.Object{
			&vm.Int{Value: 1},
			&vm.Int{Value: 2},
			&vm.Int{Value: 3},
		}})
	testBinaryOp(t, &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 1},
		&vm.Int{Value: 2},
		&vm.Int{Value: 3},
	}}, token.Add, &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 4},
		&vm.Int{Value: 5},
		&vm.Int{Value: 6},
	}}, &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 1},
		&vm.Int{Value: 2},
		&vm.Int{Value: 3},
		&vm.Int{Value: 4},
		&vm.Int{Value: 5},
		&vm.Int{Value: 6},
	}})
}

func TestError_Equals(t *testing.T) {
	err1 := &vm.Error{Value: &vm.String{Value: "some error"}}
	err2 := err1
	require.True(t, err1.Equals(err2))
	require.True(t, err2.Equals(err1))

	err2 = &vm.Error{Value: &vm.String{Value: "some error"}}
	require.False(t, err1.Equals(err2))
	require.False(t, err2.Equals(err1))
}

func TestFloat_BinaryOp(t *testing.T) {
	// float + float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.Add,
				&vm.Float{Value: r}, &vm.Float{Value: l + r})
		}
	}

	// float - float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.Sub,
				&vm.Float{Value: r}, &vm.Float{Value: l - r})
		}
	}

	// float * float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.Mul,
				&vm.Float{Value: r}, &vm.Float{Value: l * r})
		}
	}

	// float / float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			if r != 0 {
				testBinaryOp(t, &vm.Float{Value: l}, token.Quo,
					&vm.Float{Value: r}, &vm.Float{Value: l / r})
			}
		}
	}

	// float < float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.Less,
				&vm.Float{Value: r}, boolValue(l < r))
		}
	}

	// float > float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.Greater,
				&vm.Float{Value: r}, boolValue(l > r))
		}
	}

	// float <= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.LessEq,
				&vm.Float{Value: r}, boolValue(l <= r))
		}
	}

	// float >= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &vm.Float{Value: l}, token.GreaterEq,
				&vm.Float{Value: r}, boolValue(l >= r))
		}
	}

	// float + int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.Add,
				&vm.Int{Value: r}, &vm.Float{Value: l + float64(r)})
		}
	}

	// float - int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.Sub,
				&vm.Int{Value: r}, &vm.Float{Value: l - float64(r)})
		}
	}

	// float * int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.Mul,
				&vm.Int{Value: r}, &vm.Float{Value: l * float64(r)})
		}
	}

	// float / int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, &vm.Float{Value: l}, token.Quo,
					&vm.Int{Value: r},
					&vm.Float{Value: l / float64(r)})
			}
		}
	}

	// float < int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.Less,
				&vm.Int{Value: r}, boolValue(l < float64(r)))
		}
	}

	// float > int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.Greater,
				&vm.Int{Value: r}, boolValue(l > float64(r)))
		}
	}

	// float <= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.LessEq,
				&vm.Int{Value: r}, boolValue(l <= float64(r)))
		}
	}

	// float >= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Float{Value: l}, token.GreaterEq,
				&vm.Int{Value: r}, boolValue(l >= float64(r)))
		}
	}
}

func TestInt_BinaryOp(t *testing.T) {
	// int + int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.Add,
				&vm.Int{Value: r}, &vm.Int{Value: l + r})
		}
	}

	// int - int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.Sub,
				&vm.Int{Value: r}, &vm.Int{Value: l - r})
		}
	}

	// int * int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.Mul,
				&vm.Int{Value: r}, &vm.Int{Value: l * r})
		}
	}

	// int / int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, &vm.Int{Value: l}, token.Quo,
					&vm.Int{Value: r}, &vm.Int{Value: l / r})
			}
		}
	}

	// int % int
	for l := int64(-4); l <= 4; l++ {
		for r := -int64(-4); r <= 4; r++ {
			if r == 0 {
				testBinaryOp(t, &vm.Int{Value: l}, token.Rem,
					&vm.Int{Value: r}, &vm.Int{Value: l % r})
			}
		}
	}

	// int & int
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.And, &vm.Int{Value: 0},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.And, &vm.Int{Value: 0},
		&vm.Int{Value: int64(1) & int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.And, &vm.Int{Value: 1},
		&vm.Int{Value: int64(0) & int64(1)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.And, &vm.Int{Value: 1},
		&vm.Int{Value: int64(1)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.And, &vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0) & int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.And, &vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1) & int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: int64(0xffffffff)}, token.And,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: 1984}, token.And,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1984) & int64(0xffffffff)})
	testBinaryOp(t, &vm.Int{Value: -1984}, token.And,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(-1984) & int64(0xffffffff)})

	// int | int
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.Or, &vm.Int{Value: 0},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.Or, &vm.Int{Value: 0},
		&vm.Int{Value: int64(1) | int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.Or, &vm.Int{Value: 1},
		&vm.Int{Value: int64(0) | int64(1)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.Or, &vm.Int{Value: 1},
		&vm.Int{Value: int64(1)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.Or, &vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0) | int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.Or, &vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1) | int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: int64(0xffffffff)}, token.Or,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: 1984}, token.Or,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1984) | int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: -1984}, token.Or,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(-1984) | int64(0xffffffff)})

	// int ^ int
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.Xor, &vm.Int{Value: 0},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.Xor, &vm.Int{Value: 0},
		&vm.Int{Value: int64(1) ^ int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.Xor, &vm.Int{Value: 1},
		&vm.Int{Value: int64(0) ^ int64(1)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.Xor, &vm.Int{Value: 1},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.Xor, &vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0) ^ int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.Xor, &vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1) ^ int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: int64(0xffffffff)}, token.Xor,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 1984}, token.Xor,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1984) ^ int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: -1984}, token.Xor,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(-1984) ^ int64(0xffffffff)})

	// int &^ int
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.AndNot, &vm.Int{Value: 0},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.AndNot, &vm.Int{Value: 0},
		&vm.Int{Value: int64(1) &^ int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.AndNot,
		&vm.Int{Value: 1}, &vm.Int{Value: int64(0) &^ int64(1)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.AndNot, &vm.Int{Value: 1},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 0}, token.AndNot,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0) &^ int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: 1}, token.AndNot,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1) &^ int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: int64(0xffffffff)}, token.AndNot,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(0)})
	testBinaryOp(t,
		&vm.Int{Value: 1984}, token.AndNot,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(1984) &^ int64(0xffffffff)})
	testBinaryOp(t,
		&vm.Int{Value: -1984}, token.AndNot,
		&vm.Int{Value: int64(0xffffffff)},
		&vm.Int{Value: int64(-1984) &^ int64(0xffffffff)})

	// int << int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			&vm.Int{Value: 0}, token.Shl, &vm.Int{Value: s},
			&vm.Int{Value: int64(0) << uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: 1}, token.Shl, &vm.Int{Value: s},
			&vm.Int{Value: int64(1) << uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: 2}, token.Shl, &vm.Int{Value: s},
			&vm.Int{Value: int64(2) << uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: -1}, token.Shl, &vm.Int{Value: s},
			&vm.Int{Value: int64(-1) << uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: -2}, token.Shl, &vm.Int{Value: s},
			&vm.Int{Value: int64(-2) << uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: int64(0xffffffff)}, token.Shl,
			&vm.Int{Value: s},
			&vm.Int{Value: int64(0xffffffff) << uint(s)})
	}

	// int >> int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			&vm.Int{Value: 0}, token.Shr, &vm.Int{Value: s},
			&vm.Int{Value: int64(0) >> uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: 1}, token.Shr, &vm.Int{Value: s},
			&vm.Int{Value: int64(1) >> uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: 2}, token.Shr, &vm.Int{Value: s},
			&vm.Int{Value: int64(2) >> uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: -1}, token.Shr, &vm.Int{Value: s},
			&vm.Int{Value: int64(-1) >> uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: -2}, token.Shr, &vm.Int{Value: s},
			&vm.Int{Value: int64(-2) >> uint(s)})
		testBinaryOp(t,
			&vm.Int{Value: int64(0xffffffff)}, token.Shr,
			&vm.Int{Value: s},
			&vm.Int{Value: int64(0xffffffff) >> uint(s)})
	}

	// int < int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.Less,
				&vm.Int{Value: r}, boolValue(l < r))
		}
	}

	// int > int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.Greater,
				&vm.Int{Value: r}, boolValue(l > r))
		}
	}

	// int <= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.LessEq,
				&vm.Int{Value: r}, boolValue(l <= r))
		}
	}

	// int >= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &vm.Int{Value: l}, token.GreaterEq,
				&vm.Int{Value: r}, boolValue(l >= r))
		}
	}

	// int + float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.Add,
				&vm.Float{Value: r},
				&vm.Float{Value: float64(l) + r})
		}
	}

	// int - float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.Sub,
				&vm.Float{Value: r},
				&vm.Float{Value: float64(l) - r})
		}
	}

	// int * float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.Mul,
				&vm.Float{Value: r},
				&vm.Float{Value: float64(l) * r})
		}
	}

	// int / float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			if r != 0 {
				testBinaryOp(t, &vm.Int{Value: l}, token.Quo,
					&vm.Float{Value: r},
					&vm.Float{Value: float64(l) / r})
			}
		}
	}

	// int < float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.Less,
				&vm.Float{Value: r}, boolValue(float64(l) < r))
		}
	}

	// int > float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.Greater,
				&vm.Float{Value: r}, boolValue(float64(l) > r))
		}
	}

	// int <= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.LessEq,
				&vm.Float{Value: r}, boolValue(float64(l) <= r))
		}
	}

	// int >= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &vm.Int{Value: l}, token.GreaterEq,
				&vm.Float{Value: r}, boolValue(float64(l) >= r))
		}
	}
}

func TestMap_Index(t *testing.T) {
	m := &vm.Map{Value: make(map[string]vm.Object)}
	k := &vm.Int{Value: 1}
	v := &vm.String{Value: "abcdef"}
	err := m.IndexSet(k, v)

	require.NoError(t, err)

	res, err := m.IndexGet(k)
	require.NoError(t, err)
	require.Equal(t, v, res)
}

func TestString_BinaryOp(t *testing.T) {
	lstr := "abcde"
	rstr := "01234"
	for l := 0; l < len(lstr); l++ {
		for r := 0; r < len(rstr); r++ {
			ls := lstr[l:]
			rs := rstr[r:]
			testBinaryOp(t, &vm.String{Value: ls}, token.Add,
				&vm.String{Value: rs},
				&vm.String{Value: ls + rs})

			rc := []rune(rstr)[r]
			testBinaryOp(t, &vm.String{Value: ls}, token.Add,
				&vm.Char{Value: rc},
				&vm.String{Value: ls + string(rc)})
		}
	}
}

func testBinaryOp(
	t *testing.T,
	lhs vm.Object,
	op token.Token,
	rhs vm.Object,
	expected vm.Object,
) {
	t.Helper()
	actual, err := lhs.BinaryOp(op, rhs)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func boolValue(b bool) vm.Object {
	if b {
		return vm.TrueValue
	}
	return vm.FalseValue
}

// Integer division and modulo by zero produce a clean, matchable sentinel error.
//
// When an integer division (/) or modulo (%) operation is performed with a
// zero right-hand operand, the Go runtime would normally panic with
// "runtime error: integer divide by zero". That panic is caught by the VM's
// top-level recover(), which converts it into an ErrPanic — a confusing
// multi-line message containing a full goroutine stack trace rather than a
// clean, actionable error.
//
// The fix checks for zero divisor in Int.BinaryOp (for both token.Quo and
// token.Rem) and returns vm.ErrDivisionByZero — a sentinel error that callers
// can match with errors.Is and that produces a clear "division by zero"
// message without any stack trace noise.

// TestIntDivisionByZeroSentinel verifies that Int / 0 returns the sentinel
// error ErrDivisionByZero, not an opaque fmt.Errorf or an ErrPanic.
func TestIntDivisionByZeroSentinel(t *testing.T) {
	a := &vm.Int{Value: 10}
	zero := &vm.Int{Value: 0}

	_, err := a.BinaryOp(token.Quo, zero)
	require.Error(t, err)
	require.True(t, errors.Is(err, vm.ErrDivisionByZero),
		"expected errors.Is(err, ErrDivisionByZero) to be true, got: %v (type %T)", err, err)
}

// TestIntModuloByZeroSentinel verifies that Int % 0 returns ErrDivisionByZero.
func TestIntModuloByZeroSentinel(t *testing.T) {
	a := &vm.Int{Value: 10}
	zero := &vm.Int{Value: 0}

	_, err := a.BinaryOp(token.Rem, zero)
	require.Error(t, err)
	require.True(t, errors.Is(err, vm.ErrDivisionByZero),
		"expected errors.Is(err, ErrDivisionByZero) to be true, got: %v (type %T)", err, err)
}

// TestIntDivisionByZeroNotPanic verifies that Int / 0 does NOT produce a
// runtime panic; the error is returned cleanly, not via Go's panic mechanism.
func TestIntDivisionByZeroNotPanic(t *testing.T) {
	a := &vm.Int{Value: 10}
	zero := &vm.Int{Value: 0}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Int / 0 must not panic, but got: %v", r)
		}
	}()

	_, err := a.BinaryOp(token.Quo, zero)
	require.Error(t, err)
}

// TestIntModuloByZeroNotPanic verifies that Int % 0 does NOT produce a
// runtime panic.
func TestIntModuloByZeroNotPanic(t *testing.T) {
	a := &vm.Int{Value: 10}
	zero := &vm.Int{Value: 0}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Int %% 0 must not panic, but got: %v", r)
		}
	}()

	_, err := a.BinaryOp(token.Rem, zero)
	require.Error(t, err)
}

// TestIntNegativeDivisionByZeroSentinel verifies negative dividend / 0 also
// returns ErrDivisionByZero (not a panic).
func TestIntNegativeDivisionByZeroSentinel(t *testing.T) {
	a := &vm.Int{Value: -42}
	zero := &vm.Int{Value: 0}

	_, err := a.BinaryOp(token.Quo, zero)
	require.Error(t, err)
	require.True(t, errors.Is(err, vm.ErrDivisionByZero),
		"expected ErrDivisionByZero, got: %v", err)

	_, err = a.BinaryOp(token.Rem, zero)
	require.Error(t, err)
	require.True(t, errors.Is(err, vm.ErrDivisionByZero),
		"expected ErrDivisionByZero, got: %v", err)
}

// TestIntZeroDivisionByZeroSentinel verifies 0 / 0 also returns ErrDivisionByZero.
func TestIntZeroDivisionByZeroSentinel(t *testing.T) {
	zero := &vm.Int{Value: 0}

	_, err := zero.BinaryOp(token.Quo, zero)
	require.Error(t, err)
	require.True(t, errors.Is(err, vm.ErrDivisionByZero),
		"expected ErrDivisionByZero, got: %v", err)
}

// Issue #25: Map and Array mutations are not thread-safe.
//
// IndexSet, builtinAppend, builtinDelete, and builtinSplice mutate the
// underlying Go map/slice directly without any synchronisation. When the same
// *Array or *Map object is shared between two concurrent routines (e.g. stored
// in a global that both child VMs received as a snapshot pointer), concurrent
// calls to these operations produce data races detected by -race and can crash
// the process with "concurrent map read and map write" or corrupt slice state.
//
// The fix adds a sync.RWMutex to both Array and Map structs and acquires the
// appropriate lock in every method that reads or writes the underlying
// value (IndexGet, IndexSet, Copy, String, IsFalsy, Equals, Iterate, BinaryOp).
// builtinDelete and builtinSplice, which bypass IndexSet to mutate the
// underlying data directly, are also updated to acquire the write lock.

// TestIssue25_ArrayIndexSetConcurrent verifies that concurrent IndexSet calls
// on the same *Array do not race. Without the fix this triggers the Go race
// detector and can corrupt slice state.
func TestIssue25_ArrayIndexSetConcurrent(t *testing.T) {
	const size = 100
	arr := &vm.Array{Value: make([]vm.Object, size)}
	for i := range arr.Value {
		arr.Value[i] = &vm.Int{Value: 0}
	}

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				idx := &vm.Int{Value: int64((g*500 + j) % size)}
				_ = arr.IndexSet(idx, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_ArrayIndexGetConcurrent verifies that concurrent IndexGet and
// IndexSet calls on the same *Array do not race.
func TestIssue25_ArrayIndexGetConcurrent(t *testing.T) {
	const size = 50
	arr := &vm.Array{Value: make([]vm.Object, size)}
	for i := range arr.Value {
		arr.Value[i] = &vm.Int{Value: int64(i)}
	}

	var wg sync.WaitGroup
	// writers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				idx := &vm.Int{Value: int64(j % size)}
				_ = arr.IndexSet(idx, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	// readers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_, _ = arr.IndexGet(&vm.Int{Value: int64(j % size)})
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_MapIndexSetConcurrent verifies that concurrent IndexSet calls on
// the same *Map do not race. Without the fix Go panics with "concurrent map
// writes" or the race detector fires.
func TestIssue25_MapIndexSetConcurrent(t *testing.T) {
	m := &vm.Map{Value: make(map[string]vm.Object)}

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				key := &vm.String{Value: fmt.Sprintf("key%d", j%20)}
				_ = m.IndexSet(key, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_MapIndexGetConcurrent verifies that concurrent IndexGet and
// IndexSet calls on the same *Map do not race.
func TestIssue25_MapIndexGetConcurrent(t *testing.T) {
	m := &vm.Map{Value: map[string]vm.Object{
		"a": &vm.Int{Value: 1},
		"b": &vm.Int{Value: 2},
	}}

	var wg sync.WaitGroup
	// writers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				key := &vm.String{Value: fmt.Sprintf("key%d", j%10)}
				_ = m.IndexSet(key, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	// readers
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				key := &vm.String{Value: fmt.Sprintf("key%d", j%10)}
				_, _ = m.IndexGet(key)
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_ArrayCopyConcurrent verifies that Copy() can be called
// concurrently with writes without data races.
func TestIssue25_ArrayCopyConcurrent(t *testing.T) {
	arr := &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 1}, &vm.Int{Value: 2}, &vm.Int{Value: 3},
	}}

	var wg sync.WaitGroup
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = arr.Copy()
			}
		}(g)
	}
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = arr.IndexSet(&vm.Int{Value: int64(j % 3)}, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_MapCopyConcurrent verifies that Map.Copy() can be called
// concurrently with IndexSet without data races.
func TestIssue25_MapCopyConcurrent(t *testing.T) {
	m := &vm.Map{Value: map[string]vm.Object{
		"x": &vm.Int{Value: 1},
	}}

	var wg sync.WaitGroup
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = m.Copy()
			}
		}(g)
	}
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				key := &vm.String{Value: fmt.Sprintf("k%d", j%5)}
				_ = m.IndexSet(key, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_ArrayIterateConcurrent verifies that creating an iterator from
// a *Array concurrent with writes does not race.
func TestIssue25_ArrayIterateConcurrent(t *testing.T) {
	arr := &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 0}, &vm.Int{Value: 1}, &vm.Int{Value: 2},
	}}

	var wg sync.WaitGroup
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				it := arr.Iterate()
				for it.Next() {
					_ = it.Value()
				}
			}
		}()
	}
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = arr.IndexSet(&vm.Int{Value: int64(j % 3)}, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	wg.Wait()
}

// TestIssue25_MapIterateConcurrent verifies that creating a map iterator
// concurrently with map writes does not race or panic.
func TestIssue25_MapIterateConcurrent(t *testing.T) {
	m := &vm.Map{Value: map[string]vm.Object{
		"a": &vm.Int{Value: 1},
		"b": &vm.Int{Value: 2},
	}}

	var wg sync.WaitGroup
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				it := m.Iterate()
				for it.Next() {
					_ = it.Key()
					_ = it.Value()
				}
			}
		}()
	}
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := &vm.String{Value: fmt.Sprintf("k%d", j%5)}
				_ = m.IndexSet(key, &vm.Int{Value: int64(j)})
			}
		}(g)
	}
	wg.Wait()
}
