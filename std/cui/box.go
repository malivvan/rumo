package cui

import (
	"sync"

	"github.com/gdamore/tcell/v3"
)

// Box is the base Widget for all widgets. It has a background color and
// optional surrounding elements such as a border and a title. It does not have
// inner text. Widgets embed Box and draw their text over it.
type Box struct {
	// The position of the rect.
	x, y, width, height int

	// Padding.
	paddingTop, paddingBottom, paddingLeft, paddingRight int

	// The inner rect reserved for the box's content.
	innerX, innerY, innerWidth, innerHeight int

	// Whether or not the box is visible.
	visible bool

	// The border color when the box has focus.
	borderColorFocused tcell.Color

	// The box's background color.
	backgroundColor tcell.Color

	// Whether or not the box's background is transparent.
	backgroundTransparent bool

	// Whether or not a border is drawn, reducing the box's space for content by
	// two in width and height.
	border bool

	// The color of the border.
	borderColor tcell.Color

	// The style attributes of the border.
	borderAttributes tcell.AttrMask

	// The title. Only visible if there is a border, too.
	title []byte

	// The color of the title.
	titleColor tcell.Color

	// The alignment of the title.
	titleAlign int

	// Provides a way to find out if this box has focus. We always go through
	// this interface because it may be overridden by implementing classes.
	focus Focusable

	// Whether or not this box has focus.
	hasFocus bool

	// Whether or not this box shows its focus.
	showFocus bool

	// An optional capture function which receives a key event and returns the
	// event to be forwarded to the widget's default input handler (nil if
	// nothing should be forwarded).
	inputCapture func(event *tcell.EventKey) *tcell.EventKey

	// An optional function which is called before the box is drawn.
	draw func(screen tcell.Screen, x, y, width, height int) (int, int, int, int)

	// An optional capture function which receives a mouse event and returns the
	// event to be forwarded to the widget's default mouse event handler (at
	// least one nil if nothing should be forwarded).
	mouseCapture func(action MouseAction, event *tcell.EventMouse) (MouseAction, *tcell.EventMouse)

	l sync.RWMutex
}

// NewBox returns a Box without a border.
func NewBox() *Box {
	b := &Box{
		width:              15,
		height:             10,
		visible:            true,
		backgroundColor:    Styles.WidgetBackgroundColor,
		borderColor:        Styles.BorderColor,
		titleColor:         Styles.TitleColor,
		borderColorFocused: ColorUnset,
		titleAlign:         AlignCenter,
		showFocus:          true,
	}
	b.focus = b
	b.updateInnerRect()
	return b
}

func (b *Box) updateInnerRect() {
	x, y, width, height := b.x, b.y, b.width, b.height

	// Subtract border space
	if b.border {
		x++
		y++
		width -= 2
		height -= 2
	}

	// Subtract padding
	x, y, width, height =
		x+b.paddingLeft,
		y+b.paddingTop,
		width-b.paddingLeft-b.paddingRight,
		height-b.paddingTop-b.paddingBottom

	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}

	b.innerX, b.innerY, b.innerWidth, b.innerHeight = x, y, width, height
}

// GetPadding returns the size of the padding around the box content.
func (b *Box) GetPadding() (top, bottom, left, right int) {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.paddingTop, b.paddingBottom, b.paddingLeft, b.paddingRight
}

// SetPadding sets the size of the padding around the box content.
func (b *Box) SetPadding(top, bottom, left, right int) {
	b.l.Lock()
	defer b.l.Unlock()

	b.paddingTop, b.paddingBottom, b.paddingLeft, b.paddingRight = top, bottom, left, right

	b.updateInnerRect()
}

// GetRect returns the current position of the rectangle, x, y, width, and
// height.
func (b *Box) GetRect() (int, int, int, int) {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.x, b.y, b.width, b.height
}

// GetInnerRect returns the position of the inner rectangle (x, y, width,
// height), without the border and without any padding. Width and height values
// will clamp to 0 and thus never be negative.
func (b *Box) GetInnerRect() (int, int, int, int) {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.innerX, b.innerY, b.innerWidth, b.innerHeight
}

// SetRect sets a new position of the widget. Note that this has no effect
// if this widget is part of a layout (e.g. Flex, Grid) or if it was added
// like this:
//
//	application.SetRoot(b, true)
func (b *Box) SetRect(x, y, width, height int) {
	b.l.Lock()
	defer b.l.Unlock()

	b.x, b.y, b.width, b.height = x, y, width, height

	b.updateInnerRect()
}

