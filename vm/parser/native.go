//go:build native

package parser

import (
	"strconv"
	"strings"

	"github.com/malivvan/rumo/vm/token"
)

// NativeTypeRef references a native type in a function signature or struct
// field. Pointer is true when the source spelling was `*Name` (only meaningful
// for struct names, since scalar pointers use the `ptr` keyword).
type NativeTypeRef struct {
	StarPos Pos // position of leading '*' or NoPos
	Pointer bool
	Name    *Ident
}

// Pos returns the position of first character belonging to the node.
func (r *NativeTypeRef) Pos() Pos {
	if r.Pointer {
		return r.StarPos
	}
	return r.Name.Pos()
}

// End returns the position of first character immediately after the node.
func (r *NativeTypeRef) End() Pos { return r.Name.End() }

// String renders the type reference back to source form.
func (r *NativeTypeRef) String() string {
	if r.Pointer {
		return "*" + r.Name.Name
	}
	return r.Name.Name
}

// NativeFieldDecl is a single field declaration inside a NativeStructDecl.
type NativeFieldDecl struct {
	Name *Ident
	Type *NativeTypeRef
}

// Pos returns the position of first character belonging to the node.
func (f *NativeFieldDecl) Pos() Pos { return f.Name.Pos() }

// End returns the position of first character immediately after the node.
func (f *NativeFieldDecl) End() Pos { return f.Type.End() }

// String returns the field declaration as source.
func (f *NativeFieldDecl) String() string {
	return f.Name.Name + " " + f.Type.String()
}

// NativeStructDecl declares a C-compatible struct layout that may be
// referenced as a parameter or return type by sibling NativeFuncDecls.
type NativeStructDecl struct {
	StructPos Pos
	Name      *Ident
	LBrace    Pos
	Fields    []*NativeFieldDecl
	RBrace    Pos
}

// Pos returns the position of first character belonging to the node.
func (d *NativeStructDecl) Pos() Pos { return d.StructPos }

// End returns the position of first character immediately after the node.
func (d *NativeStructDecl) End() Pos { return d.RBrace + 1 }

// String returns the struct declaration as source.
func (d *NativeStructDecl) String() string {
	var fields []string
	for _, f := range d.Fields {
		fields = append(fields, f.String())
	}
	return "struct " + d.Name.Name + " { " + strings.Join(fields, "; ") + " }"
}

// NativeFuncDecl represents a single native function declaration inside a
// NativeStmt body.  It captures the binding name, the list of parameter
// types and the return type (or nil for void).
type NativeFuncDecl struct {
	Name       *Ident
	LParen     Pos
	ParamTypes []*NativeTypeRef
	RParen     Pos
	ReturnType *NativeTypeRef // may be nil (void / no return)
}

// Pos returns the position of first character belonging to the node.
func (d *NativeFuncDecl) Pos() Pos { return d.Name.Pos() }

// End returns the position of first character immediately after the node.
func (d *NativeFuncDecl) End() Pos {
	if d.ReturnType != nil {
		return d.ReturnType.End()
	}
	return d.RParen + 1
}

func (d *NativeFuncDecl) String() string {
	var params []string
	for _, p := range d.ParamTypes {
		params = append(params, p.String())
	}
	ret := ""
	if d.ReturnType != nil {
		ret = " " + d.ReturnType.String()
	}
	return d.Name.Name + ": func(" + strings.Join(params, ", ") + ")" + ret
}

// NativeStmt represents a native library binding statement.
//
// Example:
//
//	native libm = "libm.so.6" {
//	    sin: func(float) float
//	    cos: func(float) float
//	    pow: func(float, float) float
//	}
type NativeStmt struct {
	NativePos Pos
	Name      *Ident
	AssignPos Pos
	Path      string
	PathLit   *StringLit
	LBrace    Pos
	Structs   []*NativeStructDecl
	Funcs     []*NativeFuncDecl
	RBrace    Pos
}

func (s *NativeStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *NativeStmt) Pos() Pos { return s.NativePos }

// End returns the position of first character immediately after the node.
func (s *NativeStmt) End() Pos { return s.RBrace + 1 }

func (s *NativeStmt) String() string {
	var parts []string
	for _, sd := range s.Structs {
		parts = append(parts, sd.String())
	}
	for _, f := range s.Funcs {
		parts = append(parts, f.String())
	}
	return "native " + s.Name.Name + " = " + s.PathLit.Literal + " {" +
		strings.Join(parts, "; ") + "}"
}

