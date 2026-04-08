// Demo code for the Grid widget.
package main

import (
	"github.com/malivvan/rumo/std/cui"
)

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	newWidget := func(text string) cui.Widget {
		tv := cui.NewTextView()
		tv.SetTextAlign(cui.AlignCenter)
		tv.SetText(text)
		return tv
	}
	menu := newWidget("Menu")
	main := newWidget("Main content")
	sideBar := newWidget("Side Bar")

	grid := cui.NewGrid()
	grid.SetRows(3, 0, 3)
	grid.SetColumns(30, 0, 30)
	grid.SetBorders(true)
	grid.AddItem(newWidget("Header"), 0, 0, 1, 3, 0, 0, false)
	grid.AddItem(newWidget("Footer"), 2, 0, 1, 3, 0, 0, false)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(menu, 0, 0, 0, 0, 0, 0, false)
	grid.AddItem(main, 1, 0, 1, 3, 0, 0, false)
	grid.AddItem(sideBar, 0, 0, 0, 0, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false)
	grid.AddItem(main, 1, 1, 1, 1, 0, 100, false)
	grid.AddItem(sideBar, 1, 2, 1, 1, 0, 100, false)

	app.SetRoot(grid, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
