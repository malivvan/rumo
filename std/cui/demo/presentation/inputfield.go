package main

import (
	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

// InputField demonstrates the InputField.
func InputField(nextSlide func()) (title string, info string, content cui.Widget) {
	input := cui.NewInputField()
	input.SetLabel("Enter a number: ")
	input.SetAcceptanceFunc(cui.InputFieldInteger)
	input.SetDoneFunc(func(key tcell.Key) {
		nextSlide()
	})
	return "InputField", "", Code(input, 30, 1, "inputfield")
}
