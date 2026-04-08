// Demo code which illustrates how to implement your own widget.
package main

import (
	"fmt"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
)

// RadioButtons implements a simple widget for radio button selections.
type RadioButtons struct {
	*cui.Box
	options       []string
	currentOption int
}

// NewRadioButtons returns a new radio button widget.
func NewRadioButtons(options []string) *RadioButtons {
	return &RadioButtons{
		Box:     cui.NewBox(),
		options: options,
	}
}

// Draw draws this widget onto the screen.
func (r *RadioButtons) Draw(screen tcell.Screen) {
	r.Box.Draw(screen)
	x, y, width, height := r.GetInnerRect()

	for index, option := range r.options {
		if index >= height {
			break
		}
		radioButton := "\u25ef" // Unchecked.
		if index == r.currentOption {
			radioButton = "\u25c9" // Checked.
		}
		line := fmt.Sprintf(`%s[white]  %s`, radioButton, option)
		cui.Print(screen, []byte(line), x, y+index, width, cui.AlignLeft, color.Yellow)
	}

}

// InputHandler returns the handler for this widget.
func (r *RadioButtons) InputHandler() func(event *tcell.EventKey, setFocus func(p cui.Widget)) {
	return r.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p cui.Widget)) {
		switch event.Key() {
		case tcell.KeyUp:
			r.currentOption--
			if r.currentOption < 0 {
				r.currentOption = 0
			}
		case tcell.KeyDown:
			r.currentOption++
			if r.currentOption >= len(r.options) {
				r.currentOption = len(r.options) - 1
			}
		}
	})
}

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	radioButtons := NewRadioButtons([]string{"Lions", "Elephants", "Giraffes"})
	radioButtons.SetBorder(true)
	radioButtons.SetTitle("Radio Button Demo")
	radioButtons.SetRect(0, 0, 30, 5)

	app.SetRoot(radioButtons, false)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
