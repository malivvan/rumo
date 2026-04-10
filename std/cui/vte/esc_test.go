package vte

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// VTE-F08: ESC N (SS2) and ESC O (SS3) set vt.charsets.selected to g2/g3
// but never save the current value to vt.charsets.saved. When print() reverts
// the single shift, it restores saved which defaults to g0. If the terminal
// was in G1 (after SO/0x0E), the single shift incorrectly reverts to G0
// instead of G1.
func TestSS2_SavesPreviousCharset(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	// Switch to G1 via SO (0x0E)
	vt.charsets.selected = g1
	assert.Equal(t, charsetDesignator(g1), vt.charsets.selected)

	// SS2 (ESC N): should save g1 to saved, then set selected = g2
	vt.esc("N")

	assert.True(t, vt.charsets.singleShift)
	assert.Equal(t, charsetDesignator(g2), vt.charsets.selected)
	// Bug: saved is g0 (default zero value) instead of g1
	assert.Equal(t, charsetDesignator(g1), vt.charsets.saved)
}

func TestSS3_SavesPreviousCharset(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	// Switch to G1 via SO (0x0E)
	vt.charsets.selected = g1

	// SS3 (ESC O): should save g1 to saved, then set selected = g3
	vt.esc("O")

	assert.True(t, vt.charsets.singleShift)
	assert.Equal(t, charsetDesignator(g3), vt.charsets.selected)
	// Bug: saved is g0 (default zero value) instead of g1
	assert.Equal(t, charsetDesignator(g1), vt.charsets.saved)
}

// VTE-F08 regression: SS2 from default G0 state should save g0.
func TestSS2_SavesG0WhenDefault(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	assert.Equal(t, charsetDesignator(g0), vt.charsets.selected)

	vt.esc("N")

	assert.Equal(t, charsetDesignator(g0), vt.charsets.saved)
	assert.Equal(t, charsetDesignator(g2), vt.charsets.selected)
}

// VTE-F23: DECBI (ESC 6) — Back Index. If the cursor is at the left margin,
// content within margins is scrolled right by one column (a blank column is
// inserted at the left margin). Otherwise the cursor moves left by one column.
func TestDECBI_CursorNotAtLeftMargin(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	// Cursor at col 2
	assert.Equal(t, column(2), vt.cursor.col)

	// DECBI should move cursor left by 1
	vt.esc("6")
	assert.Equal(t, column(1), vt.cursor.col)
	// Screen content unchanged
	assert.Equal(t, "ab  ", vt.String())
}

func TestDECBI_CursorAtLeftMargin(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	// Place cursor at left margin
	vt.cursor.col = 0

	// DECBI at left margin should scroll content right within margins,
	// inserting a blank at the left margin. 'd' falls off the right.
	vt.esc("6")
	assert.Equal(t, column(0), vt.cursor.col)
	assert.Equal(t, " abc", vt.String())
}

// VTE-F23 regression: DECBI with left/right margins set.
func TestDECBI_WithMargins(t *testing.T) {
	vt := New()
	vt.Resize(6, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	vt.print('e')
	vt.print('f')
	assert.Equal(t, "abcdef", vt.String())

	// Set left/right margins
	vt.margin.left = 1
	vt.margin.right = 4

	// Cursor at left margin (col 1)
	vt.cursor.col = 1

	vt.esc("6")
	// 'e' (col 4) should fall off, columns 1..3 shift right, blank at col 1
	assert.Equal(t, column(1), vt.cursor.col)
	assert.Equal(t, 'a', vt.activeScreen[0][0].content) // outside margin
	assert.Equal(t, ' ', vt.activeScreen[0][1].rune())   // blank inserted
	assert.Equal(t, 'b', vt.activeScreen[0][2].content)
	assert.Equal(t, 'c', vt.activeScreen[0][3].content)
	assert.Equal(t, 'd', vt.activeScreen[0][4].content)
	assert.Equal(t, 'f', vt.activeScreen[0][5].content) // outside margin
}

// VTE-F23: DECFI (ESC 9) — Forward Index. If the cursor is at the right margin,
// content within margins is scrolled left by one column (a blank column is
// inserted at the right margin). Otherwise the cursor moves right by one column.
func TestDECFI_CursorNotAtRightMargin(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	// Cursor at col 1
	assert.Equal(t, column(1), vt.cursor.col)

	// DECFI should move cursor right by 1
	vt.esc("9")
	assert.Equal(t, column(2), vt.cursor.col)
	// Screen content unchanged
	assert.Equal(t, "a   ", vt.String())
}

func TestDECFI_CursorAtRightMargin(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	// Place cursor at right margin
	vt.cursor.col = 3

	// DECFI at right margin should scroll content left within margins,
	// inserting a blank at the right margin. 'a' falls off the left.
	vt.esc("9")
	assert.Equal(t, column(3), vt.cursor.col)
	assert.Equal(t, "bcd ", vt.String())
}

// VTE-F23 regression: DECFI with left/right margins set.
func TestDECFI_WithMargins(t *testing.T) {
	vt := New()
	vt.Resize(6, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	vt.print('e')
	vt.print('f')
	assert.Equal(t, "abcdef", vt.String())

	// Set left/right margins
	vt.margin.left = 1
	vt.margin.right = 4

	// Cursor at right margin (col 4)
	vt.cursor.col = 4

	vt.esc("9")
	// 'b' (col 1) should fall off left within margins, columns 2..4 shift left, blank at col 4
	assert.Equal(t, column(4), vt.cursor.col)
	assert.Equal(t, 'a', vt.activeScreen[0][0].content) // outside margin
	assert.Equal(t, 'c', vt.activeScreen[0][1].content)
	assert.Equal(t, 'd', vt.activeScreen[0][2].content)
	assert.Equal(t, 'e', vt.activeScreen[0][3].content)
	assert.Equal(t, ' ', vt.activeScreen[0][4].rune())   // blank inserted
	assert.Equal(t, 'f', vt.activeScreen[0][5].content) // outside margin
}