// SetVisible sets the flag indicating whether or not the box is visible.
func (b *Box) SetVisible(v bool) {
	b.l.Lock()
	defer b.l.Unlock()

	b.visible = v
}

// GetVisible returns a value indicating whether or not the box is visible.
func (b *Box) GetVisible() bool {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.visible
}

// SetDrawFunc sets a callback function which is invoked after the box widget
// has been drawn. This allows you to add a more individual style to the box
// (and all widgets which extend it).
//
// The function is provided with the box's dimensions (set via SetRect()). It
// must return the box's inner dimensions (x, y, width, height) which will be
// returned by GetInnerRect(), used by descendent widgets to draw their own
// content.
func (b *Box) SetDrawFunc(handler func(screen tcell.Screen, x, y, width, height int) (int, int, int, int)) {
	b.l.Lock()
	defer b.l.Unlock()

	b.draw = handler
}

// GetDrawFunc returns the callback function which was installed with
// SetDrawFunc() or nil if no such function has been installed.
func (b *Box) GetDrawFunc() func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.draw
}

// WrapInputHandler wraps an input handler (see InputHandler()) with the
// functionality to capture input (see SetInputCapture()) before passing it
// on to the provided (default) input handler.
//
// This is only meant to be used by subclassing widgets.
func (b *Box) WrapInputHandler(inputHandler func(*tcell.EventKey, func(w Widget))) func(*tcell.EventKey, func(w Widget)) {
	return func(event *tcell.EventKey, setFocus func(w Widget)) {
		b.l.RLock()
		capture := b.inputCapture
		b.l.RUnlock()
		if capture != nil {
			event = capture(event)
		}
		if event != nil && inputHandler != nil {
			inputHandler(event, setFocus)
		}
	}
}

// InputHandler returns nil.
func (b *Box) InputHandler() func(event *tcell.EventKey, setFocus func(w Widget)) {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.WrapInputHandler(nil)
}

// SetInputCapture installs a function which captures key events before they are
// forwarded to the widget's default key event handler. This function can
// then choose to forward that key event (or a different one) to the default
// handler by returning it. If nil is returned, the default handler will not
// be called.
//
// Providing a nil handler will remove a previously existing handler.
//
// Note that this function will not have an effect on widgets composed of
// other widgets, such as Form, Flex, or Grid. Key events are only captured
// by the widgets that have focus (e.g. InputField) and only one widget
// can have focus at a time. Composing widgets such as Form pass the focus on
// to their contained widgets and thus never receive any key events
// themselves. Therefore, they cannot intercept key events.
func (b *Box) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	b.l.Lock()
	defer b.l.Unlock()

	b.inputCapture = capture
}

// GetInputCapture returns the function installed with SetInputCapture() or nil
// if no such function has been installed.
func (b *Box) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.inputCapture
}

// WrapMouseHandler wraps a mouse event handler (see MouseHandler()) with the
// functionality to capture mouse events (see SetMouseCapture()) before passing
// them on to the provided (default) event handler.
//
// This is only meant to be used by subclassing widgets.
func (b *Box) WrapMouseHandler(mouseHandler func(MouseAction, *tcell.EventMouse, func(w Widget)) (bool, Widget)) func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
	return func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
		b.l.RLock()
		captureHandler := b.mouseCapture
		b.l.RUnlock()
		if captureHandler != nil {
			action, event = captureHandler(action, event)
		}
		if event != nil && mouseHandler != nil {
			consumed, capture = mouseHandler(action, event, setFocus)
		}
		return
	}
}

// MouseHandler returns nil.
func (b *Box) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
	return b.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
		if action == MouseLeftClick && b.InRect(event.Position()) {
			setFocus(b)
			consumed = true
		}
		return
	})
}

// SetMouseCapture sets a function which captures mouse events (consisting of
// the original tcell mouse event and the semantic mouse action) before they are
// forwarded to the widget's default mouse event handler. This function can
// then choose to forward that event (or a different one) by returning it or
// returning a nil mouse event, in which case the default handler will not be
// called.
//
// Providing a nil handler will remove a previously existing handler.
func (b *Box) SetMouseCapture(capture func(action MouseAction, event *tcell.EventMouse) (MouseAction, *tcell.EventMouse)) {
	b.l.Lock()
	defer b.l.Unlock()

	b.mouseCapture = capture
}