func (p *Parser) parseNativeStmt() Stmt {
	if p.trace {
		defer untracep(tracep(p, "NativeStmt"))
	}

	pos := p.expect(token.Native)

	// name
	if p.token != token.Ident {
		p.errorExpected(p.pos, "identifier")
		p.advance(stmtStart)
		return &BadStmt{From: pos, To: p.pos}
	}
	name := p.parseIdent()

	// '=' sign
	assignPos := p.pos
	if p.token != token.Assign {
		p.errorExpected(p.pos, "'='")
		p.advance(stmtStart)
		return &BadStmt{From: pos, To: p.pos}
	}
	p.next()

	// library path (string literal)
	if p.token != token.String {
		p.errorExpected(p.pos, "native library path (string literal)")
		p.advance(stmtStart)
		return &BadStmt{From: pos, To: p.pos}
	}
	pathVal, _ := strconv.Unquote(p.tokenLit)
	pathLit := &StringLit{
		Value:    pathVal,
		ValuePos: p.pos,
		Literal:  p.tokenLit,
	}
	p.next()

	// '{'
	if p.token != token.LBrace {
		p.errorExpected(p.pos, "'{'")
		p.advance(stmtStart)
		return &BadStmt{From: pos, To: p.pos}
	}
	lbrace := p.expect(token.LBrace)

	var structs []*NativeStructDecl
	var funcs []*NativeFuncDecl
	for p.token != token.RBrace && p.token != token.EOF {
		// allow blank lines / explicit semicolons between decls
		if p.token == token.Semicolon {
			p.next()
			continue
		}

		if p.token == token.Struct {
			sd := p.parseNativeStructDecl()
			if sd == nil {
				for p.token != token.Semicolon && p.token != token.RBrace && p.token != token.EOF {
					p.next()
				}
				continue
			}
			structs = append(structs, sd)
			if p.token == token.Semicolon {
				p.next()
			} else if p.token != token.RBrace {
				p.errorExpected(p.pos, "';' or newline")
				break
			}
			continue
		}

		decl := p.parseNativeFuncDecl()
		if decl == nil {
			// recovery: skip to next line / closing brace
			for p.token != token.Semicolon && p.token != token.RBrace && p.token != token.EOF {
				p.next()
			}
			continue
		}
		funcs = append(funcs, decl)

		if p.token == token.Semicolon {
			p.next()
		} else if p.token != token.RBrace {
			p.errorExpected(p.pos, "';' or newline")
			break
		}
	}

	rbrace := p.expect(token.RBrace)
	p.expectSemi()

	return &NativeStmt{
		NativePos: pos,
		Name:      name,
		AssignPos: assignPos,
		Path:      pathVal,
		PathLit:   pathLit,
		LBrace:    lbrace,
		Structs:   structs,
		Funcs:     funcs,
		RBrace:    rbrace,
	}
}

// parseNativeTypeRef parses an optional `*` followed by an identifier.
func (p *Parser) parseNativeTypeRef() *NativeTypeRef {
	ref := &NativeTypeRef{}
	if p.token == token.Mul {
		ref.StarPos = p.pos
		ref.Pointer = true
		p.next()
	}
	if p.token != token.Ident {
		p.errorExpected(p.pos, "native type name")
		return nil
	}
	ref.Name = p.parseIdent()
	return ref
}

// parseNativeStructDecl parses `struct Name { field type; ... }`.
func (p *Parser) parseNativeStructDecl() *NativeStructDecl {
	structPos := p.expect(token.Struct)
	if p.token != token.Ident {
		p.errorExpected(p.pos, "struct name")
		return nil
	}
	name := p.parseIdent()
	if p.token != token.LBrace {
		p.errorExpected(p.pos, "'{'")
		return nil
	}
	lbrace := p.expect(token.LBrace)

	var fields []*NativeFieldDecl
	for p.token != token.RBrace && p.token != token.EOF {
		if p.token == token.Semicolon {
			p.next()
			continue
		}
		if p.token != token.Ident {
			p.errorExpected(p.pos, "field name")
			return nil
		}
		fname := p.parseIdent()
		typ := p.parseNativeTypeRef()
		if typ == nil {
			return nil
		}
		fields = append(fields, &NativeFieldDecl{Name: fname, Type: typ})
		if p.token == token.Semicolon {
			p.next()
		} else if p.token != token.RBrace {
			p.errorExpected(p.pos, "';' or newline")
			return nil
		}
	}
	rbrace := p.expect(token.RBrace)

	return &NativeStructDecl{
		StructPos: structPos,
		Name:      name,
		LBrace:    lbrace,
		Fields:    fields,
		RBrace:    rbrace,
	}
}

// parseNativeFuncDecl parses a single native function declaration inside a
// native {...} block.  Grammar (one of):
//
//	name : func ( type, type ) retType
//	name : func ( type, type )                 (void return)
//	name ( type, type ) retType                (short form)
//	name ( type, type )
func (p *Parser) parseNativeFuncDecl() *NativeFuncDecl {
	if p.token != token.Ident {
		p.errorExpected(p.pos, "native function name")
		return nil
	}
	name := p.parseIdent()

	// optional ": func" prefix
	if p.token == token.Colon {
		p.next()
		if p.token != token.Func {
			p.errorExpected(p.pos, "'func'")
			return nil
		}
		p.next() // consume 'func'
	}

	if p.token != token.LParen {
		p.errorExpected(p.pos, "'('")
		return nil
	}
	lparen := p.expect(token.LParen)

	var params []*NativeTypeRef
	if p.token != token.RParen {
		for {
			ref := p.parseNativeTypeRef()
			if ref == nil {
				return nil
			}
			params = append(params, ref)
			if p.token != token.Comma {
				break
			}
			p.next()
		}
	}
	rparen := p.expect(token.RParen)

	// optional return type
	var ret *NativeTypeRef
	if p.token == token.Ident || p.token == token.Mul {
		ret = p.parseNativeTypeRef()
	}

	return &NativeFuncDecl{
		Name:       name,
		LParen:     lparen,
		ParamTypes: params,
		RParen:     rparen,
		ReturnType: ret,
	}
}

