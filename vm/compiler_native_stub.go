//go:build !native

package vm

import (
	"github.com/malivvan/rumo/vm/parser"
)

// compileNative is a stub that always returns an error when the native build
// tag is not set.  The parser still recognises the native keyword and
// produces a *parser.NativeStmt, but the compiler refuses to lower it.
func (c *Compiler) compileNative(node *parser.NativeStmt) error {
	return c.errorf(node, "native: not supported (rebuild with -tags native)")
}


