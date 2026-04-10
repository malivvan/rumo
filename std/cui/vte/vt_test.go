package vte

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResize(t *testing.T) {
	vt := New()
	w := 4
	h := 1
	vt.Resize(w, h)
	assert.Equal(t, h, len(vt.activeScreen))
	assert.Equal(t, w, len(vt.activeScreen[0]))
}

func TestString(t *testing.T) {
	vt := New()
	w := 2
	h := 1
	vt.Resize(w, h)
	assert.Equal(t, "  ", vt.String())

	vt.activeScreen[0][0].content = 'v'
	vt.activeScreen[0][1].content = 't'
	assert.Equal(t, "vt", vt.String())
}

func TestPrint(t *testing.T) {
	t.Run("No modes", func(t *testing.T) {
		vt := New()
		vt.mode = 0
		w := 2
		h := 1
		vt.Resize(w, h)

		vt.print('v')
		vt.print('t')
		assert.Equal(t, "vt", vt.String())
		assert.Equal(t, column(1), vt.cursor.col)
		vt.print('x')
		assert.Equal(t, "vx", vt.String())
	})

	t.Run("IRM = set", func(t *testing.T) {
		vt := New()
		w := 4
		h := 1
		vt.Resize(w, h)

		vt.print('v')
		vt.print('t')
		vt.bs()
		vt.bs()
		assert.Equal(t, column(0), vt.cursor.col)
		assert.Equal(t, "vt  ", vt.String())
		vt.mode |= irm
		vt.print('i')
		assert.Equal(t, "ivt ", vt.String())
		vt.print('j')
		vt.print('k')
		assert.Equal(t, "ijkv", vt.String())
	})

	t.Run("DECAWM = set", func(t *testing.T) {
		vt := New()
		w := 3
		h := 2
		vt.Resize(w, h)
		vt.mode |= decawm

		vt.print('v')
		vt.print('t')
		assert.Equal(t, "vt \n   ", vt.String())
		vt.print('i')
		assert.Equal(t, "vti\n   ", vt.String())
		vt.print('j')
		assert.Equal(t, "vti\nj  ", vt.String())
	})

	t.Run("Wide character", func(t *testing.T) {
		vt := New()
		w := 1
		h := 1
		vt.Resize(w, h)

		vt.print('つ')
		assert.Equal(t, "つ", vt.String())
	})
}

func TestScrollUp(t *testing.T) {
	vt := New()
	vt.mode = 0
	w := 2
	h := 2
	vt.Resize(w, h)

	vt.print('v')
	vt.print('t')
	assert.Equal(t, "vt\n  ", vt.String())
	vt.scrollUp(1)
	assert.Equal(t, "  \n  ", vt.String())

	vt = New()
	w = 1
	h = 8
	vt.Resize(w, h)

	vt.cursor.row = 4
	vt.print('v')
	vt.lastCol = false
	vt.cursor.row = 7
	vt.print('t')
	vt.margin.bottom = 5
	assert.Equal(t, " \n \n \n \nv\n \n \nt", vt.String())
	vt.scrollUp(1)
	assert.Equal(t, " \n \n \nv\n \n \n \nt", vt.String())
}

func TestScrollDown(t *testing.T) {
	vt := New()
	w := 2
	h := 2
	vt.Resize(w, h)

	vt.print('v')
	vt.print('t')
	assert.Equal(t, "vt\n  ", vt.String())
	vt.scrollDown(1)
	assert.Equal(t, "  \nvt", vt.String())
	vt.lastCol = false
	vt.print('b')
	assert.Equal(t, " b\nvt", vt.String())
	vt.scrollDown(1)
	assert.Equal(t, "  \n b", vt.String())
}

func TestCombiningRunes(t *testing.T) {
	vt := New()
	vt.Resize(2, 2)
	vt.print('h')
	vt.print(0x337)
	vt.print(0x317)

	assert.Equal(t, "h̷̗ \n  ", vt.String())
}

// VTE-007: cell.erase() must clear combining runes, width, and wrapped flag.
func TestCellErase_ClearsCombiningAndWrapped(t *testing.T) {
	vt := New()
	vt.Resize(2, 1)
	vt.print('h')
	vt.print(0x0337) // combining short solidus overlay
	// Verify combining is set
	assert.Len(t, vt.activeScreen[0][0].combining, 1)
	vt.activeScreen[0][0].wrapped = true
	vt.activeScreen[0][0].width = 1

	vt.activeScreen[0][0].erase(vt.cursor.attrs)

	assert.Nil(t, vt.activeScreen[0][0].combining)
	assert.False(t, vt.activeScreen[0][0].wrapped)
	assert.Equal(t, 0, vt.activeScreen[0][0].width)
}

// VTE-012: Resize must not panic when pty is nil.
func TestResize_NilPty(t *testing.T) {
	vt := New()
	// pty is nil by default — Resize should not panic
	assert.NotPanics(t, func() {
		vt.Resize(10, 5)
	})
}

