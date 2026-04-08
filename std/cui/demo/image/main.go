package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/gif"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

//go:embed starfield.gif
var starfieldGif []byte

// generateModal makes a centered object
func generateModal(p cui.Widget, width, height int) cui.Widget {
	subflex := cui.NewFlex()
	subflex.SetDirection(cui.FlexRow)
	subflex.AddItem(nil, 0, 1, false)
	subflex.AddItem(p, height, 1, true)
	subflex.AddItem(nil, 0, 1, false)
	flex := cui.NewFlex()
	flex.AddItem(nil, 0, 1, false)
	flex.AddItem(subflex, width, 1, true)
	flex.AddItem(nil, 0, 1, false)
	return flex
}

func main() {
	// Create the application.
	a := cui.NewApp()

	// Create our starfield GIF.
	bg := cui.NewImage()

	img, err := gif.DecodeAll(bytes.NewBuffer(starfieldGif))
	if err != nil {
		panic(fmt.Errorf("unable to decode gif: %v", err))
	}

	_, err = bg.SetImage(img)
	if err != nil {
		panic(fmt.Errorf("unable to load gif: %v", err))
	}
	go cui.Animate(a)

	// Create our Hello World text.
	txt := cui.NewTextView()
	txt.SetText("Hello, World")
	txt.SetTextAlign(cui.AlignCenter)
	txt.SetDoneFunc(func(e tcell.Key) {
		a.Stop()
	})
	txt.SetBorder(true)

	// Create a layered page view with a modal
	pages := cui.NewPages()
	pages.AddPage("bg", bg, true, true)
	pages.AddPage("txt", generateModal(txt, 24, 3), true, true)
	a.SetRoot(pages, true)

	// Start the application.
	if err := a.Run(); err != nil {
		panic(err)
	}
}
