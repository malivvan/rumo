//go:build native

package vm

import (
	"github.com/malivvan/rumo/vm/parser"
)

// compileNative lowers a `native name = "path" { ... }` statement into a
// constant push + OpCall + variable definition.  The constant is a *Native
// loader object; invoking it at runtime performs the dlopen and returns a
// *Map containing one callable per declared symbol.
func (c *Compiler) compileNative(node *parser.NativeStmt) error {
	if node.Name == nil {
		return c.errorf(node, "native: missing binding name")
	}
	if node.Path == "" {
		return c.errorf(node, "native: empty library path")
	}

	// First pass: collect struct declarations into specs and a name->index
	// table so func signatures can reference them by name.
	structIdx := make(map[string]int, len(node.Structs))
	structs := make([]NativeStructSpec, 0, len(node.Structs))
	for _, sd := range node.Structs {
		if sd == nil || sd.Name == nil {
			continue
		}
		sname := sd.Name.Name
		if sname == "" {
			continue
		}
		if _, dup := structIdx[sname]; dup {
			return c.errorf(node, "native: duplicate struct declaration %q", sname)
		}
		// Reserve the index up front so fields may reference siblings
		// declared later (forward references).
		structIdx[sname] = len(structs)
		structs = append(structs, NativeStructSpec{Name: sname})
	}
	// Second pass over struct fields once names are known.
	for _, sd := range node.Structs {
		if sd == nil || sd.Name == nil {
			continue
		}
		idx, ok := structIdx[sd.Name.Name]
		if !ok {
			continue
		}
		fields := make([]NativeStructFieldSpec, 0, len(sd.Fields))
		seenField := make(map[string]struct{}, len(sd.Fields))
		for _, fd := range sd.Fields {
			if fd == nil || fd.Name == nil || fd.Type == nil || fd.Type.Name == nil {
				continue
			}
			fname := fd.Name.Name
			if _, dup := seenField[fname]; dup {
				return c.errorf(node, "native: duplicate field %q in struct %q", fname, sd.Name.Name)
			}
			seenField[fname] = struct{}{}

			fSpec := NativeStructFieldSpec{
				Name:      fname,
				StructIdx: -1,
				Pointer:   fd.Type.Pointer,
			}
			if k, ok := nativeKindByName(fd.Type.Name.Name); ok {
				if k == NativeVoid {
					return c.errorf(node, "native: 'void' is not allowed as a field type in struct %q", sd.Name.Name)
				}
				if fd.Type.Pointer {
					return c.errorf(node, "native: pointer fields are not supported (struct %q field %q)", sd.Name.Name, fname)
				}
				fSpec.Kind = k
			} else if si, ok := structIdx[fd.Type.Name.Name]; ok {
				if fd.Type.Pointer {
					return c.errorf(node, "native: pointer-to-struct fields are not supported (struct %q field %q)", sd.Name.Name, fname)
				}
				fSpec.Kind = NativeStruct
				fSpec.StructIdx = si
			} else {
				return c.errorf(node,
					"native: unknown field type %q in struct %q",
					fd.Type.Name.Name, sd.Name.Name)
			}
			fields = append(fields, fSpec)
		}
		structs[idx].Fields = fields
	}

	// resolveTypeRef converts a parser NativeTypeRef into the corresponding
	// (NativeKind, structIndex, pointerFlag) triple, validating against the
	// current set of declared structs and built-in scalar types.
	resolveTypeRef := func(ref *parser.NativeTypeRef, allowVoid bool, fname string, what string) (NativeKind, int, bool, error) {
		if ref == nil || ref.Name == nil {
			return NativeInvalid, -1, false, c.errorf(node, "native: missing %s type in function %q", what, fname)
		}
		nm := ref.Name.Name
		if k, ok := nativeKindByName(nm); ok {
			if k == NativeVoid && !allowVoid {
				return NativeInvalid, -1, false, c.errorf(node,
					"native: 'void' is not allowed as a %s type in function %q", what, fname)
			}
			if ref.Pointer {
				return NativeInvalid, -1, false, c.errorf(node,
					"native: '*%s' is not a valid type (use 'ptr' for raw pointers) in function %q", nm, fname)
			}
			return k, -1, false, nil
		}
		if si, ok := structIdx[nm]; ok {
			return NativeStruct, si, ref.Pointer, nil
		}
		return NativeInvalid, -1, false, c.errorf(node,
			"native: unknown %s type %q in function %q",
			what, nm, fname)
	}

	// Translate each declared function into its internal spec form.
	specs := make([]NativeFuncSpec, 0, len(node.Funcs))
	seen := make(map[string]struct{}, len(node.Funcs))
	for _, decl := range node.Funcs {
		if decl == nil || decl.Name == nil {
			continue
		}
		fname := decl.Name.Name
		if fname == "" {
			continue
		}
		if _, dup := seen[fname]; dup {
			return c.errorf(node, "native: duplicate function binding %q", fname)
		}
		seen[fname] = struct{}{}

		params := make([]NativeKind, 0, len(decl.ParamTypes))
		paramStructIdx := make([]int, 0, len(decl.ParamTypes))
		paramPointer := make([]bool, 0, len(decl.ParamTypes))
		for _, pt := range decl.ParamTypes {
			if pt == nil {
				continue
			}
			k, si, ptr, err := resolveTypeRef(pt, false, fname, "parameter")
			if err != nil {
				return err
			}
			params = append(params, k)
			paramStructIdx = append(paramStructIdx, si)
			paramPointer = append(paramPointer, ptr)
		}

		ret := NativeVoid
		retStruct := -1
		retPtr := false
		if decl.ReturnType != nil {
			k, si, ptr, err := resolveTypeRef(decl.ReturnType, true, fname, "return")
			if err != nil {
				return err
			}
			ret, retStruct, retPtr = k, si, ptr
		}

		specs = append(specs, NativeFuncSpec{
			Name:            fname,
			Params:          params,
			ParamStructIdx:  paramStructIdx,
			ParamPointer:    paramPointer,
			Return:          ret,
			ReturnStructIdx: retStruct,
			ReturnPointer:   retPtr,
		})
	}

	// Emit: push loader constant, call with zero args, bind result.
	loader := &Native{Path: node.Path, Funcs: specs, Structs: structs}
	c.emit(node, parser.OpConstant, c.addConstant(loader))
	c.emit(node, parser.OpCall, 0, 0)

	// Define/assign the LHS identifier (same logic as compileAssign's Define path).
	ident := node.Name.Name
	symbol, depth, exists := c.symbolTable.Resolve(ident, false)
	if depth == 0 && exists {
		return c.errorf(node, "'%s' redeclared in this block", ident)
	}
	symbol = c.symbolTable.Define(ident)
	switch symbol.Scope {
	case ScopeGlobal:
		c.emit(node, parser.OpSetGlobal, symbol.Index)
	case ScopeLocal:
		c.emit(node, parser.OpDefineLocal, symbol.Index)
		symbol.LocalAssigned = true
	default:
		return c.errorf(node, "native: unexpected symbol scope: %s", symbol.Scope)
	}
	return nil
}

