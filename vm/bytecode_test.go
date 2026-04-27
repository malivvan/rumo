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

// A user-supplied ImmutableMap that happens to contain a "__module_name__" key
// must not be mistaken for a legitimately compiled builtin-module constant.
// Before the fix, fixDecodedObject replaced any ImmutableMap whose Value map
// contained {"__module_name__": "x"} with the real "x" builtin module,
// allowing either crafted bytecode or RemoveDuplicates cross-contamination to
// escalate script privileges.  After the fix the module name is tracked
// out-of-band (ImmutableMap.ModuleName), so user data with that string key
// is inert.
func TestImmutableMap_ModuleNameNotSpoofable(t *testing.T) {
	const modName = "testmod"

	// Build a real module with a recognisable sentinel attribute.
	realMod := vm.NewModuleMap()
	realMod.AddBuiltinModule(modName, map[string]vm.Object{
		"secret": &vm.String{Value: "REAL_SECRET"},
	})

	// Build a fake ImmutableMap that only has __module_name__ in its Value map
	// — exactly what user script code or crafted bytecode can produce.
	fakeMap := &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"__module_name__": &vm.String{Value: modName},
			"secret":          &vm.String{Value: "FAKE_VALUE"},
		},
	}

	// Wrap it in bytecode so we can round-trip through Marshal/Unmarshal.
	b := &vm.Bytecode{
		FileSet:      parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{Instructions: []byte{}},
		Constants:    []vm.Object{fakeMap},
	}

	data, err := b.Marshal()
	require.NoError(t, err)

	restored := &vm.Bytecode{}
	err = restored.Unmarshal(data, realMod)
	require.NoError(t, err)
	require.Equal(t, 1, len(restored.Constants))

	im, ok := restored.Constants[0].(*vm.ImmutableMap)
	require.True(t, ok, "constant must remain an ImmutableMap")

	// The fake map must NOT have been replaced by the real module.
	// Its "secret" attribute must still carry the fake value, not "REAL_SECRET".
	secretObj, exists := im.Value["secret"]
	require.True(t, exists, `"secret" key must still be present`)
	secretStr, ok := secretObj.(*vm.String)
	require.True(t, ok, `"secret" must be a *String`)
	require.Equal(t, "FAKE_VALUE", secretStr.Value,
		"user ImmutableMap must not be replaced by the real module")
}

// RemoveDuplicates must not conflate a legitimate module constant with a
// user-constructed ImmutableMap that carries the same __module_name__ value.
func TestRemoveDuplicates_ModuleNameIsolation(t *testing.T) {
	const modName = "mymod"

	// real module ImmutableMap (produced by AsImmutableMap)
	realMod := &vm.BuiltinModule{Attrs: map[string]vm.Object{
		"val": &vm.String{Value: "real"},
	}}
	realMap := realMod.AsImmutableMap(modName)

	// user-constructed ImmutableMap that spoofs the same module name
	fakeMap := &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"__module_name__": &vm.String{Value: modName},
			"val":             &vm.String{Value: "fake"},
		},
	}

	// Both constants are referenced from the main function.
	b := &vm.Bytecode{
		FileSet: parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{
			Instructions: concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0), // real module
				vm.MakeInstruction(parser.OpConstant, 1), // fake map
			),
		},
		Constants: []vm.Object{realMap, fakeMap},
	}

	b.RemoveDuplicates()

	// After deduplication the two constants must remain distinct: the fake map
	// must not have been collapsed onto the real module (or vice versa).
	require.Equal(t, 2, len(b.Constants),
		"RemoveDuplicates must keep real module and user map as separate constants")
}

// Issue 4.5: RemoveDuplicates previously skipped *Bytes and *Map constants
// entirely, passing every occurrence through as a distinct constant even when
// two embed directives embedded the same file content. The fix deduplicates
// *Bytes by SHA-256 of the raw content and *Map by a canonical content hash
// (sorted keys + marshalled values).

// TestRemoveDuplicates_BytesDedup verifies that two *Bytes constants with
// identical content are collapsed to a single constant after RemoveDuplicates,
// and that instructions referencing both point to the surviving constant.
func TestRemoveDuplicates_BytesDedup(t *testing.T) {
	data := []byte("hello, embedded world")

	b := &vm.Bytecode{
		FileSet: parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{
			Instructions: concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0), // first Bytes
				vm.MakeInstruction(parser.OpConstant, 1), // identical Bytes → should map to 0
			),
		},
		Constants: []vm.Object{
			&vm.Bytes{Value: append([]byte(nil), data...)},
			&vm.Bytes{Value: append([]byte(nil), data...)}, // same content, different pointer
		},
	}

	b.RemoveDuplicates()

	require.Equal(t, 1, len(b.Constants),
		"two identical Bytes constants must be collapsed to one")

	// Both OpConstant instructions must now reference index 0.
	insts := b.MainFunction.Instructions
	idx0 := int(insts[2]) | int(insts[1])<<8
	idx1 := int(insts[5]) | int(insts[4])<<8
	require.Equal(t, 0, idx0, "first OpConstant must point to index 0")
	require.Equal(t, 0, idx1, "second OpConstant must also point to index 0 after dedup")
}

