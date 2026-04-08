package main

import (
	"github.com/malivvan/rumo/std/cui"
)

// The width of the code window.
const codeWidth = 56

// Code returns a widget which displays the given widget (with the given
// size) on the left side and its source code on the right side.
func Code(p cui.Widget, width, height int, name string) cui.Widget {
	// Set up code view.
	codeView := cui.NewTextView()
	codeView.SetWrap(true)
	codeView.SetWordWrap(true)
	codeView.SetDynamicColors(false)
	codeView.SetPadding(1, 1, 2, 0)
	codeView.Write(exampleCode(name))

	f := cui.NewFlex()
	f.AddItem(Center(width, height, p), 0, 1, true)
	f.AddItem(codeView, codeWidth, 1, false)
	return f
}
