// Demo code for the bar chart widget.
package main

import (
	"github.com/malivvan/rumo/std/cui"
)

func main() {
	app := cui.NewApp()
	dialog := cui.NewMessageDialog(cui.ErrorDailog)
	dialog.SetTitle("error dialog")
	dialog.SetMessage("This is first line of error\nThis is second line of the error message")
	dialog.SetDoneFunc(func() {
		app.Stop()
	})
	app.SetRoot(dialog, true)
	app.EnableMouse(true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