// TestRemoveDuplicates_BytesDedupDistinct verifies that two *Bytes constants
// with different content are kept as distinct constants.
func TestRemoveDuplicates_BytesDedupDistinct(t *testing.T) {
	b := &vm.Bytecode{
		FileSet: parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{
			Instructions: concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
			),
		},
		Constants: []vm.Object{
			&vm.Bytes{Value: []byte("file-a.txt content")},
			&vm.Bytes{Value: []byte("file-b.txt content")},
		},
	}

	b.RemoveDuplicates()

	require.Equal(t, 2, len(b.Constants),
		"two distinct Bytes constants must remain as two constants")
}

// TestRemoveDuplicates_MapDedup verifies that two *Map constants with identical
// contents are collapsed to a single constant after RemoveDuplicates.
func TestRemoveDuplicates_MapDedup(t *testing.T) {
	makeMap := func() *vm.Map {
		return &vm.Map{Value: map[string]vm.Object{
			"a.txt": &vm.Bytes{Value: []byte("content of a")},
			"b.txt": &vm.Bytes{Value: []byte("content of b")},
		}}
	}

	b := &vm.Bytecode{
		FileSet: parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{
			Instructions: concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
			),
		},
		Constants: []vm.Object{makeMap(), makeMap()},
	}

	b.RemoveDuplicates()

	require.Equal(t, 1, len(b.Constants),
		"two identical Map constants must be collapsed to one")

	insts := b.MainFunction.Instructions
	idx0 := int(insts[2]) | int(insts[1])<<8
	idx1 := int(insts[5]) | int(insts[4])<<8
	require.Equal(t, 0, idx0)
	require.Equal(t, 0, idx1, "second OpConstant must point to index 0 after dedup")
}

