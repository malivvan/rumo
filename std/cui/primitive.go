package cui

import "github.com/gdamore/tcell/v3"

// Widget is the top-most interface for all graphical widgets.
type Widget interface {
	// Draw draws this widget onto the screen. Implementers can call the
	// screen's ShowCursor() function but should only do so when they have focus.
	// (They will need to keep track of this themselves.)
	Draw(screen tcell.Screen)

	// GetRect returns the current position of the widget, x, y, width, and
	// height.
	GetRect() (int, int, int, int)

	// SetRect sets a new position of the widget.
	SetRect(x, y, width, height int) Widget

	// GetVisible returns whether or not the widget is visible.
	GetVisible() bool

	// SetVisible sets whether or not the widget is visible.
	SetVisible(v bool) Widget

	// InputHandler returns a handler which receives key events when it has focus.
	// It is called by the App class.
	//
	// A value of nil may also be returned, in which case this widget cannot
	// receive focus and will not process any key events.
	//
	// The handler will receive the key event and a function that allows it to
	// set the focus to a different widget, so that future key events are sent
	// to that widget.
	//
	// The App's Draw() function will be called automatically after the
	// handler returns.
	//
	// The Box class provides functionality to intercept keyboard input. If you
	// subclass from Box, it is recommended that you wrap your handler using
	// Box.WrapInputHandler() so you inherit that functionality.
	InputHandler() func(event *tcell.EventKey, setFocus func(w Widget))

	// Focus is called by the application when the widget receives focus.
	// Implementers may call delegate() to pass the focus on to another widget.
	Focus(delegate func(w Widget))

	// Blur is called by the application when the widget loses focus.
	Blur()

	// GetFocusable returns the item's Focusable.
	GetFocusable() Focusable

	// MouseHandler returns a handler which receives mouse events.
	// It is called by the App class.
	//
	// A value of nil may also be returned to stop the downward propagation of
	// mouse events.
	//
	// The Box class provides functionality to intercept mouse events. If you
	// subclass from Box, it is recommended that you wrap your handler using
	// Box.WrapMouseHandler() so you inherit that functionality.
	MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget)
}
