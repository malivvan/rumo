package parser

import (
	"strings"
)

// StructType represents a struct type literal.
//
// Example:
//
//	struct {
//	    x
//	    y
//	    name
//	}
type StructType struct {
	StructPos Pos
	LBrace    Pos
	Fields    []*Ident
	RBrace    Pos
}

func (e *StructType) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StructType) Pos() Pos { return e.StructPos }

// End returns the position of first character immediately after the node.
func (e *StructType) End() Pos { return e.RBrace + 1 }

func (e *StructType) String() string {
	var names []string
	for _, f := range e.Fields {
		names = append(names, f.Name)
	}
	return "struct { " + strings.Join(names, "; ") + " }"
}

// TypeStmt represents a Go-style type definition statement.
//
// Supported forms:
//
//	type Point struct { x; y }     // struct type
//	type Handler func(a, b)        // func type
//	type MyInt int                 // value (alias / named) type
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
