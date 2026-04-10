package vte

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBS(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.cursor.col = 1
	vt.bs()
	assert.Equal(t, column(0), vt.cursor.col)
	vt.bs()
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestLF(t *testing.T) {
	t.Run("LNM reset", func(t *testing.T) {
		vt := New()
		vt.Resize(2, 2)
		vt.print('v')
		vt.print('t')
		assert.Equal(t, "vt\n  ", vt.String())
		vt.lf()
		assert.Equal(t, "vt\n  ", vt.String())
		assert.Equal(t, column(1), vt.cursor.col)
		assert.Equal(t, row(1), vt.cursor.row)
	})

	t.Run("LNM set", func(t *testing.T) {
		vt := New()
		vt.Resize(2, 2)
		vt.print('v')
		vt.print('t')
		assert.Equal(t, "vt\n  ", vt.String())
		vt.mode |= lnm
		vt.lf()
		assert.Equal(t, "vt\n  ", vt.String())
		assert.Equal(t, column(0), vt.cursor.col)
		assert.Equal(t, row(1), vt.cursor.row)

		vt.print('x')
		vt.lf()
		assert.Equal(t, "x \n  ", vt.String())
		assert.Equal(t, column(0), vt.cursor.col)
		assert.Equal(t, row(1), vt.cursor.row)
	})
}

// VTE-001: SI (0x0F) should invoke G0 into GL, not G2.
func TestSI_SelectsG0(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	// SO selects G1
	vt.c0(0x0E)
	assert.Equal(t, charsetDesignator(g1), vt.charsets.selected)
	// SI should select G0
	vt.c0(0x0F)
	assert.Equal(t, charsetDesignator(g0), vt.charsets.selected)
}

// VTE-008: Backspace reverse-wrap should only happen when DECAWM is set.
func TestBS_NoReverseWrapWithoutDECAWM(t *testing.T) {
	vt := New()
	vt.Resize(4, 2)
	// Disable DECAWM
	vt.mode &^= decawm
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.bs()
	// Without DECAWM, cursor should stay at col 0, row 1 (no reverse wrap)
	assert.Equal(t, column(0), vt.cursor.col)
	assert.Equal(t, row(1), vt.cursor.row)
}

func TestBS_ReverseWrapWithDECAWM(t *testing.T) {
	vt := New()
	vt.Resize(4, 2)
	// Enable DECAWM (already default, but be explicit)
	vt.mode |= decawm
	vt.cursor.row = 1
	vt.cursor.col = 0
	vt.bs()
	// With DECAWM, cursor should reverse-wrap to end of previous line
	assert.Equal(t, column(3), vt.cursor.col)
	assert.Equal(t, row(0), vt.cursor.row)
}

// // Linefeed 0x10
// func (vt *vt) LF() {
// 	switch {
// 	case vt.cursor.row == vt.margin.bottom:
// 		vt.ScrollUp(1)
// 	default:
// 		vt.cursor.row += 1
// 	}
//
// 	if vt.mode&LNM != LNM {
// 		return
// 	}
// 	vt.cursor.col = vt.margin.left
// }
