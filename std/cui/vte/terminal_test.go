package vte

import (
	"io"
	"os"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
	"github.com/stretchr/testify/assert"
)

func newMousePipe(t *testing.T, term *Terminal) (*os.File, *os.File) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	term.term.pty = w
	return r, w
}

func readMousePipe(t *testing.T, r *os.File, w *os.File) string {
	t.Helper()
	assert.NoError(t, w.Close())
	defer func() {
		_ = r.Close()
	}()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(out)
}

func TestTerminalMouseHandlerForwardsInnerRectCoordinates(t *testing.T) {
	term := NewTerminal(nil, nil)
	term.SetBorder(true)
	term.SetRect(5, 7, 10, 4)
	term.term.mode |= mouseSGR

	r, w := newMousePipe(t, term)
	var focused cui.Widget

	handler := term.MouseHandler()
	consumed, capture := handler(cui.MouseLeftDown, tcell.NewEventMouse(8, 9, tcell.Button1, tcell.ModNone), func(w cui.Widget) {
		focused = w
	})

	assert.True(t, consumed)
	assert.Equal(t, term, capture)
	assert.Equal(t, term, focused)
	assert.Equal(t, "\x1b[<0;3;2M", readMousePipe(t, r, w))
}

func TestTerminalMouseHandlerForwardsMouseMove(t *testing.T) {
	term := NewTerminal(nil, nil)
	term.SetBorder(true)
	term.SetRect(5, 7, 10, 4)
	term.term.mode |= mouseSGR

	r, w := newMousePipe(t, term)
	handler := term.MouseHandler()
	consumed, capture := handler(cui.MouseMove, tcell.NewEventMouse(7, 8, tcell.ButtonNone, tcell.ModNone), func(w cui.Widget) {})

	assert.True(t, consumed)
	assert.Nil(t, capture)
	assert.Equal(t, "\x1b[<3;2;1M", readMousePipe(t, r, w))
}

func TestTerminalMouseHandlerIgnoresBorderClicks(t *testing.T) {
	term := NewTerminal(nil, nil)
	term.SetBorder(true)
	term.SetRect(5, 7, 10, 4)
	term.term.mode |= mouseSGR

	r, w := newMousePipe(t, term)
	handler := term.MouseHandler()
	consumed, capture := handler(cui.MouseLeftDown, tcell.NewEventMouse(5, 7, tcell.Button1, tcell.ModNone), func(w cui.Widget) {})

	assert.False(t, consumed)
	assert.Nil(t, capture)
	assert.Equal(t, "", readMousePipe(t, r, w))
}

func TestTerminalMouseHandlerKeepsCaptureForDragOutsideViewport(t *testing.T) {
	term := NewTerminal(nil, nil)
	term.SetBorder(true)
	term.SetRect(5, 7, 10, 4)
	term.term.mode |= mouseSGR

	r, w := newMousePipe(t, term)
	handler := term.MouseHandler()

	consumed, capture := handler(cui.MouseLeftDown, tcell.NewEventMouse(6, 8, tcell.Button1, tcell.ModNone), func(w cui.Widget) {})
	assert.True(t, consumed)
	assert.Equal(t, term, capture)

	consumed, capture = handler(cui.MouseMove, tcell.NewEventMouse(40, 40, tcell.Button1, tcell.ModNone), func(w cui.Widget) {})
	assert.True(t, consumed)
	assert.Equal(t, term, capture)

	consumed, capture = handler(cui.MouseLeftUp, tcell.NewEventMouse(40, 40, tcell.ButtonNone, tcell.ModNone), func(w cui.Widget) {})
	assert.True(t, consumed)
	assert.Nil(t, capture)

	assert.Equal(t, "\x1b[<0;1;1M\x1b[<32;8;2M\x1b[<0;8;2m", readMousePipe(t, r, w))
}