// InRect returns true if the given coordinate is within the bounds of the box's
// rectangle.
func (b *Box) InRect(x, y int) bool {
	rectX, rectY, width, height := b.GetRect()
	return x >= rectX && x < rectX+width && y >= rectY && y < rectY+height
}

// GetMouseCapture returns the function installed with SetMouseCapture() or nil
// if no such function has been installed.
func (b *Box) GetMouseCapture() func(action MouseAction, event *tcell.EventMouse) (MouseAction, *tcell.EventMouse) {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.mouseCapture
}

// SetBackgroundColor sets the box's background color.
func (b *Box) SetBackgroundColor(color tcell.Color) {
	b.l.Lock()
	defer b.l.Unlock()

	b.backgroundColor = color
}

// GetBackgroundColor returns the box's background color.
func (b *Box) GetBackgroundColor() tcell.Color {
	b.l.RLock()
	defer b.l.RUnlock()
	return b.backgroundColor
}

// SetBackgroundTransparent sets the flag indicating whether or not the box's
// background is transparent. The screen is not cleared before drawing the
// application. Overlaying transparent widgets directly onto the screen may
// result in artifacts. To resolve this, add a blank, non-transparent Box to
// the bottom layer of the interface via Panels, or set a handler via
// SetBeforeDrawFunc which clears the screen.
func (b *Box) SetBackgroundTransparent(transparent bool) {
	b.l.Lock()
	defer b.l.Unlock()

	b.backgroundTransparent = transparent
}

// GetBorder returns a value indicating whether the box have a border
// or not.
func (b *Box) GetBorder() bool {
	b.l.RLock()
	defer b.l.RUnlock()
	return b.border
}

// SetBorder sets the flag indicating whether or not the box should have a
// border.
func (b *Box) SetBorder(show bool) {
	b.l.Lock()
	defer b.l.Unlock()

	b.border = show

	b.updateInnerRect()
}

// SetBorderColor sets the box's border color.
func (b *Box) SetBorderColor(color tcell.Color) {
	b.l.Lock()
	defer b.l.Unlock()

	b.borderColor = color
}

// SetBorderColorFocused sets the box's border color when the box is focused.
func (b *Box) SetBorderColorFocused(color tcell.Color) {
	b.l.Lock()
	defer b.l.Unlock()
	b.borderColorFocused = color
}

// SetBorderAttributes sets the border's style attributes. You can combine
// different attributes using bitmask operations:
//
//	box.SetBorderAttributes(tcell.AttrUnderline | tcell.AttrBold)
func (b *Box) SetBorderAttributes(attr tcell.AttrMask) {
	b.l.Lock()
	defer b.l.Unlock()

	b.borderAttributes = attr
}

// SetTitle sets the box's title.
func (b *Box) SetTitle(title string) {
	b.l.Lock()
	defer b.l.Unlock()

	b.title = []byte(title)
}

// GetTitle returns the box's current title.
func (b *Box) GetTitle() string {
	b.l.RLock()
	defer b.l.RUnlock()

	return string(b.title)
}

// SetTitleColor sets the box's title color.
func (b *Box) SetTitleColor(color tcell.Color) {
	b.l.Lock()
	defer b.l.Unlock()

	b.titleColor = color
}

// SetTitleAlign sets the alignment of the title, one of AlignLeft, AlignCenter,
// or AlignRight.
func (b *Box) SetTitleAlign(align int) {
	b.l.Lock()
	defer b.l.Unlock()

	b.titleAlign = align
}

// Draw draws this widget onto the screen.
func (b *Box) Draw(screen tcell.Screen) {
	b.drawWithOverrides(screen, nil, nil)
}

