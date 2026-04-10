package vte

import (
	"bytes"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

// mockPty implements io.ReadWriteCloser for capturing pty output in tests.
type mockPty struct {
	bytes.Buffer
}

func (m *mockPty) Close() error { return nil }

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

// VTE-F01: ICH boundary check `col+i > column(vt.width()-1)` adds cursor position
// to the loop variable (which is already an absolute column index), producing a
// meaningless value. This causes the shift loop to break prematurely when the cursor
// is not at column 0, so characters are not shifted right — only a blank is written
// at the cursor position.
func TestICH_BoundaryCheckAtNonZeroCursor(t *testing.T) {
	vt := New()
	vt.Resize(6, 1)
	vt.mode = 0

	// Fill screen: "abcdef"
	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	vt.print('e')
	vt.print('f')
	assert.Equal(t, "abcdef", vt.String())

	// Place cursor at column 2 and insert 1 blank character.
	// Expected: characters at col 2..4 shift right by 1, 'f' falls off,
	// blank inserted at col 2 → "ab cde"
	vt.cursor.col = 2
	vt.ich(1)
	assert.Equal(t, "ab cde", vt.String())
	// Cursor must not move
	assert.Equal(t, column(2), vt.cursor.col)
}

// VTE-F01 regression: ICH with cursor at column 0 (existing behavior should still work)
func TestICH_BoundaryCheckAtColumn0(t *testing.T) {
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

	vt.cursor.col = 0
	vt.ich(1)
	assert.Equal(t, " abcde", vt.String())
	assert.Equal(t, column(0), vt.cursor.col)
}

// VTE-F01 regression: ICH inserting multiple blanks from a non-zero cursor
func TestICH_MultipleInsertAtNonZeroCursor(t *testing.T) {
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

	// Insert 2 blanks at column 1 → shift cdef right by 2, ef fall off
	// Expected: "a  bcd"
	vt.cursor.col = 1
	vt.ich(2)
	assert.Equal(t, "a  bcd", vt.String())
	assert.Equal(t, column(1), vt.cursor.col)
}

// VTE-F02: ICH blank-fill loop uses `>= (vt.width() - 1)` which prevents writing
// a blank to the last screen column. The condition should be `>= vt.width()`.
func TestICH_BlankFillLastColumn(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	// Insert 3 blanks at column 0 → shifts 'a' to col 3, 'b','c','d' fall off.
	// Blanks should fill columns 0, 1, 2. But with the bug, the blank-fill
	// breaks at i=3 because int(0)+3 >= (4-1) = 3, so column 2 never gets blanked.
	vt.cursor.col = 0
	vt.ich(3)
	assert.Equal(t, "   a", vt.String())
}

// VTE-F02 regression: blank-fill at last column position
func TestICH_BlankFillAtLastColumn(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	// Cursor at last column (3), insert 1 blank → 'd' falls off, blank at col 3
	vt.cursor.col = 3
	vt.ich(1)
	assert.Equal(t, "abc ", vt.String())
}

// VTE-F03: ICH source validity check `(i - column(ps)) < 0` only prevents negative
// indices. It should be `(i - column(ps)) < col` to avoid copying characters from
// positions before the cursor, which violates the ICH spec (characters before the
// cursor are not affected).
func TestICH_NoCopyFromBeforeCursor(t *testing.T) {
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

	// Insert 4 blanks at column 3. Per ICH spec, only characters at col 3..5
	// are affected. Characters before cursor (a,b,c) must NOT be copied.
	// Expected: "abc   " (columns 3-5 become blanks, nothing shifts in because
	// all 3 chars at col 3..5 are pushed off the right edge by ps=4)
	vt.cursor.col = 3
	vt.ich(4)
	// With the bug, the shift loop copies from i-4 which can be 1 or 2,
	// pulling 'b' or 'c' into the post-cursor region.
	assert.Equal(t, "abc   ", vt.String())
}

// VTE-F03 regression: ICH where ps exactly covers remaining columns
func TestICH_PsExactlyCoversRemaining(t *testing.T) {
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

	// Insert 3 blanks at column 3 → 3 chars at col 3..5 pushed off, all blanks
	vt.cursor.col = 3
	vt.ich(3)
	assert.Equal(t, "abc   ", vt.String())
}

// VTE-F04: CUD unconditionally clamps cursor to margin.bottom. Per DEC VT510 spec,
// if cursor is already below the scroll region, CUD should stop at the last screen
// line, not the bottom margin. Same issue affects CUF and CUB.
func TestCUD_OutsideScrollRegion(t *testing.T) {
	vt := New()
	vt.Resize(4, 10)
	vt.mode = 0

	// Set scroll region to rows 2..5 (0-indexed)
	vt.margin.top = 2
	vt.margin.bottom = 5

	// Place cursor below the scroll region (row 7)
	vt.cursor.row = 7
	vt.cud(5)
	// Should clamp to last screen line (row 9), NOT to margin.bottom (row 5)
	assert.Equal(t, row(9), vt.cursor.row)
}

func TestCUD_InsideScrollRegion(t *testing.T) {
	vt := New()
	vt.Resize(4, 10)
	vt.mode = 0

	vt.margin.top = 2
	vt.margin.bottom = 5

	// Cursor inside scroll region
	vt.cursor.row = 3
	vt.cud(5)
	// Should clamp to margin.bottom (row 5)
	assert.Equal(t, row(5), vt.cursor.row)
}

func TestCUF_OutsideScrollRegion(t *testing.T) {
	vt := New()
	vt.Resize(10, 1)
	vt.mode = 0

	// Set left/right margins
	vt.margin.left = 2
	vt.margin.right = 5

	// Place cursor to the right of the scroll region (col 7)
	vt.cursor.col = 7
	vt.cuf(5)
	// Should clamp to last screen column (9), NOT margin.right (5)
	assert.Equal(t, column(9), vt.cursor.col)
}

func TestCUF_InsideScrollRegion(t *testing.T) {
	vt := New()
	vt.Resize(10, 1)
	vt.mode = 0

	vt.margin.left = 2
	vt.margin.right = 5

	// Cursor inside scroll region
	vt.cursor.col = 3
	vt.cuf(5)
	// Should clamp to margin.right (5)
	assert.Equal(t, column(5), vt.cursor.col)
}

func TestCUB_OutsideScrollRegion(t *testing.T) {
	vt := New()
	vt.Resize(10, 1)
	vt.mode = 0

	// Set left/right margins
	vt.margin.left = 2
	vt.margin.right = 5

	// Place cursor to the left of the scroll region (col 1)
	vt.cursor.col = 1
	vt.cub(5)
	// Should clamp to column 0, NOT margin.left (2)
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestCUB_InsideScrollRegion(t *testing.T) {
	vt := New()
	vt.Resize(10, 1)
	vt.mode = 0

	vt.margin.left = 2
	vt.margin.right = 5

	// Cursor inside scroll region
	vt.cursor.col = 4
	vt.cub(5)
	// Should clamp to margin.left (2)
	assert.Equal(t, column(2), vt.cursor.col)
}

// VTE-F05: CNL (Cursor Next Line) incorrectly scrolls at bottom margin.
// Per ECMA-48 §8.3.20, CNL should move cursor down Ps lines to column 1,
// stopping at the bottom margin WITHOUT scrolling.
func TestCNL_NoScroll(t *testing.T) {
	vt := New()
	vt.Resize(4, 4)
	vt.mode = 0

	// Fill all 4 rows
	for r := 0; r < 4; r++ {
		vt.cursor.row = row(r)
		vt.cursor.col = 0
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
	}
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", vt.String())

	// Place cursor at bottom margin (row 3) and CNL 5
	vt.cursor.row = 3
	vt.cursor.col = 2
	vt.cnl(5)

	// Cursor should stop at bottom margin row (3), col at left margin (0)
	assert.Equal(t, row(3), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	// Screen content must NOT have scrolled
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", vt.String())
}

// VTE-F05 regression: CNL from a row above bottom margin
func TestCNL_FromAbove(t *testing.T) {
	vt := New()
	vt.Resize(4, 4)
	vt.mode = 0

	for r := 0; r < 4; r++ {
		vt.cursor.row = row(r)
		vt.cursor.col = 0
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
	}

	// CNL 2 from row 1 → should move to row 3, col 0
	vt.cursor.row = 1
	vt.cursor.col = 2
	vt.cnl(2)
	assert.Equal(t, row(3), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
	// No scrolling
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", vt.String())
}

// VTE-F06: CPL (Cursor Preceding Line) incorrectly scrolls at top margin.
// Per ECMA-48 §8.3.13, CPL should move cursor up Ps lines to column 1,
// stopping at the top margin WITHOUT scrolling.
func TestCPL_NoScroll(t *testing.T) {
	vt := New()
	vt.Resize(4, 4)
	vt.mode = 0

	for r := 0; r < 4; r++ {
		vt.cursor.row = row(r)
		vt.cursor.col = 0
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
	}
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", vt.String())

	// Place cursor at top margin (row 0) and CPL 5
	vt.cursor.row = 0
	vt.cursor.col = 2
	vt.cpl(5)

	// Cursor should stop at top margin row (0), col at left margin (0)
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	// Screen content must NOT have scrolled
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", vt.String())
}

// VTE-F06 regression: CPL from a row below top margin
func TestCPL_FromBelow(t *testing.T) {
	vt := New()
	vt.Resize(4, 4)
	vt.mode = 0

	for r := 0; r < 4; r++ {
		vt.cursor.row = row(r)
		vt.cursor.col = 0
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
		vt.print(rune('a' + r))
	}

	// CPL 2 from row 3 → should move to row 1, col 0
	vt.cursor.row = 3
	vt.cursor.col = 2
	vt.cpl(2)
	assert.Equal(t, row(1), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
	// No scrolling
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", vt.String())
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

// VTE-F07: Private DSR 5 (CSI ? 5 n) responds with \x1b[?13n which means
// "no printer" (the response for CSI ? 15 n). The correct response for a
// general DEC status query is \x1b[?0n ("no malfunction detected").
func TestPrivateDSR5_Response(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	pty := &mockPty{}
	vt.pty = pty

	vt.csi("?n", []int{5})

	assert.Equal(t, "\x1b[?0n", pty.String())
}

// VTE-F07 regression: Private DSR 6 (cursor position report) should still work.
func TestPrivateDSR6_Response(t *testing.T) {
	vt := New()
	vt.Resize(10, 10)
	pty := &mockPty{}
	vt.pty = pty

	vt.cursor.row = 2
	vt.cursor.col = 5
	vt.csi("?n", []int{6})

	assert.Equal(t, "\x1b[?3;6R", pty.String())
}

// VTE-F07 regression: Standard (non-private) DSR 5 should be unaffected.
func TestStandardDSR5_Response(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	pty := &mockPty{}
	vt.pty = pty

	vt.csi("n", []int{5})

	assert.Equal(t, "\x1b[0n", pty.String())
}

// VTE-F10: DECSED/DECSEL erase all cells unconditionally. selectiveErase()
// should only erase cells that do NOT have the DECSCA protected attribute.
// Without the fix, protected cells are erased like any other cell.
func TestDECSED_SkipsProtectedCells(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')
	assert.Equal(t, "abcd", vt.String())

	// Mark cell at column 1 ('b') as protected
	vt.activeScreen[0][1].protected = true

	// Selective erase entire display (CSI ? 2 J)
	vt.cursor.col = 0
	vt.decsed(2)

	// 'b' should survive (protected), all others should be erased
	assert.Equal(t, rune(0), vt.activeScreen[0][0].content)
	assert.Equal(t, 'b', vt.activeScreen[0][1].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][2].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][3].content)
}

// VTE-F10: DECSEL should also skip protected cells on the current line.
func TestDECSEL_SkipsProtectedCells(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')

	// Protect columns 0 and 3
	vt.activeScreen[0][0].protected = true
	vt.activeScreen[0][3].protected = true

	// Selective erase entire line (CSI ? 2 K)
	vt.cursor.col = 0
	vt.decsel(2)

	// Protected cells should survive
	assert.Equal(t, 'a', vt.activeScreen[0][0].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][1].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][2].content)
	assert.Equal(t, 'd', vt.activeScreen[0][3].content)
}

// VTE-F10 regression: regular erase (ED/EL) should still erase ALL cells
// including protected ones.
func TestED_IgnoresProtection(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	vt.print('a')
	vt.print('b')
	vt.print('c')
	vt.print('d')

	// Protect cell at column 1
	vt.activeScreen[0][1].protected = true

	// Regular erase entire display (CSI 2 J) — should erase everything
	vt.cursor.col = 0
	vt.ed(2)

	assert.Equal(t, "    ", vt.String())
}

// VTE-F10: DECSCA (CSI Ps " q) should set the protection attribute on
// subsequently printed characters.
func TestDECSCA_ProtectsNewCharacters(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	vt.mode = 0

	// Enable DECSCA protection
	vt.csi("\"q", []int{1})

	vt.print('a')
	vt.print('b')

	// Disable DECSCA protection
	vt.csi("\"q", []int{0})

	vt.print('c')
	vt.print('d')

	assert.Equal(t, "abcd", vt.String())

	// 'a' and 'b' should be protected, 'c' and 'd' should not
	assert.True(t, vt.activeScreen[0][0].protected)
	assert.True(t, vt.activeScreen[0][1].protected)
	assert.False(t, vt.activeScreen[0][2].protected)
	assert.False(t, vt.activeScreen[0][3].protected)

	// Selective erase should only erase 'c' and 'd'
	vt.cursor.col = 0
	vt.decsed(2)

	assert.Equal(t, 'a', vt.activeScreen[0][0].content)
	assert.Equal(t, 'b', vt.activeScreen[0][1].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][2].content)
	assert.Equal(t, rune(0), vt.activeScreen[0][3].content)
}

// VTE-F22: XTWINOPS (CSI Ps t) is not handled. This includes window size
// reporting (Ps=18 → report size in chars, Ps=14 → report size in pixels)
// used by applications like vim and tmux to query terminal dimensions.
func TestXTWINOPS_ReportSizeChars(t *testing.T) {
	vt := New()
	vt.Resize(80, 24)
	pty := &mockPty{}
	vt.pty = pty

	// CSI 18 t → report size in characters: ESC [ 8 ; height ; width t
	vt.csi("t", []int{18})

	assert.Equal(t, "\x1b[8;24;80t", pty.String())
}

// VTE-F22 regression: CSI 14 t should report pixel dimensions (approximated).
func TestXTWINOPS_ReportSizePixels(t *testing.T) {
	vt := New()
	vt.Resize(80, 24)
	pty := &mockPty{}
	vt.pty = pty

	// CSI 14 t → report size in pixels: ESC [ 4 ; height_px ; width_px t
	// We approximate pixels as chars * a default cell size (e.g., 8x16)
	vt.csi("t", []int{14})

	assert.Equal(t, "\x1b[4;384;640t", pty.String())
}

// VTE-F22 regression: unknown CSI t params should be silently ignored.
func TestXTWINOPS_UnknownParamIgnored(t *testing.T) {
	vt := New()
	vt.Resize(80, 24)
	pty := &mockPty{}
	vt.pty = pty

	// Unknown param should not produce output or panic
	assert.NotPanics(t, func() {
		vt.csi("t", []int{99})
	})
	assert.Equal(t, "", pty.String())
}
