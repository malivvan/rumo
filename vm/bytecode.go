package vm

import (
	"fmt"

	"github.com/malivvan/rumo/vm/codec"
	"github.com/malivvan/rumo/vm/parser"
)

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *parser.SourceFileSet
	MainFunction *CompiledFunction
	Constants    []Object
}

// Equals compares two Bytecode instances for equality.
func (b *Bytecode) Equals(other *Bytecode) bool {
	if b == nil || other == nil {
		return b == other
	}
	if !b.FileSet.Equals(other.FileSet) {
		return false
	}
	f1 := FormatInstructions(b.MainFunction.Instructions, 0)
	f2 := FormatInstructions(other.MainFunction.Instructions, 0)
	if len(f1) != len(f2) {
		return false
	}
	for i, l1 := range f1 {
		if l1 != f2[i] {
			return false
		}
	}
	if len(b.Constants) != len(other.Constants) {
		return false
	}
	for i, c := range b.Constants {
		if !c.Equals(other.Constants[i]) {
			return false
		}
	}
	return true
}

// Marshal writes Bytecode data to the writer.
func (b *Bytecode) Marshal() ([]byte, error) {
	n := 0
	c := make([]byte, parser.SizeFileSet(b.FileSet)+SizeOfObject(b.MainFunction)+codec.SizeSlice[Object](b.Constants, SizeOfObject))
	n = parser.MarshalFileSet(n, c, b.FileSet)
	n = MarshalObject(n, c, b.MainFunction)
	n = codec.MarshalSlice(n, c, b.Constants, MarshalObject)
	if n != len(c) {
		return nil, fmt.Errorf("encoded length mismatch: %d != %d", n, len(c))
	}
	return c, nil
}

// CountObjects returns the number of objects found in Constants.
func (b *Bytecode) CountObjects() int {
	n := 0
	for _, c := range b.Constants {
		n += CountObjects(c)
	}
	return n
}

// FormatInstructions returns human readable string representations of
// compiled instructions.
func (b *Bytecode) FormatInstructions() []string {
	return FormatInstructions(b.MainFunction.Instructions, 0)
}

// FormatConstants returns human readable string representations of
// compiled constants.
func (b *Bytecode) FormatConstants() (output []string) {
	for cidx, cn := range b.Constants {
		switch cn := cn.(type) {
		case *CompiledFunction:
			output = append(output, fmt.Sprintf(
				"[% 3d] (Compiled Function|%p)", cidx, &cn))
			for _, l := range FormatInstructions(cn.Instructions, 0) {
				output = append(output, fmt.Sprintf("     %s", l))
			}
		default:
			output = append(output, fmt.Sprintf("[% 3d] %s (%p)", cidx, cn, &cn))
		}
	}
	return
}

// Unmarshal decodes Bytecode from the given data.
func (b *Bytecode) Unmarshal(data []byte, modules *ModuleMap) (err error) {
	if modules == nil {
		modules = NewModuleMap()
	}

	n := 0
	n, b.FileSet, err = parser.UnmarshalFileSet(n, data)
	if err != nil {

		return err
	}

	var mainFuncObj Object
	n, mainFuncObj, err = UnmarshalObject(n, data)
	if err != nil {
		return err
	}
	mainFunc, ok := mainFuncObj.(*CompiledFunction)
	if !ok {
		return fmt.Errorf("main function is not a compiled function")
	}
	b.MainFunction = mainFunc

	n, b.Constants, err = codec.UnmarshalSlice[Object](n, data, UnmarshalObject)
	if err != nil {
		return err
	}

	for i, v := range b.Constants {
		fv, err := fixDecodedObject(v, modules)
		if err != nil {
			return err
		}
		b.Constants[i] = fv
	}

	if len(b.Constants) == 0 {
		b.Constants = nil
	}

	return nil
}

