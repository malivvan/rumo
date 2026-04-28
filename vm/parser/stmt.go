package parser

import (
	"strings"

	"github.com/malivvan/rumo/vm/token"
)

// Stmt represents a statement in the AST.
type Stmt interface {
	Node
	stmtNode()
}

// AssignStmt represents an assignment statement.
type AssignStmt struct {
	LHS      []Expr
	RHS      []Expr
	Token    token.Token
	TokenPos Pos
}

func (s *AssignStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *AssignStmt) Pos() Pos {
	return s.LHS[0].Pos()
}

// End returns the position of first character immediately after the node.
func (s *AssignStmt) End() Pos {
	return s.RHS[len(s.RHS)-1].End()
}

func (s *AssignStmt) String() string {
	var lhs, rhs []string
	for _, e := range s.LHS {
		lhs = append(lhs, e.String())
	}
	for _, e := range s.RHS {
		rhs = append(rhs, e.String())
	}
	return strings.Join(lhs, ", ") + " " + s.Token.String() +
		" " + strings.Join(rhs, ", ")
}

// EmbedStmt represents a file embed statement (produced when an //embed
// directive comment precedes a := assignment).
type EmbedStmt struct {
	Patterns []string
	Assign   *AssignStmt
}

func (s *EmbedStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *EmbedStmt) Pos() Pos { return s.Assign.Pos() }

// End returns the position of first character immediately after the node.
func (s *EmbedStmt) End() Pos { return s.Assign.End() }

func (s *EmbedStmt) String() string { return s.Assign.String() }


// BadStmt represents a bad statement.
type BadStmt struct {
	From Pos
	To   Pos
}

func (s *BadStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BadStmt) Pos() Pos {
	return s.From
}

// End returns the position of first character immediately after the node.
func (s *BadStmt) End() Pos {
	return s.To
}

func (s *BadStmt) String() string {
	return "<bad statement>"
}

// BlockStmt represents a block statement.
type BlockStmt struct {
	Stmts  []Stmt
	LBrace Pos
	RBrace Pos
}

func (s *BlockStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BlockStmt) Pos() Pos {
	return s.LBrace
}

// End returns the position of first character immediately after the node.
func (s *BlockStmt) End() Pos {
	return s.RBrace + 1
}

func (s *BlockStmt) String() string {
	var list []string
	for _, e := range s.Stmts {
		list = append(list, e.String())
	}
	return "{" + strings.Join(list, "; ") + "}"
}

// BranchStmt represents a branch statement.
type BranchStmt struct {
	Token    token.Token
	TokenPos Pos
	Label    *Ident
}

func (s *BranchStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *BranchStmt) Pos() Pos {
	return s.TokenPos
}

// End returns the position of first character immediately after the node.
func (s *BranchStmt) End() Pos {
	if s.Label != nil {
		return s.Label.End()
	}

	return Pos(int(s.TokenPos) + len(s.Token.String()))
}

func (s *BranchStmt) String() string {
	var label string
	if s.Label != nil {
		label = " " + s.Label.Name
	}
	return s.Token.String() + label
}

// LabeledStmt represents a labeled statement (e.g. `L: stmt`).
type LabeledStmt struct {
	Label *Ident
	Colon Pos
	Stmt  Stmt
}

func (s *LabeledStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *LabeledStmt) Pos() Pos { return s.Label.Pos() }

// End returns the position of first character immediately after the node.
func (s *LabeledStmt) End() Pos { return s.Stmt.End() }

func (s *LabeledStmt) String() string {
	return s.Label.Name + ": " + s.Stmt.String()
}

// EmptyStmt represents an empty statement.
type EmptyStmt struct {
	Semicolon Pos
	Implicit  bool
}

func (s *EmptyStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *EmptyStmt) Pos() Pos {
	return s.Semicolon
}

// End returns the position of first character immediately after the node.
func (s *EmptyStmt) End() Pos {
	if s.Implicit {
		return s.Semicolon
	}
	return s.Semicolon + 1
}

