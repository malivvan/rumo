package vm_test

import (
	"context"
	"testing"
	"time"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/parser"
	"github.com/malivvan/rumo/vm/require"
)

type srcfile struct {
	name string
	size int
}

func TestBytecode(t *testing.T) {
	testBytecodeSerialization(t, bytecode(concatInsts(), objectsArray()))

	testBytecodeSerialization(t, bytecode(
		concatInsts(), objectsArray(
			&vm.Char{Value: 'y'},
			&vm.Float{Value: 93.11},
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpGetFree, 0)),
			&vm.Float{Value: 39.2},
			&vm.Int{Value: 192},
			&vm.String{Value: "bar"})))

	testBytecodeSerialization(t, bytecodeFileSet(
		concatInsts(
			vm.MakeInstruction(parser.OpConstant, 0),
			vm.MakeInstruction(parser.OpSetGlobal, 0),
			vm.MakeInstruction(parser.OpConstant, 6),
			vm.MakeInstruction(parser.OpPop)),
		objectsArray(
			&vm.Int{Value: 55},
			&vm.Int{Value: 66},
			&vm.Int{Value: 77},
			&vm.Int{Value: 88},
			&vm.ImmutableMap{
				Value: map[string]vm.Object{
					"array": &vm.ImmutableArray{
						Value: []vm.Object{
							&vm.Int{Value: 1},
							&vm.Int{Value: 2},
							&vm.Int{Value: 3},
							vm.TrueValue,
							vm.FalseValue,
							vm.UndefinedValue,
						},
					},
					"true":  vm.TrueValue,
					"false": vm.FalseValue,
					"bytes": &vm.Bytes{Value: make([]byte, 16)},
					"char":  &vm.Char{Value: 'Y'},
					"error": &vm.Error{Value: &vm.String{
						Value: "some error",
					}},
					"float": &vm.Float{Value: -19.84},
					"immutable_array": &vm.ImmutableArray{
						Value: []vm.Object{
							&vm.Int{Value: 1},
							&vm.Int{Value: 2},
							&vm.Int{Value: 3},
							vm.TrueValue,
							vm.FalseValue,
							vm.UndefinedValue,
						},
					},
					"immutable_map": &vm.ImmutableMap{
						Value: map[string]vm.Object{
							"a": &vm.Int{Value: 1},
							"b": &vm.Int{Value: 2},
							"c": &vm.Int{Value: 3},
							"d": vm.TrueValue,
							"e": vm.FalseValue,
							"f": vm.UndefinedValue,
						},
					},
					"int": &vm.Int{Value: 91},
					"map": &vm.Map{
						Value: map[string]vm.Object{
							"a": &vm.Int{Value: 1},
							"b": &vm.Int{Value: 2},
							"c": &vm.Int{Value: 3},
							"d": vm.TrueValue,
							"e": vm.FalseValue,
							"f": vm.UndefinedValue,
						},
					},
					"string":    &vm.String{Value: "foo bar"},
					"time":      &vm.Time{Value: time.Now()},
					"undefined": vm.UndefinedValue,
				},
			},
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetGlobal, 0),
				vm.MakeInstruction(parser.OpGetFree, 0),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpGetFree, 1),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpBinaryOp, 11),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetFree, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpClosure, 4, 2),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpSetLocal, 0),
				vm.MakeInstruction(parser.OpGetLocal, 0),
				vm.MakeInstruction(parser.OpClosure, 5, 1),
				vm.MakeInstruction(parser.OpReturn, 1))),
		fileSet(srcfile{name: "file1", size: 100},
			srcfile{name: "file2", size: 200})))
}

func TestBytecode_RemoveDuplicates(t *testing.T) {
	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(), objectsArray(
				&vm.Char{Value: 'y'},
				&vm.Float{Value: 93.11},
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)),
				&vm.Float{Value: 39.2},
				&vm.Int{Value: 192},
				&vm.String{Value: "bar"})),
		bytecode(
			concatInsts(), objectsArray(
				&vm.Char{Value: 'y'},
				&vm.Float{Value: 93.11},
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)),
				&vm.Float{Value: 39.2},
				&vm.Int{Value: 192},
				&vm.String{Value: "bar"})))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 5),
				vm.MakeInstruction(parser.OpConstant, 6),
				vm.MakeInstruction(parser.OpConstant, 7),
				vm.MakeInstruction(parser.OpConstant, 8),
				vm.MakeInstruction(parser.OpClosure, 4, 1)),
			objectsArray(
				&vm.Int{Value: 1},
				&vm.Float{Value: 2.0},
				&vm.Char{Value: '3'},
				&vm.String{Value: "four"},
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpConstant, 7),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)),
				&vm.Int{Value: 1},
				&vm.Float{Value: 2.0},
				&vm.Char{Value: '3'},
				&vm.String{Value: "four"})),
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpClosure, 4, 1)),
			objectsArray(
				&vm.Int{Value: 1},
				&vm.Float{Value: 2.0},
				&vm.Char{Value: '3'},
				&vm.String{Value: "four"},
				compiledFunction(1, 0,
					vm.MakeInstruction(parser.OpConstant, 3),
					vm.MakeInstruction(parser.OpConstant, 2),
					vm.MakeInstruction(parser.OpSetLocal, 0),
					vm.MakeInstruction(parser.OpGetGlobal, 0),
					vm.MakeInstruction(parser.OpGetFree, 0)))))

	testBytecodeRemoveDuplicates(t,
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpConstant, 4)),
			objectsArray(
				&vm.Int{Value: 1},
				&vm.Int{Value: 2},
				&vm.Int{Value: 3},
				&vm.Int{Value: 1},
				&vm.Int{Value: 3})),
		bytecode(
			concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 2)),
			objectsArray(
				&vm.Int{Value: 1},
				&vm.Int{Value: 2},
				&vm.Int{Value: 3})))
}