// RemoveDuplicates finds and remove the duplicate values in Constants.
// Note this function mutates Bytecode.
func (b *Bytecode) RemoveDuplicates() {
	var deduped []Object

	indexMap := make(map[int]int) // mapping from old constant index to new index
	fns := make(map[*CompiledFunction]int)
	ints := make(map[int64]int)
	strings := make(map[string]int)
	floats32 := make(map[float32]int)
	floats64 := make(map[float64]int)
	chars := make(map[rune]int)
	immutableMaps := make(map[string]int) // for modules

	for curIdx, c := range b.Constants {
		switch c := c.(type) {
		case *CompiledFunction:
			if newIdx, ok := fns[c]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				fns[c] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *ImmutableMap:
			modName := inferModuleName(c)
			newIdx, ok := immutableMaps[modName]
			if modName != "" && ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				immutableMaps[modName] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *Int:
			if newIdx, ok := ints[c.Value]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				ints[c.Value] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *String:
			if newIdx, ok := strings[c.Value]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				strings[c.Value] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *Float32:
			if newIdx, ok := floats32[c.Value]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				floats32[c.Value] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *Float64:
			if newIdx, ok := floats64[c.Value]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				floats64[c.Value] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *Char:
			if newIdx, ok := chars[c.Value]; ok {
				indexMap[curIdx] = newIdx
			} else {
				newIdx = len(deduped)
				chars[c.Value] = newIdx
				indexMap[curIdx] = newIdx
				deduped = append(deduped, c)
			}
		case *Bytes:
			// Bytes constants (from embed directives) cannot be deduplicated by
			// value without an expensive comparison; pass them through as-is.
			newIdx := len(deduped)
			indexMap[curIdx] = newIdx
			deduped = append(deduped, c)
		case *Map:
			// Map constants (from embed directives) are always unique; pass through.
			newIdx := len(deduped)
			indexMap[curIdx] = newIdx
			deduped = append(deduped, c)
		case *Native:
			// Native loader constants carry per-statement bindings and a
			// lazily-populated runtime handle; never attempt to share them.
			newIdx := len(deduped)
			indexMap[curIdx] = newIdx
			deduped = append(deduped, c)
		case *UserType:
			// User-defined types are introduced once per `type` statement and
			// referenced by pointer identity; always pass through.
			newIdx := len(deduped)
			indexMap[curIdx] = newIdx
			deduped = append(deduped, c)
		default:
			panic(fmt.Errorf("unsupported top-level constant type: %s",
				c.TypeName()))
		}
	}

	// replace with de-duplicated constants
	b.Constants = deduped

	// update CONST instructions with new indexes
	// main function
	updateConstIndexes(b.MainFunction.Instructions, indexMap)
	// other compiled functions in constants
	for _, c := range b.Constants {
		switch c := c.(type) {
		case *CompiledFunction:
			updateConstIndexes(c.Instructions, indexMap)
		}
	}
}

func fixDecodedObject(
	o Object,
	modules *ModuleMap,
) (Object, error) {
	switch o := o.(type) {
	case *BuiltinFunction:
		for _, bf := range builtinFuncs {
			if bf.Name == o.Name {
				return bf, nil
			}
		}
		return nil, fmt.Errorf("unknown builtin function: %q", o.Name)
	case *Bool:
		if o.IsFalsy() {
			return FalseValue, nil
		}
		return TrueValue, nil
	case *Undefined:
		return UndefinedValue, nil
	case *Array:
		for i, v := range o.Value {
			fv, err := fixDecodedObject(v, modules)
			if err != nil {
				return nil, err
			}
			o.Value[i] = fv
		}
	case *ImmutableArray:
		for i, v := range o.Value {
			fv, err := fixDecodedObject(v, modules)
			if err != nil {
				return nil, err
			}
			o.Value[i] = fv
		}
	case *Map:
		for k, v := range o.Value {
			fv, err := fixDecodedObject(v, modules)
			if err != nil {
				return nil, err
			}
			o.Value[k] = fv
		}
	case *ImmutableMap:
		modName := inferModuleName(o)
		if mod := modules.GetBuiltinModule(modName); mod != nil {
			return mod.AsImmutableMap(modName), nil
		}

		for k, v := range o.Value {

			fv, err := fixDecodedObject(v, modules)
			if err != nil {
				return nil, err
			}
			o.Value[k] = fv
		}
	}
	return o, nil
}

func updateConstIndexes(insts []byte, indexMap map[int]int) {
	i := 0
	for i < len(insts) {
		op := insts[i]
		numOperands := parser.OpcodeOperands[op]
		_, read := parser.ReadOperands(numOperands, insts[i+1:])

		switch op {
		case parser.OpConstant:
			curIdx := int(insts[i+2]) | int(insts[i+1])<<8
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx))
		case parser.OpClosure:
			curIdx := int(insts[i+2]) | int(insts[i+1])<<8
			numFree := int(insts[i+3])
			newIdx, ok := indexMap[curIdx]
			if !ok {
				panic(fmt.Errorf("constant index not found: %d", curIdx))
			}
			copy(insts[i:], MakeInstruction(op, newIdx, numFree))
		}

		i += 1 + read
	}
}

func inferModuleName(mod *ImmutableMap) string {
	return mod.moduleName
}
