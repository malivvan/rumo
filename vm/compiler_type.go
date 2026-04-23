package vm

import (
	"github.com/malivvan/rumo/vm/parser"
)

// compileType lowers a `type Name <underlying>` statement into a single
// constant push + variable bind. The constant is a *UserType that carries
// enough metadata to construct values at runtime when the type is called.
func (c *Compiler) compileType(node *parser.TypeStmt) error {
	if node.Name == nil || node.Name.Name == "" {
		return c.errorf(node, "type: missing name")
	}
	name := node.Name.Name

	ut := &UserType{Name: name}

	switch t := node.Type.(type) {
	case *parser.StructType:
		ut.Kind = UserTypeStruct
		seen := make(map[string]struct{}, len(t.Fields))
		for _, f := range t.Fields {
			if f == nil || f.Name == "" {
				continue
			}
			if _, dup := seen[f.Name]; dup {
				return c.errorf(node, "type %s: duplicate field %q", name, f.Name)
			}
			seen[f.Name] = struct{}{}
			ut.Fields = append(ut.Fields, f.Name)
		}

	case *parser.FuncType:
		ut.Kind = UserTypeFunc
		if t.Params != nil {
			ut.VarArgs = t.Params.VarArgs
			params := make([]string, 0, len(t.Params.List))
			for _, p := range t.Params.List {
				if p == nil {
					continue
				}
				params = append(params, p.Name)
			}
			ut.Params = params
			ut.NumParams = len(params)
			if ut.VarArgs && ut.NumParams > 0 {
				ut.NumParams--
			}
		}

	case *parser.Ident:
		ut.Kind = UserTypeValue
		ut.Underlying = t.Name
		if _, ok := valueTypeConverter(t.Name); !ok {
			return c.errorf(node, "type %s: unknown underlying type %q", name, t.Name)
		}

	default:
		return c.errorf(node, "type %s: unsupported underlying expression", name)
	}

	c.emit(node, parser.OpConstant, c.addConstant(ut))

	symbol, depth, exists := c.symbolTable.Resolve(name, false)
	if depth == 0 && exists {
		return c.errorf(node, "'%s' redeclared in this block", name)
	}
	symbol = c.symbolTable.Define(name)
	switch symbol.Scope {
	case ScopeGlobal:
		c.emit(node, parser.OpSetGlobal, symbol.Index)
	case ScopeLocal:
		c.emit(node, parser.OpDefineLocal, symbol.Index)
		symbol.LocalAssigned = true
	default:
		return c.errorf(node, "type: unexpected symbol scope: %s", symbol.Scope)
	}
	return nil
}
