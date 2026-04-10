package vte

import (
	"github.com/gdamore/tcell/v3"
)

type cursor struct {
	attrs     tcell.Style
	style     tcell.CursorStyle
	protected bool // DECSCA protection attribute
	overline  bool // SGR 53 overline attribute

	// position
	row row    // 0-indexed
	col column // 0-indexed
}
