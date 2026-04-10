package vte

import (
	"io"
	"os"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

// captureSurface is a test Surface that captures SetContent calls.
type captureSurface struct {
	w, h  int
	cells [][]capturedCell
}

type capturedCell struct {
	ch    rune
	comb  []rune
	style tcell.Style
	set   bool
}

func (s *captureSurface) SetContent(x, y int, ch rune, comb []rune, style tcell.Style) {
	if s.cells == nil {
		s.cells = make([][]capturedCell, s.h)
		for i := range s.cells {
			s.cells[i] = make([]capturedCell, s.w)
		}
	}
	if y >= 0 && y < s.h && x >= 0 && x < s.w {
		s.cells[y][x] = capturedCell{ch: ch, comb: comb, style: style, set: true}
	}
}

func (s *captureSurface) Size() (int, int) {
	return s.w, s.h
}

func TestCUPDefaultsAndOriginMode(t *testing.T) {
	vt := New()
	vt.Resize(10, 5)

	vt.cursor.row = 4
	vt.cursor.col = 9
	vt.cup([]int{0, 0})
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.margin.top = 1
	vt.margin.bottom = 3
	vt.mode |= decom
	vt.cup([]int{1, 1})
	assert.Equal(t, row(1), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.cup([]int{999, 999})
	assert.Equal(t, row(3), vt.cursor.row)
	assert.Equal(t, column(9), vt.cursor.col)
}

func TestDECSTBMDefaultsAndHome(t *testing.T) {
	vt := New()
	vt.Resize(10, 4)

	vt.cursor.row = 3
	vt.cursor.col = 7
	vt.decstbm([]int{0, 3})
	assert.Equal(t, row(0), vt.margin.top)
	assert.Equal(t, row(2), vt.margin.bottom)
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.mode |= decom
	vt.cursor.row = 0
	vt.cursor.col = 9
	vt.decstbm([]int{2, 4})
	assert.Equal(t, row(1), vt.margin.top)
	assert.Equal(t, row(3), vt.margin.bottom)
	assert.Equal(t, row(1), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestPrivateDSRResponses(t *testing.T) {
	vt := New()
	vt.Resize(10, 5)
	vt.margin.top = 1
	vt.margin.bottom = 3
	vt.mode |= decom
	vt.cursor.row = 2
	vt.cursor.col = 4

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	defer func() {
		_ = r.Close()
	}()
	vt.pty = w

	vt.csi("?n", []int{5})
	vt.csi("?n", []int{6})
	assert.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	assert.Equal(t, "\x1b[?0n\x1b[?2;5R", string(out))
}

func TestDECOMHomesCursor(t *testing.T) {
	vt := New()
	vt.Resize(10, 5)
	vt.margin.top = 2
	vt.margin.bottom = 4
	vt.cursor.row = 4
	vt.cursor.col = 8

	vt.decset([]int{6})
	assert.Equal(t, row(2), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)

	vt.cursor.row = 4
	vt.cursor.col = 8
	vt.decrst([]int{6})
	assert.Equal(t, row(0), vt.cursor.row)
	assert.Equal(t, column(0), vt.cursor.col)
}

func TestAlternateScreenModeVariants(t *testing.T) {
	t.Run("47 and 1047 switch screens without clobbering primary", func(t *testing.T) {
		vt := New()
		vt.Resize(2, 1)
		vt.print('p')

		vt.decset([]int{47})
		vt.cursor.row = 0
		vt.cursor.col = 0
		vt.print('a')
		vt.decrst([]int{47})
		assert.Equal(t, "p ", vt.String())

		vt.decset([]int{1047})
		vt.cursor.row = 0
		vt.cursor.col = 0
		vt.print('b')
		vt.decrst([]int{1047})
		assert.Equal(t, "p ", vt.String())
	})

	t.Run("1048 saves and restores cursor", func(t *testing.T) {
		vt := New()
		vt.Resize(2, 2)
		vt.cursor.row = 1
		vt.cursor.col = 1

		vt.decset([]int{1048})
		vt.cursor.row = 0
		vt.cursor.col = 0
		vt.decrst([]int{1048})

		assert.Equal(t, row(1), vt.cursor.row)
		assert.Equal(t, column(1), vt.cursor.col)
	})
}

// VTE-F18: DECSET/DECRST 1004 (Send FocusIn/FocusOut events) is not handled.
// Many modern TUI applications (neovim, tmux, etc.) enable this mode to detect
// when the terminal gains or loses focus. Without it, these applications cannot
// react to focus changes.
func TestFocusEventMode_DECSET1004(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	// Focus events mode should be off by default
	assert.Equal(t, mode(0), vt.mode&focusEvents)

	// DECSET 1004 should enable focus event reporting
	vt.decset([]int{1004})
	assert.NotEqual(t, mode(0), vt.mode&focusEvents)

	// DECRST 1004 should disable focus event reporting
	vt.decrst([]int{1004})
	assert.Equal(t, mode(0), vt.mode&focusEvents)
}

// VTE-F18 regression: enabling/disabling 1004 should not affect other modes.
func TestFocusEventMode_DoesNotAffectOtherModes(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	// Enable paste mode and mouse
	vt.decset([]int{2004})
	vt.decset([]int{1000})

	// Enable then disable focus events
	vt.decset([]int{1004})
	vt.decrst([]int{1004})

	// Other modes should be unaffected
	assert.NotEqual(t, mode(0), vt.mode&paste)
	assert.NotEqual(t, mode(0), vt.mode&mouseButtons)
}

// VTE-F19: DECSET/DECRST 2026 (synchronized output) is not handled. Modern
// terminals use this to batch screen updates and prevent tearing. Applications
// that enable it will not see the expected synchronization barrier.
func TestSyncOutputMode_DECSET2026(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	// Sync output mode should be off by default
	assert.Equal(t, mode(0), vt.mode&syncOutput)

	// DECSET 2026 should enable synchronized output
	vt.decset([]int{2026})
	assert.NotEqual(t, mode(0), vt.mode&syncOutput)

	// DECRST 2026 should disable synchronized output
	vt.decrst([]int{2026})
	assert.Equal(t, mode(0), vt.mode&syncOutput)
}

// VTE-F19 regression: enabling/disabling 2026 should not affect other modes.
func TestSyncOutputMode_DoesNotAffectOtherModes(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	vt.decset([]int{2004})
	vt.decset([]int{2026})
	vt.decrst([]int{2026})

	assert.NotEqual(t, mode(0), vt.mode&paste)
}

// VTE-F20: DECSET/DECRST 5 (DECSCNM) is recognized in mode.go but has no
// rendering effect. Applications that enable reverse video mode will not see
// the screen colors inverted.
func TestDECSCNM_ReverseVideo(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	// DECSCNM should be off by default
	assert.Equal(t, mode(0), vt.mode&decscnm)

	// DECSET 5 should enable reverse video mode
	vt.decset([]int{5})
	assert.NotEqual(t, mode(0), vt.mode&decscnm)

	// DECRST 5 should disable reverse video mode
	vt.decrst([]int{5})
	assert.Equal(t, mode(0), vt.mode&decscnm)
}

// VTE-F20 regression: DECSCNM Draw() should invert fg/bg on all cells when set.
func TestDECSCNM_DrawInvertsCells(t *testing.T) {
	vt := New()
	srf := &captureSurface{w: 4, h: 1}
	vt.SetSurface(srf)
	vt.Resize(4, 1)
	vt.mode = 0

	// Print a character with known foreground/background
	fg := tcell.ColorRed
	bg := tcell.ColorBlue
	vt.cursor.attrs = tcell.StyleDefault.Foreground(fg).Background(bg)
	vt.print('A')

	// Enable DECSCNM (reverse video)
	vt.mode |= decscnm

	vt.Draw()

	// When DECSCNM is on, Draw() should invert the style (apply Reverse)
	c := srf.cells[0][0]
	assert.True(t, c.set)
	// The style should have Reverse applied
	assert.True(t, c.style.HasReverse())
}

// VTE-F24: DECSET 9 (X10 mouse compatibility) and DECSET 1005 (UTF-8 mouse
// encoding) are not handled. While SGR mouse (1006) is the modern standard,
// some legacy applications still use these older protocols.
func TestX10MouseMode_DECSET9(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	// X10 mouse mode should be off by default
	assert.Equal(t, mode(0), vt.mode&mouseX10)

	// DECSET 9 should enable X10 mouse mode
	vt.decset([]int{9})
	assert.NotEqual(t, mode(0), vt.mode&mouseX10)

	// DECRST 9 should disable X10 mouse mode
	vt.decrst([]int{9})
	assert.Equal(t, mode(0), vt.mode&mouseX10)
}

func TestUTF8MouseMode_DECSET1005(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	// UTF-8 mouse encoding should be off by default
	assert.Equal(t, mode(0), vt.mode&mouseUTF8)

	// DECSET 1005 should enable UTF-8 mouse encoding
	vt.decset([]int{1005})
	assert.NotEqual(t, mode(0), vt.mode&mouseUTF8)

	// DECRST 1005 should disable UTF-8 mouse encoding
	vt.decrst([]int{1005})
	assert.Equal(t, mode(0), vt.mode&mouseUTF8)
}

// VTE-F24 regression: enabling mouse mode 9 should not interfere with SGR mouse (1006).
func TestMouseModes_IndependentFlags(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	vt.decset([]int{9})
	vt.decset([]int{1005})
	vt.decset([]int{1006})

	assert.NotEqual(t, mode(0), vt.mode&mouseX10)
	assert.NotEqual(t, mode(0), vt.mode&mouseUTF8)
	assert.NotEqual(t, mode(0), vt.mode&mouseSGR)

	// Disabling one should not affect the others
	vt.decrst([]int{9})
	assert.Equal(t, mode(0), vt.mode&mouseX10)
	assert.NotEqual(t, mode(0), vt.mode&mouseUTF8)
	assert.NotEqual(t, mode(0), vt.mode&mouseSGR)
}