func (s *EmptyStmt) String() string {
	return ";"
}

// ExportStmt represents an export statement.
type ExportStmt struct {
	ExportPos Pos
	Result    Expr
}

func (s *ExportStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ExportStmt) Pos() Pos {
	return s.ExportPos
}

// End returns the position of first character immediately after the node.
func (s *ExportStmt) End() Pos {
	return s.Result.End()
}

func (s *ExportStmt) String() string {
	return "export " + s.Result.String()
}

// ExprStmt represents an expression statement.
type ExprStmt struct {
	Expr Expr
}

func (s *ExprStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ExprStmt) Pos() Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *ExprStmt) End() Pos {
	return s.Expr.End()
}

func (s *ExprStmt) String() string {
	return s.Expr.String()
}

// ForInStmt represents a for-in statement.
type ForInStmt struct {
	ForPos   Pos
	Key      *Ident
	Value    *Ident
	Iterable Expr
	Body     *BlockStmt
}

func (s *ForInStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForInStmt) Pos() Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForInStmt) End() Pos {
	return s.Body.End()
}

func (s *ForInStmt) String() string {
	if s.Value != nil {
		return "for " + s.Key.String() + ", " + s.Value.String() +
			" in " + s.Iterable.String() + " " + s.Body.String()
	}
	return "for " + s.Key.String() + " in " + s.Iterable.String() +
		" " + s.Body.String()
}

// ForStmt represents a for statement.
type ForStmt struct {
	ForPos Pos
	Init   Stmt
	Cond   Expr
	Post   Stmt
	Body   *BlockStmt
}

func (s *ForStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForStmt) Pos() Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForStmt) End() Pos {
	return s.Body.End()
}

func (s *ForStmt) String() string {
	var init, cond, post string
	if s.Init != nil {
		init = s.Init.String()
	}
	if s.Cond != nil {
		cond = s.Cond.String() + " "
	}
	if s.Post != nil {
		post = s.Post.String()
	}

	if init != "" || post != "" {
		return "for " + init + " ; " + cond + " ; " + post + s.Body.String()
	}
	return "for " + cond + s.Body.String()
}

// IfStmt represents an if statement.
type IfStmt struct {
	IfPos Pos
	Init  Stmt
	Cond  Expr
	Body  *BlockStmt
	Else  Stmt // else branch; or nil
}

func (s *IfStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IfStmt) Pos() Pos {
	return s.IfPos
}

// End returns the position of first character immediately after the node.
func (s *IfStmt) End() Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	return s.Body.End()
}

func (s *IfStmt) String() string {
	var initStmt, elseStmt string
	if s.Init != nil {
		initStmt = s.Init.String() + "; "
	}
	if s.Else != nil {
		elseStmt = " else " + s.Else.String()
	}
	return "if " + initStmt + s.Cond.String() + " " +
		s.Body.String() + elseStmt
}

// IncDecStmt represents increment or decrement statement.
type IncDecStmt struct {
	Expr     Expr
	Token    token.Token
	TokenPos Pos
}

func (s *IncDecStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IncDecStmt) Pos() Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *IncDecStmt) End() Pos {
	return Pos(int(s.TokenPos) + 2)
}

func (s *IncDecStmt) String() string {
	return s.Expr.String() + s.Token.String()
}

// DeferStmt represents a defer statement.
type DeferStmt struct {
	DeferPos Pos
	Call     *CallExpr
}

func (s *DeferStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *DeferStmt) Pos() Pos {
	return s.DeferPos
}

// End returns the position of first character immediately after the node.
func (s *DeferStmt) End() Pos {
	return s.Call.End()
}

func (s *DeferStmt) String() string {
	return "defer " + s.Call.String()
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	ReturnPos Pos
	Result    Expr
}

func (s *ReturnStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ReturnStmt) Pos() Pos {
	return s.ReturnPos
}

