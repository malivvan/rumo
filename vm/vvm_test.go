package vm_test

import (
	"strings"
	"testing"
	"time"

	"github.com/malivvan/vv/vm"
	"github.com/malivvan/vv/vm/parser"
	"github.com/malivvan/vv/vm/require"
)

func TestInstructions_String(t *testing.T) {
	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(parser.OpConstant, 1),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpConstant, 65535),
		},
		`0000 CONST   1    
0003 CONST   2    
0006 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(parser.OpBinaryOp, 11),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpConstant, 65535),
		},
		`0000 BINARYOP 11   
0002 CONST   2    
0005 CONST   65535`)

	assertInstructionString(t,
		[][]byte{
			vm.MakeInstruction(parser.OpBinaryOp, 11),
			vm.MakeInstruction(parser.OpGetLocal, 1),
			vm.MakeInstruction(parser.OpConstant, 2),
			vm.MakeInstruction(parser.OpConstant, 65535),
		},
		`0000 BINARYOP 11   
0002 GETL    1    
0004 CONST   2    
0007 CONST   65535`)
}

func TestMakeInstruction(t *testing.T) {
	makeInstruction(t, []byte{parser.OpConstant, 0, 0},
		parser.OpConstant, 0)
	makeInstruction(t, []byte{parser.OpConstant, 0, 1},
		parser.OpConstant, 1)
	makeInstruction(t, []byte{parser.OpConstant, 255, 254},
		parser.OpConstant, 65534)
	makeInstruction(t, []byte{parser.OpPop}, parser.OpPop)
	makeInstruction(t, []byte{parser.OpTrue}, parser.OpTrue)
	makeInstruction(t, []byte{parser.OpFalse}, parser.OpFalse)
}

func TestNumObjects(t *testing.T) {
	testCountObjects(t, &vm.Array{}, 1)
	testCountObjects(t, &vm.Array{Value: []vm.Object{
		&vm.Int{Value: 1},
		&vm.Int{Value: 2},
		&vm.Array{Value: []vm.Object{
			&vm.Int{Value: 3},
			&vm.Int{Value: 4},
			&vm.Int{Value: 5},
		}},
	}}, 7)
	testCountObjects(t, vm.TrueValue, 1)
	testCountObjects(t, vm.FalseValue, 1)
	testCountObjects(t, &vm.BuiltinFunction{}, 1)
	testCountObjects(t, &vm.Bytes{Value: []byte("foobar")}, 1)
	testCountObjects(t, &vm.Char{Value: '가'}, 1)
	testCountObjects(t, &vm.CompiledFunction{}, 1)
	testCountObjects(t, &vm.Error{Value: &vm.Int{Value: 5}}, 2)
	testCountObjects(t, &vm.Float{Value: 19.84}, 1)
	testCountObjects(t, &vm.ImmutableArray{Value: []vm.Object{
		&vm.Int{Value: 1},
		&vm.Int{Value: 2},
		&vm.ImmutableArray{Value: []vm.Object{
			&vm.Int{Value: 3},
			&vm.Int{Value: 4},
			&vm.Int{Value: 5},
		}},
	}}, 7)
	testCountObjects(t, &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"k1": &vm.Int{Value: 1},
			"k2": &vm.Int{Value: 2},
			"k3": &vm.Array{Value: []vm.Object{
				&vm.Int{Value: 3},
				&vm.Int{Value: 4},
				&vm.Int{Value: 5},
			}},
		}}, 7)
	testCountObjects(t, &vm.Int{Value: 1984}, 1)
	testCountObjects(t, &vm.Map{Value: map[string]vm.Object{
		"k1": &vm.Int{Value: 1},
		"k2": &vm.Int{Value: 2},
		"k3": &vm.Array{Value: []vm.Object{
			&vm.Int{Value: 3},
			&vm.Int{Value: 4},
			&vm.Int{Value: 5},
		}},
	}}, 7)
	testCountObjects(t, &vm.String{Value: "foo bar"}, 1)
	testCountObjects(t, &vm.Time{Value: time.Now()}, 1)
	testCountObjects(t, vm.UndefinedValue, 1)
}

func testCountObjects(t *testing.T, o vm.Object, expected int) {
	require.Equal(t, expected, vm.CountObjects(o))
}

func assertInstructionString(
	t *testing.T,
	instructions [][]byte,
	expected string,
) {
	concatted := make([]byte, 0)
	for _, e := range instructions {
		concatted = append(concatted, e...)
	}
	require.Equal(t, expected, strings.Join(
		vm.FormatInstructions(concatted, 0), "\n"))
}

func makeInstruction(
	t *testing.T,
	expected []byte,
	opcode parser.Opcode,
	operands ...int,
) {
	inst := vm.MakeInstruction(opcode, operands...)
	require.Equal(t, expected, inst)
}
