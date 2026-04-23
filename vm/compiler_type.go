package vm

import (
	"fmt"

	"github.com/malivvan/rumo/vm/parser"
)

// compileType lowers a `type Name <underlying>` statement into a single
// constant push + variable bind. The constant is a *UserType that carries
// enough metadata to construct (and, for func types, wrap) values at runtime.
func (c *Compiler) compileType(node *parser.TypeStmt) error {
	if node.Name == nil || node.Name.Name == "" {
		return c.errorf(node, "type: missing name")
	}
	name := node.Name.Name

	ut := &UserType{Name: name}

	switch t := node.Type.(type) {
	case *parser.StructType:
		ut.Kind = UserTypeStruct
		seen := make(map[string]struct{})
		for _, group := range t.Fields {
			if group == nil || group.Type == nil {
				continue
			}
			typeName, err := typeExprName(group.Type)
			if err != nil {
				return c.errorf(node, "type %s: %v", name, err)
			}
			for _, id := range group.Names {
				if id == nil || id.Name == "" {
					continue
				}
				if _, dup := seen[id.Name]; dup {
					return c.errorf(node, "type %s: duplicate field %q", name, id.Name)
				}
				seen[id.Name] = struct{}{}
				ut.Fields = append(ut.Fields, id.Name)
				ut.FieldTypes = append(ut.FieldTypes, typeName)
			}
		}

	case *parser.TypedFuncType:
		ut.Kind = UserTypeFunc
		ut.VarArgs = t.VarArgs
		for _, p := range t.Params {
			if p == nil || p.Type == nil {
				continue
			}
			typeName, err := typeExprName(p.Type)
			if err != nil {
				return c.errorf(node, "type %s: %v", name, err)
			}
			pname := ""
			if p.Name != nil {
				pname = p.Name.Name
			}
			ut.Params = append(ut.Params, pname)
			ut.ParamTypes = append(ut.ParamTypes, typeName)
		}
		ut.NumParams = len(ut.Params)
		if ut.VarArgs && ut.NumParams > 0 {
			ut.NumParams--
		}
		if t.Result != nil {
			res, err := typeExprName(t.Result)
			if err != nil {
				return c.errorf(node, "type %s: %v", name, err)
			}
			ut.Result = res
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

// typeExprName returns the canonical string form of a type expression used
// inside a type statement (for struct fields or func parameters/results).
// Currently only identifiers are supported.
func typeExprName(e parser.Expr) (string, error) {
	if id, ok := e.(*parser.Ident); ok {
		return id.Name, nil
	}
	return "", fmt.Errorf("unsupported type expression %q (only simple identifiers are allowed)", e.String())
}
