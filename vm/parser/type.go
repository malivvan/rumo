package parser

import (
	"strings"
)

// StructField is a single struct field declaration. All listed Names share
// the same Type — `x, y int` parses into one StructField with two names.
type StructField struct {
	Names []*Ident
	Type  Expr // never nil
}

// Pos returns the position of first character belonging to the node.
func (f *StructField) Pos() Pos {
	if len(f.Names) > 0 {
		return f.Names[0].Pos()
	}
	return f.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (f *StructField) End() Pos { return f.Type.End() }

func (f *StructField) String() string {
	var names []string
	for _, n := range f.Names {
		names = append(names, n.Name)
	}
	return strings.Join(names, ", ") + " " + f.Type.String()
}

// StructType represents a typed struct type literal.
//
//	struct {
//	    x, y int
//	    name string
//	}
type StructType struct {
	StructPos Pos
	LBrace    Pos
	Fields    []*StructField
	RBrace    Pos
}

func (e *StructType) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StructType) Pos() Pos { return e.StructPos }

// End returns the position of first character immediately after the node.
func (e *StructType) End() Pos { return e.RBrace + 1 }

func (e *StructType) String() string {
	var parts []string
	for _, f := range e.Fields {
		parts = append(parts, f.String())
	}
	return "struct { " + strings.Join(parts, "; ") + " }"
}

// FuncParam is one entry in a typed function signature. Name may be nil when
// parameters are declared without names: `func(int, string) int`.
type FuncParam struct {
	Name *Ident
	Type Expr // never nil
}

// Pos returns the position of first character belonging to the node.
func (p *FuncParam) Pos() Pos {
	if p.Name != nil {
		return p.Name.Pos()
	}
	return p.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (p *FuncParam) End() Pos { return p.Type.End() }

func (p *FuncParam) String() string {
	if p.Name != nil {
		return p.Name.Name + " " + p.Type.String()
	}
	return p.Type.String()
}

// TypedFuncType represents a Go-style typed function signature, used only as
// the right-hand side of a `type` statement:
//
//	func(a int, b int) int
//	func(string, string)
//	func(xs ...int) bool
type TypedFuncType struct {
	FuncPos Pos
	LParen  Pos
	Params  []*FuncParam
	VarArgs bool
	RParen  Pos
	Result  Expr // may be nil (void)
}

func (e *TypedFuncType) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *TypedFuncType) Pos() Pos { return e.FuncPos }

// End returns the position of first character immediately after the node.
func (e *TypedFuncType) End() Pos {
	if e.Result != nil {
		return e.Result.End()
	}
	return e.RParen + 1
}

func (e *TypedFuncType) String() string {
	var parts []string
	for i, p := range e.Params {
		s := p.String()
		if e.VarArgs && i == len(e.Params)-1 {
			if p.Name != nil {
				s = p.Name.Name + " ..." + p.Type.String()
			} else {
				s = "..." + p.Type.String()
			}
		}
		parts = append(parts, s)
	}
	sig := "func(" + strings.Join(parts, ", ") + ")"
	if e.Result != nil {
		sig += " " + e.Result.String()
	}
	return sig
}

// TypeStmt represents a Go-style type definition statement.
//
// Supported forms:
//
//	type Point struct { x, y int }         // struct type
//	type Handler func(r int) int           // func type
//	type MyInt int                         // value (alias / named) type
type TypeStmt struct {
	TypePos Pos
	Name    *Ident
	Type    Expr
}

func (s *TypeStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *TypeStmt) Pos() Pos { return s.TypePos }

// End returns the position of first character immediately after the node.
func (s *TypeStmt) End() Pos {
	if s.Type != nil {
		return s.Type.End()
	}
	return s.Name.End()
}

func (s *TypeStmt) String() string {
	if s.Type == nil {
		return "type " + s.Name.String()
	}
	return "type " + s.Name.String() + " " + s.Type.String()
}
