//go:build native

package parser

import (
	"strconv"
	"strings"

	"github.com/malivvan/rumo/vm/token"
)

// NativeFuncDecl represents a single native function declaration inside a
// NativeStmt body.  It captures the binding name, the list of parameter
// types (as identifiers, e.g. "int", "float") and the return type (or nil
// for void).
type NativeFuncDecl struct {
	Name       *Ident
	LParen     Pos
	ParamTypes []*Ident
	RParen     Pos
	ReturnType *Ident // may be nil (void / no return)
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
		params = append(params, p.Name)
	}
	ret := ""
	if d.ReturnType != nil {
		ret = " " + d.ReturnType.Name
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

	var funcs []*NativeFuncDecl
	for p.token != token.RBrace && p.token != token.EOF {
		// allow blank lines / explicit semicolons between decls
		if p.token == token.Semicolon {
			p.next()
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
		Funcs:     funcs,
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

	var params []*Ident
	if p.token != token.RParen {
		for {
			if p.token != token.Ident {
				p.errorExpected(p.pos, "native type name")
				return nil
			}
			params = append(params, p.parseIdent())
			if p.token != token.Comma {
				break
			}
			p.next()
		}
	}
	rparen := p.expect(token.RParen)

	// optional return type
	var ret *Ident
	if p.token == token.Ident {
		ret = p.parseIdent()
	}

	return &NativeFuncDecl{
		Name:       name,
		LParen:     lparen,
		ParamTypes: params,
		RParen:     rparen,
		ReturnType: ret,
	}
}