func (b *Box) drawWithOverrides(screen tcell.Screen, backgroundOverride, borderOverride *tcell.Color) {
	b.l.RLock()
	visible := b.visible
	x, y, width, height := b.x, b.y, b.width, b.height
	backgroundColor := b.backgroundColor
	backgroundTransparent := b.backgroundTransparent
	borderEnabled := b.border
	borderColor := b.borderColor
	borderColorFocused := b.borderColorFocused
	borderAttributes := b.borderAttributes
	title := append([]byte(nil), b.title...)
	titleColor := b.titleColor
	titleAlign := b.titleAlign
	focus := b.focus
	hasFocus := b.hasFocus
	showFocus := b.showFocus
	draw := b.draw
	b.l.RUnlock()

	if backgroundOverride != nil {
		backgroundColor = *backgroundOverride
	}
	if borderOverride != nil {
		borderColor = *borderOverride
	}

	if !visible || width <= 0 || height <= 0 {
		return
	}

	if focus != nil && focus != b {
		hasFocus = focus.HasFocus()
	}

	def := tcell.StyleDefault
	background := def.Background(backgroundColor)
	if !backgroundTransparent {
		for drawY := y; drawY < y+height; drawY++ {
			for drawX := x; drawX < x+width; drawX++ {
				screen.Put(drawX, drawY, " ", background)
			}
		}
	}

	if borderEnabled && width >= 2 && height >= 2 {
		border := SetAttributes(background.Foreground(borderColor), borderAttributes)
		var vertical, horizontal, topLeft, topRight, bottomLeft, bottomRight rune

		if hasFocus && borderColorFocused != ColorUnset {
			border = SetAttributes(background.Foreground(borderColorFocused), borderAttributes)
		}

		if hasFocus && showFocus {
			horizontal = Borders.HorizontalFocus
			vertical = Borders.VerticalFocus
			topLeft = Borders.TopLeftFocus
			topRight = Borders.TopRightFocus
			bottomLeft = Borders.BottomLeftFocus
			bottomRight = Borders.BottomRightFocus
		} else {
			horizontal = Borders.Horizontal
			vertical = Borders.Vertical
			topLeft = Borders.TopLeft
			topRight = Borders.TopRight
			bottomLeft = Borders.BottomLeft
			bottomRight = Borders.BottomRight
		}
		for drawX := x + 1; drawX < x+width-1; drawX++ {
			screen.Put(drawX, y, string(horizontal), border)
			screen.Put(drawX, y+height-1, string(horizontal), border)
		}
		for drawY := y + 1; drawY < y+height-1; drawY++ {
			screen.Put(x, drawY, string(vertical), border)
			screen.Put(x+width-1, drawY, string(vertical), border)
		}
		screen.Put(x, y, string(topLeft), border)
		screen.Put(x+width-1, y, string(topRight), border)
		screen.Put(x, y+height-1, string(bottomLeft), border)
		screen.Put(x+width-1, y+height-1, string(bottomRight), border)
	}

	if len(title) > 0 && width >= 4 {
		printed, _ := Print(screen, title, x+1, y, width-2, titleAlign, titleColor)
		if len(title)-printed > 0 && printed > 0 {
			_, style, _ := screen.Get(x+width-2, y)
			fg := style.GetForeground()
			Print(screen, []byte(string(SemigraphicsHorizontalEllipsis)), x+width-2, y, 1, AlignLeft, fg)
		}
	}

	if draw != nil {
		innerX, innerY, innerWidth, innerHeight := draw(screen, x, y, width, height)
		b.l.Lock()
		b.innerX, b.innerY, b.innerWidth, b.innerHeight = innerX, innerY, innerWidth, innerHeight
		b.l.Unlock()
	}
}

// ShowFocus sets the flag indicating whether or not the borders of this
// widget should change thickness when focused.
func (b *Box) ShowFocus(showFocus bool) {
	b.l.Lock()
	defer b.l.Unlock()

	b.showFocus = showFocus
}

// Focus is called when this widget receives focus.
func (b *Box) Focus(delegate func(w Widget)) {
	b.l.Lock()
	defer b.l.Unlock()

	b.hasFocus = true
}

// Blur is called when this widget loses focus.
func (b *Box) Blur() {
	b.l.Lock()
	defer b.l.Unlock()

	b.hasFocus = false
}

// HasFocus returns whether or not this widget has focus.
func (b *Box) HasFocus() bool {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.hasFocus
}

// GetFocusable returns the item's Focusable.
func (b *Box) GetFocusable() Focusable {
	b.l.RLock()
	defer b.l.RUnlock()

	return b.focus
}

// GetBorderPadding returns the size of the padding around the box content.
//
// Deprecated: This function is provided for backwards compatibility.
// Developers should use GetPadding instead.
func (b *Box) GetBorderPadding() (top, bottom, left, right int) {
	return b.GetPadding()
}

// SetBorderPadding sets the size of the padding around the box content.
//
// Deprecated: This function is provided for backwards compatibility.
// Developers should use SetPadding instead.
func (b *Box) SetBorderPadding(top, bottom, left, right int) {
	b.SetPadding(top, bottom, left, right)
}
