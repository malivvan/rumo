// Demo code for the Frame widget.
package main

import (
	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
)

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	box := cui.NewBox()
	box.SetBackgroundColor(color.Blue)

	frame := cui.NewFrame(box)
	frame.SetBorders(2, 2, 2, 2, 4, 4)
	frame.AddText("Header left", true, cui.AlignLeft, color.White)
	frame.AddText("Header middle", true, cui.AlignCenter, color.White)
	frame.AddText("Header right", true, cui.AlignRight, color.White)
	frame.AddText("Header second middle", true, cui.AlignCenter, color.Red)
	frame.AddText("Footer middle", false, cui.AlignCenter, color.Green)
	frame.AddText("Footer second middle", false, cui.AlignCenter, color.Green)

	app.SetRoot(frame, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
