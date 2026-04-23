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

	// Translate each declared function into its internal spec form, checking
	// that every referenced type keyword is valid.
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
		for _, pt := range decl.ParamTypes {
			if pt == nil {
				continue
			}
			k, ok := nativeKindByName(pt.Name)
			if !ok {
				return c.errorf(node,
					"native: unknown parameter type %q in function %q (allowed: int, uint, bool, float, string, ptr, bytes)",
					pt.Name, fname)
			}
			if k == NativeVoid {
				return c.errorf(node,
					"native: 'void' is not allowed as a parameter type in function %q", fname)
			}
			params = append(params, k)
		}

		ret := NativeVoid
		if decl.ReturnType != nil {
			k, ok := nativeKindByName(decl.ReturnType.Name)
			if !ok {
				return c.errorf(node,
					"native: unknown return type %q in function %q (allowed: int, uint, bool, float, string, ptr, bytes, void)",
					decl.ReturnType.Name, fname)
			}
			ret = k
		}

		specs = append(specs, NativeFuncSpec{Name: fname, Params: params, Return: ret})
	}

	// Emit: push loader constant, call with zero args, bind result.
	loader := &Native{Path: node.Path, Funcs: specs}
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

