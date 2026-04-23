//go:build !native

package parser

// NativeFuncDecl is a stub type present in non-native builds so that
// NativeStmt can reference it (keeping the public API consistent between
// build configurations).
type NativeFuncDecl struct{}

// NativeStmt is a stub type present in non-native builds.  The parser still
// recognises the native keyword and stores the position so that the compiler
// can emit a clear "not supported" error.
type NativeStmt struct {
	NativePos Pos
}

func (s *NativeStmt) stmtNode() {}

// Pos returns the start position of the native statement.
func (s *NativeStmt) Pos() Pos { return s.NativePos }

// End returns the end position of the native statement.
func (s *NativeStmt) End() Pos { return s.NativePos }

// String returns a placeholder string.
func (s *NativeStmt) String() string { return "native ..." }

// parseNativeStmt is a stub that emits an error and performs error recovery.
func (p *Parser) parseNativeStmt() Stmt {
	pos := p.pos
	p.error(pos, "native: not supported (rebuild with -tags native)")
	p.advance(stmtStart)
	return &BadStmt{From: pos, To: p.pos}
}