func TestBytecode_CountObjects(t *testing.T) {
	b := bytecode(
		concatInsts(),
		objectsArray(
			&vm.Int{Value: 55},
			&vm.Int{Value: 66},
			&vm.Int{Value: 77},
			&vm.Int{Value: 88},
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 3),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 2),
				vm.MakeInstruction(parser.OpReturn, 1)),
			compiledFunction(1, 0,
				vm.MakeInstruction(parser.OpConstant, 1),
				vm.MakeInstruction(parser.OpReturn, 1))))
	require.Equal(t, 7, b.CountObjects())
}

func fileSet(files ...srcfile) *parser.SourceFileSet {
	fileSet := parser.NewFileSet()
	for _, f := range files {
		fileSet.AddFile(f.name, -1, f.size)
	}
	return fileSet
}

func bytecodeFileSet(
	instructions []byte,
	constants []vm.Object,
	fileSet *parser.SourceFileSet,
) *vm.Bytecode {
	return &vm.Bytecode{
		FileSet:      fileSet,
		MainFunction: &vm.CompiledFunction{Instructions: instructions},
		Constants:    constants,
	}
}

func testBytecodeRemoveDuplicates(
	t *testing.T,
	input, expected *vm.Bytecode,
) {
	input.RemoveDuplicates()

	require.Equal(t, expected.FileSet, input.FileSet)
	require.Equal(t, expected.MainFunction, input.MainFunction)
	require.Equal(t, expected.Constants, input.Constants)
}

func testBytecodeSerialization(t *testing.T, b *vm.Bytecode) {
	bc, err := b.Marshal()
	require.NoError(t, err)

	r := &vm.Bytecode{}
	err = r.Unmarshal(bc, nil)
	require.NoError(t, err)

	require.Equal(t, b.FileSet, r.FileSet)
	require.Equal(t, b.MainFunction, r.MainFunction)
	require.Equal(t, b.Constants, r.Constants)
}

// TestBuiltinFunctionRoundTrip verifies that a BuiltinFunction serialized into
// bytecode constants is fully restored after unmarshal — including its callable
// Value field.  Before the fix, UnmarshalObject only restored the Name; the Go
// function pointer (Value) was left nil, causing a nil-pointer panic on the
// first Call.
func TestBuiltinFunctionRoundTrip(t *testing.T) {
	// Pick a known builtin to round-trip.
	builtins := vm.GetAllBuiltinFunctions()
	require.True(t, len(builtins) > 0, "expected at least one builtin function")

	for _, original := range builtins {
		t.Run(original.Name, func(t *testing.T) {
			// Build minimal bytecode that holds the builtin as a constant.
			b := &vm.Bytecode{
				FileSet:      parser.NewFileSet(),
				MainFunction: &vm.CompiledFunction{Instructions: []byte{}},
				Constants:    []vm.Object{original},
			}

			data, err := b.Marshal()
			require.NoError(t, err)

			restored := &vm.Bytecode{}
			err = restored.Unmarshal(data, nil)
			require.NoError(t, err)

			require.Equal(t, 1, len(restored.Constants),
				"expected one constant after unmarshal")

			bf, ok := restored.Constants[0].(*vm.BuiltinFunction)
			require.True(t, ok, "expected *BuiltinFunction constant")
			require.Equal(t, original.Name, bf.Name)
			require.True(t, bf.CanCall(), "deserialized builtin must be callable")

			// The Value field must be non-nil; before the fix it was always nil
			// after deserialization, causing a nil-pointer panic on Call.
			require.True(t, bf.Value != nil,
				"deserialized builtin Value must not be nil")
		})
	}
}

// TestBuiltinFunctionRoundTripUnknownName verifies that unmarshaling bytecode
// containing a BuiltinFunction with an unrecognised name returns an error
// rather than silently producing a non-functional object.
func TestBuiltinFunctionRoundTripUnknownName(t *testing.T) {
	// Manually construct bytes that encode a BuiltinFunction with a bogus name.
	b := &vm.Bytecode{
		FileSet:      parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{Instructions: []byte{}},
		Constants: []vm.Object{
			&vm.BuiltinFunction{Name: "__no_such_builtin__", Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				return vm.UndefinedValue, nil
			}},
		},
	}

	data, err := b.Marshal()
	require.NoError(t, err)

	restored := &vm.Bytecode{}
	err = restored.Unmarshal(data, nil)
	require.Error(t, err, "expected error for unknown builtin name")
}