// End returns the position of first character immediately after the node.
func (s *ReturnStmt) End() Pos {
	if s.Result != nil {
		return s.Result.End()
	}
	return s.ReturnPos + 6
}

func (s *ReturnStmt) String() string {
	if s.Result != nil {
		return "return " + s.Result.String()
	}
	return "return"
}

// SwitchStmt represents a switch statement.
//
//	switch [init;] [tag] { case ... ; case ... ; default ... }
//
// When Tag is nil, each case expression is evaluated as a boolean condition
// (Go-style "switch true"). Body.Stmts contains *CaseClause nodes only.
type SwitchStmt struct {
	SwitchPos Pos
	Init      Stmt       // optional
	Tag       Expr       // optional; nil ⇒ boolean cases
	Body      *BlockStmt // contains *CaseClause statements
}

func (s *SwitchStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *SwitchStmt) Pos() Pos { return s.SwitchPos }

// End returns the position of first character immediately after the node.
func (s *SwitchStmt) End() Pos { return s.Body.End() }

func (s *SwitchStmt) String() string {
	var parts []string
	parts = append(parts, "switch")
	if s.Init != nil {
		parts = append(parts, s.Init.String()+";")
	}
	if s.Tag != nil {
		parts = append(parts, s.Tag.String())
	}
	parts = append(parts, s.Body.String())
	return strings.Join(parts, " ")
}

// CaseClause represents a case or default clause in a switch statement.
// List is nil for the default clause.
type CaseClause struct {
	CasePos Pos
	List    []Expr // nil ⇒ default
	Colon   Pos
	Body    []Stmt
}

func (s *CaseClause) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *CaseClause) Pos() Pos { return s.CasePos }

// End returns the position of first character immediately after the node.
func (s *CaseClause) End() Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return s.Colon + 1
}

func (s *CaseClause) String() string {
	var head string
	if s.List == nil {
		head = "default:"
	} else {
		var exprs []string
		for _, e := range s.List {
			exprs = append(exprs, e.String())
		}
		head = "case " + strings.Join(exprs, ", ") + ":"
	}
	var stmts []string
	for _, b := range s.Body {
		stmts = append(stmts, b.String())
	}
	if len(stmts) == 0 {
		return head
	}
	return head + " " + strings.Join(stmts, "; ")
}

// SelectStmt represents a select statement (Go-style).
//
//	select { case ... ; case ... ; default ... }
//
// Body.Stmts contains *CommClause nodes only.
type SelectStmt struct {
	SelectPos Pos
	Body      *BlockStmt // contains *CommClause statements
}

func (s *SelectStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *SelectStmt) Pos() Pos { return s.SelectPos }

// End returns the position of first character immediately after the node.
func (s *SelectStmt) End() Pos { return s.Body.End() }

func (s *SelectStmt) String() string {
	return "select " + s.Body.String()
}

// CommClause represents a `case` or `default` clause inside a select
// statement.  When Comm is nil, the clause is the `default` clause; otherwise
// Comm is one of:
//
//   - *ExprStmt    wrapping `ch.recv()` or `ch.send(v)`
//   - *AssignStmt  whose RHS is `ch.recv()` and whose LHS is one or two
//     identifiers (`v := ch.recv()` or `v, ok := ch.recv()`)
type CommClause struct {
	CasePos Pos
	Comm    Stmt // nil ⇒ default
	Colon   Pos
	Body    []Stmt
}

func (s *CommClause) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *CommClause) Pos() Pos { return s.CasePos }

// End returns the position of first character immediately after the node.
func (s *CommClause) End() Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return s.Colon + 1
}

func (s *CommClause) String() string {
	var head string
	if s.Comm == nil {
		head = "default:"
	} else {
		head = "case " + s.Comm.String() + ":"
	}
	var stmts []string
	for _, b := range s.Body {
		stmts = append(stmts, b.String())
	}
	if len(stmts) == 0 {
		return head
	}
	return head + " " + strings.Join(stmts, "; ")
}

