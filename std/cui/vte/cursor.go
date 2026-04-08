package vte

import (
	"github.com/gdamore/tcell/v3"
)

type cursor struct {
	attrs tcell.Style
	style tcell.CursorStyle

	// position
	row row    // 0-indexed
	col column // 0-indexed
}
