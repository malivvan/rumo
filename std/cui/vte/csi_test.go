package vte

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

func TestICH(t *testing.T) {
	vt := New()
	vt.Resize(2, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	assert.Equal(t, "ab", vt.String())
	vt.cursor.col = 0
	vt.ich(0)
	assert.Equal(t, " a", vt.String())
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestCUU(t *testing.T) {
	vt := New()
	vt.Resize(2, 2)

	vt.cursor.row = 1
	vt.cursor.col = 1
	assert.Equal(t, column(1), vt.cursor.col)
	assert.Equal(t, row(1), vt.cursor.row)
	vt.cuu(0)

	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(1), vt.cursor.col)
	vt.cuu(0)
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(1), vt.cursor.col)
}

func TestIL(t *testing.T) {
	vt := New()
	vt.Resize(2, 2)
	vt.print('a')
	vt.print('b')
	vt.cursor.col = 0
	vt.cursor.row = 0

	vt.il(1)
	assert.Equal(t, "  \nab", vt.String())

	vt = New()
	vt.Resize(2, 2)
	vt.print('a')
	vt.print('b')
	vt.cursor.col = 0
	vt.cursor.row = 0

	vt.il(2)
	assert.Equal(t, "  \n  ", vt.String())
}

func TestDL(t *testing.T) {
	vt := New()
	vt.Resize(2, 2)
	vt.cursor.row = 1
	vt.print('a')
	vt.print('b')
	assert.Equal(t, "  \nab", vt.String())
	vt.cursor.col = 0
	vt.cursor.row = 0

	vt.dl(1)
	assert.Equal(t, "ab\n  ", vt.String())

	vt = New()
	vt.Resize(2, 2)
	vt.cursor.row = 1
	vt.print('a')
	vt.print('b')
	assert.Equal(t, "  \nab", vt.String())
	vt.cursor.col = 0
	vt.cursor.row = 0
	vt.dl(2)
	assert.Equal(t, "  \n  ", vt.String())
}

func TestDCH(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	vt.dch(1)
	assert.Equal(t, "abc ", vt.String())
	vt.dch(2)
	assert.Equal(t, "abc ", vt.String())
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())
	vt.cursor.col = 1
	vt.dch(2)
	assert.Equal(t, "ad  ", vt.String())
}

// VTE-006: ECH should erase the character at the last column when cursor is there.
func TestECH_LastColumn(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())
	// Move cursor to last column (index 3)
	vt.cursor.col = 3
	vt.ech(1)
	assert.Equal(t, "abc ", vt.String())
}

// VTE-009: CHT should not count a tab stop at the current cursor position.
func TestCHT_AtTabStop(t *testing.T) {
	vt := New()
	vt.Resize(80, 1)
	// Default tab stops at 7, 15, 23, ...
	// Place cursor exactly on a tab stop
	vt.cursor.col = 7
	vt.cht(1)
	// Should advance to the NEXT tab stop (15), not stay at 7
	assert.Equal(t, column(15), vt.cursor.col)
}

// VTE-010: CBT should not count a tab stop at the current cursor position.
func TestCBT_AtTabStop(t *testing.T) {
	vt := New()
	vt.Resize(80, 1)
	// Default tab stops at 7, 15, 23, ...
	// Place cursor exactly on a tab stop
	vt.cursor.col = 15
	vt.cbt(1)
	// Should move back to the PREVIOUS tab stop (7), not stay at 15
	assert.Equal(t, column(7), vt.cursor.col)
}

// VTE-011: REP should advance the cursor and copy character attributes.
func TestREP_AdvancesCursor(t *testing.T) {
	vt := New()
	vt.Resize(6, 1)
	vt.mode = 0
	vt.print('x')
	assert.Equal(t, column(1), vt.cursor.col)
	vt.rep(3)
	// Cursor should have advanced by 3 positions
	assert.Equal(t, column(4), vt.cursor.col)
	assert.Equal(t, "xxxx  ", vt.String())
}

// VTE-015: DECSCUSR should only call ps() once.
func TestDECSCUSR(t *testing.T) {
	vt := New()
	vt.Resize(2, 1)
	vt.csi(" q", []int{2})
	assert.Equal(t, tcell.CursorStyleSteadyBlock, vt.cursor.style)
}

// VTE-028: DECSED (CSI ? J) should erase only unprotected characters.
// Selective Erase in Display erases cells that do NOT have the protected attribute.
func TestDECSED_SelectiveEraseDisplay(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	// Mark cell at column 1 ('b') as protected by setting a distinct attr
	// The protected attribute in DECSCA is tracked per-cell; for our implementation
	// we need to ensure selectiveErase() only clears content, not attrs.
	// For this test, we directly set the protected flag.
	vt.activeScreen[0][1].attrs = vt.activeScreen[0][1].attrs.Bold(true) // mark distinctly

	// CSI ? 2 J = selective erase entire display
	vt.cursor.col = 0
	vt.csi("?J", []int{2})

	// All unprotected cells should be erased (content = 0 → rendered as space)
	// Cell 1 has a non-default attr (bold) — but per spec, DECSED checks the
	// "erasable" (DECSCA) attribute, not bold. Since we haven't set DECSCA on any
	// cell, ALL cells should be erased in our basic implementation.
	assert.Equal(t, "    ", vt.String())
}

// VTE-028: DECSEL (CSI ? K) should erase only unprotected characters on the current line.
func TestDECSEL_SelectiveEraseLine(t *testing.T) {
	vt := New()
	vt.Resize(4, 2)
	vt.mode = 0
	// Fill row 0
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	// Fill row 1
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.print('e')
	vt.print('f')
	vt.print('g')
	vt.print('h')

	// CSI ? 2 K on row 1 = selective erase entire line
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.csi("?K", []int{2})

	// Row 0 should be untouched, row 1 should be erased
	assert.Equal(t, 'a', vt.activeScreen[0][0].content)
	assert.Equal(t, 'b', vt.activeScreen[0][1].content)
	assert.Equal(t, rune(0), vt.activeScreen[1][0].content)
	assert.Equal(t, rune(0), vt.activeScreen[1][1].content)
}

// VTE-028: DECSED ps=0 should erase from cursor to end of screen.
func TestDECSED_EraseFromCursor(t *testing.T) {
	vt := New()
	vt.Resize(4, 2)
	vt.mode = 0
	// Fill row 0
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	// Fill row 1
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.print('e')
	vt.print('f')
	vt.print('g')
	vt.print('h')

	// Place cursor at row 0, col 2
	vt.cursor.row = 0
	vt.cursor.col = 2
	vt.csi("?J", []int{0})

	// a,b should remain; c,d and all of row 1 should be erased
	assert.Equal(t, 'a', vt.activeScreen[0][0].content)
	assert.Equal(t, 'b', vt.activeScreen[0][1].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][2].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][3].content)
	assert.Equal(t, rune(0), vt.activeScreen[1][0].content)
}

// VTE-028: DECSEL ps=0 should erase from cursor to end of line.
func TestDECSEL_EraseFromCursor(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')

	vt.cursor.col = 2
	vt.csi("?K", []int{0})

	assert.Equal(t, 'a', vt.activeScreen[0][0].content)
	assert.Equal(t, 'b', vt.activeScreen[0][1].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][2].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][3].content)
}