// TestUserTypeRoundTrip verifies that a *UserType constant (produced by a
// `type` declaration) survives a full Marshal → Unmarshal round-trip and
// that the deserialized VM can still use the type to construct instances.
//
// This is a regression test for the panic "sizeof: unsupported type:
// type:struct:Person" that occurred because encoding.go had no case for
// *UserType in TypeOfObject / SizeOfObject / MarshalObject / UnmarshalObject.
func TestUserTypeRoundTrip(t *testing.T) {
	// Helper: compile → marshal → unmarshal → run, returning globals map.
	run := func(t *testing.T, src string, symbols map[string]vm.Object) map[string]vm.Object {
		t.Helper()

		fileSet := parser.NewFileSet()
		srcFile := fileSet.AddFile("test", -1, len(src))
		p := parser.NewParser(srcFile, []byte(src), nil)
		file, err := p.ParseFile()
		require.NoError(t, err)

		symTable := vm.NewSymbolTable()
		for idx, fn := range vm.GetAllBuiltinFunctions() {
			symTable.DefineBuiltin(idx, fn.Name)
		}
		for name := range symbols {
			symTable.Define(name)
		}

		c := vm.NewCompiler(srcFile, symTable, nil, nil, nil)
		require.NoError(t, c.Compile(file))

		bc := c.Bytecode()
		bc.RemoveDuplicates()

		// Marshal then unmarshal — this is where the panic used to occur.
		data, err := bc.Marshal()
		require.NoError(t, err, "Marshal must not panic or error for UserType constants")

		restored := &vm.Bytecode{}
		require.NoError(t, restored.Unmarshal(data, nil))

		globals := make([]vm.Object, vm.DefaultConfig.GlobalsSize)
		// Pre-populate symbols in globals in the same slot order as the symbol table.
		for name, val := range symbols {
			sym, depth, ok := symTable.Resolve(name, false)
			require.True(t, ok && depth == 0, "symbol %q not found", name)
			globals[sym.Index] = val
		}

		machine := vm.NewVM(context.Background(), restored, globals, nil)
		require.NoError(t, machine.Run())

		result := make(map[string]vm.Object, len(symbols))
		for name := range symbols {
			sym, depth, ok := symTable.Resolve(name, false)
			require.True(t, ok && depth == 0)
			result[name] = globals[sym.Index]
		}
		return result
	}

	t.Run("struct_type", func(t *testing.T) {
		src := `
type Point struct { x int; y int }
p := Point(3, 4)
out = p.x + p.y
`
		res := run(t, src, map[string]vm.Object{"out": &vm.Int{}})
		require.Equal(t, &vm.Int{Value: 7}, res["out"])
	})

	t.Run("func_type", func(t *testing.T) {
		src := `
type Adder func(a int, b int) int
h := Adder(func(a, b) { return a + b })
out = h(10, 20)
`
		res := run(t, src, map[string]vm.Object{"out": &vm.Int{}})
		require.Equal(t, &vm.Int{Value: 30}, res["out"])
	})

	t.Run("value_type", func(t *testing.T) {
		src := `
type MyInt int
n := MyInt("42")
out = n
`
		res := run(t, src, map[string]vm.Object{"out": &vm.Int{}})
		require.Equal(t, &vm.Int{Value: 42}, res["out"])
	})

	t.Run("struct_with_string_fields", func(t *testing.T) {
		// Regression: the original panic occurred with the testdata/type/main.rumo
		// file which uses struct types with string fields.
		src := `
type Person struct { Name string; Age int }
p := Person("Alice", 30)
out = p.Name
`
		res := run(t, src, map[string]vm.Object{"out": &vm.String{}})
		require.Equal(t, &vm.String{Value: "Alice"}, res["out"])
	})

	t.Run("direct_bytecode_object_roundtrip", func(t *testing.T) {
		// Directly construct a Bytecode with a *UserType constant and verify
		// it round-trips through Marshal/Unmarshal without error.
		ut := &vm.UserType{
			Name:       "Person",
			Kind:       vm.UserTypeStruct,
			Fields:     []string{"Name", "Age"},
			FieldTypes: []string{"string", "int"},
		}
		funcType := &vm.UserType{
			Name:       "Handler",
			Kind:       vm.UserTypeFunc,
			Params:     []string{"req", "resp"},
			ParamTypes: []string{"string", "string"},
			NumParams:  2,
			Result:     "int",
		}
		valueType := &vm.UserType{
			Name:       "MyInt",
			Kind:       vm.UserTypeValue,
			Underlying: "int",
		}

		b := &vm.Bytecode{
			FileSet:      parser.NewFileSet(),
			MainFunction: &vm.CompiledFunction{Instructions: []byte{}},
			Constants:    []vm.Object{ut, funcType, valueType},
		}

		data, err := b.Marshal()
		require.NoError(t, err)

		restored := &vm.Bytecode{}
		require.NoError(t, restored.Unmarshal(data, nil))
		require.Equal(t, 3, len(restored.Constants))

		// Struct UserType
		rut, ok := restored.Constants[0].(*vm.UserType)
		require.True(t, ok, "first constant must be *UserType")
		require.Equal(t, "Person", rut.Name)
		require.True(t, rut.Kind == vm.UserTypeStruct, "Kind must be UserTypeStruct")
		require.Equal(t, 2, len(rut.Fields))
		require.Equal(t, "Name", rut.Fields[0])
		require.Equal(t, "Age", rut.Fields[1])
		require.Equal(t, 2, len(rut.FieldTypes))
		require.Equal(t, "string", rut.FieldTypes[0])
		require.Equal(t, "int", rut.FieldTypes[1])

		// Func UserType
		rfunc, ok := restored.Constants[1].(*vm.UserType)
		require.True(t, ok, "second constant must be *UserType")
		require.Equal(t, "Handler", rfunc.Name)
		require.True(t, rfunc.Kind == vm.UserTypeFunc, "Kind must be UserTypeFunc")
		require.Equal(t, 2, len(rfunc.Params))
		require.Equal(t, "req", rfunc.Params[0])
		require.Equal(t, "resp", rfunc.Params[1])
		require.Equal(t, 2, len(rfunc.ParamTypes))
		require.Equal(t, "string", rfunc.ParamTypes[0])
		require.Equal(t, "string", rfunc.ParamTypes[1])
		require.Equal(t, 2, rfunc.NumParams)
		require.Equal(t, "int", rfunc.Result)

		// Value UserType
		rval, ok := restored.Constants[2].(*vm.UserType)
		require.True(t, ok, "third constant must be *UserType")
		require.Equal(t, "MyInt", rval.Name)
		require.True(t, rval.Kind == vm.UserTypeValue, "Kind must be UserTypeValue")
		require.Equal(t, "int", rval.Underlying)
	})
}

// TestRemoveDuplicates_MapDedupDistinct verifies that two *Map constants
// with different contents remain distinct after RemoveDuplicates.
func TestRemoveDuplicates_MapDedupDistinct(t *testing.T) {
	b := &vm.Bytecode{
		FileSet: parser.NewFileSet(),
		MainFunction: &vm.CompiledFunction{
			Instructions: concatInsts(
				vm.MakeInstruction(parser.OpConstant, 0),
				vm.MakeInstruction(parser.OpConstant, 1),
			),
		},
		Constants: []vm.Object{
			&vm.Map{Value: map[string]vm.Object{
				"a.txt": &vm.Bytes{Value: []byte("content of a")},
			}},
			&vm.Map{Value: map[string]vm.Object{
				"b.txt": &vm.Bytes{Value: []byte("content of b")},
			}},
		},
	}

	b.RemoveDuplicates()

	require.Equal(t, 2, len(b.Constants),
		"two distinct Map constants must remain as two constants")
}

