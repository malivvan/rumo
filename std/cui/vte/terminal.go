package vte

import (
	"os/exec"
	"sync"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

type Terminal struct {
	*cui.Box

	term          *VT
	running       bool
	cmd           *exec.Cmd
	app           *cui.App
	w             int
	h             int
	mouseCaptured bool

	sync.RWMutex
}

func NewTerminal(app *cui.App, cmd *exec.Cmd) *Terminal {
	t := &Terminal{
		Box:  cui.NewBox(),
		term: New(),
		app:  app,
		cmd:  cmd,
	}
	return t
}

func (t *Terminal) Draw(s tcell.Screen) {
	if !t.GetVisible() {
		return
	}
	t.Box.Draw(s)
	t.Lock()
	defer t.Unlock()

	x, y, w, h := t.GetInnerRect()
	view := cui.NewViewPort(s, x, y, w, h)
	t.term.SetSurface(view)
	if w != t.w || h != t.h {
		t.w = w
		t.h = h
		t.term.Resize(w, h)
	}

	if !t.running {
		err := t.term.Start(t.cmd)
		if err != nil {
			panic(err)
		}
		t.term.Attach(t.HandleEvent)
		t.running = true
	}
	if t.HasFocus() {
		cy, cx, style, vis := t.term.Cursor()
		if vis {
			s.ShowCursor(cx+x, cy+y)
			s.SetCursorStyle(style)
		} else {
			s.HideCursor()
		}
	}
	t.term.Draw()
}

func (t *Terminal) HandleEvent(ev tcell.Event) {
	switch ev.(type) {
	case *EventRedraw:
		go func() {
			t.app.QueueUpdateDraw(func() {})
		}()
	}
}

func (t *Terminal) InputHandler() func(event *tcell.EventKey, setFocus func(p cui.Widget)) {
	return t.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p cui.Widget)) {
		t.term.HandleEvent(event)
	})
}

func (t *Terminal) MouseHandler() func(action cui.MouseAction, event *tcell.EventMouse, setFocus func(p cui.Widget)) (consumed bool, capture cui.Widget) {
	return t.WrapMouseHandler(func(action cui.MouseAction, event *tcell.EventMouse, setFocus func(p cui.Widget)) (consumed bool, capture cui.Widget) {
		if !forwardMouseAction(action) {
			return false, nil
		}

		localEvent, shouldForward := t.translateMouseEvent(event)
		if !shouldForward {
			return false, nil
		}

		switch action {
		case cui.MouseLeftDown, cui.MouseMiddleDown, cui.MouseRightDown:
			setFocus(t)
			t.Lock()
			t.mouseCaptured = true
			t.Unlock()
			capture = t
		case cui.MouseLeftUp, cui.MouseMiddleUp, cui.MouseRightUp:
			t.Lock()
			t.mouseCaptured = false
			t.Unlock()
		case cui.MouseMove:
			if localEvent.Buttons() != tcell.ButtonNone {
				t.RLock()
				captured := t.mouseCaptured
				t.RUnlock()
				if captured {
					capture = t
				}
			}
		}

		t.term.HandleEvent(localEvent)
		return true, capture
	})
}

func forwardMouseAction(action cui.MouseAction) bool {
	switch action {
	case cui.MouseMove,
		cui.MouseLeftDown,
		cui.MouseLeftUp,
		cui.MouseMiddleDown,
		cui.MouseMiddleUp,
		cui.MouseRightDown,
		cui.MouseRightUp,
		cui.MouseScrollUp,
		cui.MouseScrollDown,
		cui.MouseScrollLeft,
		cui.MouseScrollRight:
		return true
	default:
		return false
	}
}

func (t *Terminal) translateMouseEvent(event *tcell.EventMouse) (*tcell.EventMouse, bool) {
	x, y := event.Position()
	innerX, innerY, width, height := t.GetInnerRect()
	if width <= 0 || height <= 0 {
		return nil, false
	}

	inside := x >= innerX && x < innerX+width && y >= innerY && y < innerY+height
	t.RLock()
	captured := t.mouseCaptured
	t.RUnlock()
	if !inside && !captured {
		return nil, false
	}

	if x < innerX {
		x = innerX
	} else if x >= innerX+width {
		x = innerX + width - 1
	}
	if y < innerY {
		y = innerY
	} else if y >= innerY+height {
		y = innerY + height - 1
	}

	return tcell.NewEventMouse(x-innerX, y-innerY, event.Buttons(), event.Modifiers()), true
}