// VTE-029: scrollUp should only copy within left/right margins.
func TestScrollUp_RespectsLeftRightMargins(t *testing.T) {
	vt := New()
	vt.Resize(4, 2)
	vt.mode = 0
	// Fill row 0: "abcd"
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	// Fill row 1: "efgh"
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.print('e')
	vt.print('f')
	vt.print('g')
	vt.print('h')

	// Set left/right margins to columns 1-2
	vt.margin.left = 1
	vt.margin.right = 2

	vt.scrollUp(1)
	// Columns outside margins (0,3) of row 0 should be unchanged
	assert.Equal(t, 'a', vt.activeScreen[0][0].content)
	assert.Equal(t, 'd', vt.activeScreen[0][3].content)
	// Columns inside margins should have scrolled up from row 1
	assert.Equal(t, 'f', vt.activeScreen[0][1].content)
	assert.Equal(t, 'g', vt.activeScreen[0][2].content)
}

// VTE-030: Close must not panic when pty is nil.
func TestClose_NilPty(t *testing.T) {
	vt := New()
	assert.NotPanics(t, func() {
		vt.Close()
	})
}

// VTE-032: DECALN (ESC # 8) should fill screen with 'E' characters.
func TestDECALN(t *testing.T) {
	vt := New()
	vt.Resize(3, 2)
	vt.esc("#8")
	assert.Equal(t, "EEE\nEEE", vt.String())
}

// VTE-032 regression: DECALN should also reset margins and cursor.
func TestDECALN_ResetsState(t *testing.T) {
	vt := New()
	vt.Resize(4, 3)
	// Set cursor and margins to non-default
	vt.cursor.row = 2
	vt.cursor.col = 3
	vt.margin.top = 1
	vt.margin.bottom = 1
	vt.margin.left = 1
	vt.margin.right = 2

	vt.esc("#8")

	// Verify screen is filled with 'E'
	assert.Equal(t, "EEEE\nEEEE\nEEEE", vt.String())
	// Verify cursor is reset
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
	// Verify margins are reset
	assert.Equal(t, row(0), vt.margin.top)
	assert.Equal(t, row(2), vt.margin.bottom)
	assert.Equal(t, column(0), vt.margin.left)
	assert.Equal(t, column(3), vt.margin.right)
}

// VTE-033: ESC #3, #4, #5, #6 should be silently consumed.
func TestDECLineAttrs_SilentlyConsumed(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.print('a')
	vt.print('b')
	// These should not panic or alter screen content
	assert.NotPanics(t, func() {
		vt.esc("#3") // DECDHL top
		vt.esc("#4") // DECDHL bottom
		vt.esc("#5") // DECSWL
		vt.esc("#6") // DECDWL
	})
	// Screen should be unchanged
	assert.Equal(t, "ab  ", vt.String())
}

// VTE-016 regression: HTS should deduplicate tab stops.
func TestHTS_Dedup(t *testing.T) {
	vt := New()
	vt.Resize(80, 1)
	vt.tabStop = []column{}
	vt.cursor.col = 10
	vt.hts()
	vt.hts() // duplicate
	assert.Equal(t, []column{10}, vt.tabStop)
}

// VTE-016: HTS should maintain tab stops in sorted order.
func TestHTS_SortedOrder(t *testing.T) {
	vt := New()
	vt.Resize(80, 1)
	// Clear default tab stops
	vt.tabStop = []column{}
	// Set tab stops out of order
	vt.cursor.col = 20
	vt.hts()
	vt.cursor.col = 10
	vt.hts()
	vt.cursor.col = 30
	vt.hts()
	// Tab stops should be sorted
	assert.Equal(t, []column{10, 20, 30}, vt.tabStop)
	// CHT from col 0 should land on 10 (first tab)
	vt.cursor.col = 0
	vt.cht(1)
	assert.Equal(t, column(10), vt.cursor.col)
}

// VTE-029 regression: scrollDown should also respect left/right margins.
func TestScrollDown_RespectsLeftRightMargins(t *testing.T) {
	vt := New()
	vt.Resize(4, 2)
	vt.mode = 0
	// Fill row 0: "abcd"
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	// Fill row 1: "efgh"
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.print('e')
	vt.print('f')
	vt.print('g')
	vt.print('h')

	// Set left/right margins to columns 1-2
	vt.margin.left = 1
	vt.margin.right = 2

	vt.scrollDown(1)
	// Row 1 columns 1-2 should have row 0's content
	assert.Equal(t, 'b', vt.activeScreen[1][1].content)
	assert.Equal(t, 'c', vt.activeScreen[1][2].content)
	// Columns outside margins should be unchanged
	assert.Equal(t, 'e', vt.activeScreen[1][0].content)
	assert.Equal(t, 'h', vt.activeScreen[1][3].content)
}
