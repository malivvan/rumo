package cui

import (
	"sync"

	"github.com/gdamore/tcell/v3"
)

// Button is labeled box that triggers an action when selected.
type Button struct {
	*Box

	// The text to be displayed before the input area.
	label []byte

	// The label color.
	labelColor tcell.Color

	// The label color when the button is in focus.
	labelColorFocused tcell.Color

	// The background color when the button is in focus.
	backgroundColorFocused tcell.Color

	// An optional function which is called when the button was selected.
	selected func()

	// An optional function which is called when the user leaves the button. A
	// key is provided indicating which key was pressed to leave (tab or backtab).
	blur func(tcell.Key)

	// An optional rune which is drawn after the label when the button is focused.
	cursorRune rune

	sync.RWMutex
}

// NewButton returns a new input field.
func NewButton(label string) *Button {
	box := NewBox()
	box.SetRect(0, 0, TaggedStringWidth(label)+4, 1)
	box.SetBackgroundColor(Styles.MoreContrastBackgroundColor)
	box.SetBorderColorFocused(Styles.PrimaryTextColor)
	return &Button{
		Box:                    box,
		label:                  []byte(label),
		labelColor:             Styles.PrimaryTextColor,
		labelColorFocused:      Styles.PrimaryTextColor,
		cursorRune:             Styles.ButtonCursorRune,
		backgroundColorFocused: Styles.ContrastBackgroundColor,
	}
}

// SetLabel sets the button text.
func (b *Button) SetLabel(label string) *Button {
	b.Lock()
	defer b.Unlock()

	b.label = []byte(label)
	return b
}

// GetLabel returns the button text.
func (b *Button) GetLabel() string {
	b.RLock()
	defer b.RUnlock()

	return string(b.label)
}

// SetLabelColor sets the color of the button text.
func (b *Button) SetLabelColor(color tcell.Color) *Button {
	b.Lock()
	defer b.Unlock()

	b.labelColor = color
	return b
}

// SetLabelColorFocused sets the color of the button text when the button is
// in focus.
func (b *Button) SetLabelColorFocused(color tcell.Color) *Button {
	b.Lock()
	defer b.Unlock()

	b.labelColorFocused = color
	b.Box.SetBorderColorFocused(color)
	return b
}

// SetCursorRune sets the rune to show within the button when it is focused.
func (b *Button) SetCursorRune(rune rune) *Button {
	b.Lock()
	defer b.Unlock()

	b.cursorRune = rune
	return b
}

// SetBackgroundColorFocused sets the background color of the button text when
// the button is in focus.
func (b *Button) SetBackgroundColorFocused(color tcell.Color) *Button {
	b.Lock()
	defer b.Unlock()

	b.backgroundColorFocused = color
	return b
}

// SetSelectedFunc sets a handler which is called when the button was selected.
func (b *Button) SetSelectedFunc(handler func()) *Button {
	b.Lock()
	defer b.Unlock()

	b.selected = handler
	return b
}

// SetBlurFunc sets a handler which is called when the user leaves the button.
// The callback function is provided with the key that was pressed, which is one
// of the following:
//
//   - KeyEscape: Leaving the button with no specific direction.
//   - KeyTab: Move to the next field.
//   - KeyBacktab: Move to the previous field.
func (b *Button) SetBlurFunc(handler func(key tcell.Key)) *Button {
	b.Lock()
	defer b.Unlock()

	b.blur = handler
	return b
}

// Draw draws this widget onto the screen.
func (b *Button) Draw(screen tcell.Screen) {
	if !b.GetVisible() {
		return
	}

	hasFocus := b.GetFocusable().HasFocus()

	b.RLock()
	label := append([]byte(nil), b.label...)
	labelColor := b.labelColor
	labelColorFocused := b.labelColorFocused
	backgroundColorFocused := b.backgroundColorFocused
	cursorRune := b.cursorRune
	b.RUnlock()

	if hasFocus {
		b.Box.drawWithOverrides(screen, &backgroundColorFocused, nil)
	} else {
		b.Box.Draw(screen)
	}

	// Draw label.
	x, y, width, height := b.GetInnerRect()
	if width > 0 && height > 0 {
		y = y + height/2
		if b.focus.HasFocus() {
			labelColor = labelColorFocused
		}
		_, pw := Print(screen, label, x, y, width, AlignCenter, labelColor)

		// Draw cursor.
		if hasFocus && cursorRune != 0 {
			cursorX := x + int(float64(width)/2+float64(pw)/2)
			if cursorX > x+width-1 {
				cursorX = x + width - 1
			} else if cursorX < x+width {
				cursorX++
			}
			Print(screen, []byte(string(cursorRune)), cursorX, y, width, AlignLeft, labelColor)
		}
	}
}

// InputHandler returns the handler for this widget.
func (b *Button) InputHandler() func(event *tcell.EventKey, setFocus func(w Widget)) {
	return b.WrapInputHandler(func(event *tcell.EventKey, setFocus func(w Widget)) {
		b.RLock()
		selected := b.selected
		blur := b.blur
		b.RUnlock()

		// Process key event.
		if HitShortcut(event, Keys.Select, Keys.Select2) {
			if selected != nil {
				selected()
			}
		} else if HitShortcut(event, Keys.Cancel, Keys.MovePreviousField, Keys.MoveNextField) {
			if blur != nil {
				blur(event.Key())
			}
		}
	})
}

// MouseHandler returns the mouse handler for this widget.
func (b *Button) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
	return b.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(w Widget)) (consumed bool, capture Widget) {
		if !b.InRect(event.Position()) {
			return false, nil
		}

		b.RLock()
		selected := b.selected
		b.RUnlock()

		// Process mouse event.
		if action == MouseLeftClick {
			setFocus(b)
			if selected != nil {
				selected()
			}
			consumed = true
		}

		return
	})
}
