package main

import (
	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

// Grid demonstrates the grid layout.
func Grid(nextSlide func()) (title string, info string, content cui.Widget) {
	modalShown := false
	panels := cui.NewPanels()

	newWidget := func(text string) cui.Widget {
		tv := cui.NewTextView()
		tv.SetTextAlign(cui.AlignCenter)
		tv.SetText(text)
		tv.SetDoneFunc(func(key tcell.Key) {
			if modalShown {
				nextSlide()
				modalShown = false
			} else {
				panels.ShowPanel("modal")
				modalShown = true
			}
		})
		return tv
	}

	menu := newWidget("Menu")
	main := newWidget("Main content")
	sideBar := newWidget("Side Bar")

	grid := cui.NewGrid()
	grid.SetRows(3, 0, 3)
	grid.SetColumns(0, -4, 0)
	grid.SetBorders(true)
	grid.AddItem(newWidget("Header"), 0, 0, 1, 3, 0, 0, true)
	grid.AddItem(newWidget("Footer"), 2, 0, 1, 3, 0, 0, false)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(menu, 0, 0, 0, 0, 0, 0, false)
	grid.AddItem(main, 1, 0, 1, 3, 0, 0, false)
	grid.AddItem(sideBar, 0, 0, 0, 0, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false)
	grid.AddItem(main, 1, 1, 1, 1, 0, 100, false)
	grid.AddItem(sideBar, 1, 2, 1, 1, 0, 100, false)

	modal := cui.NewModal()
	modal.SetText("Resize the window to see how the grid layout adapts")
	modal.AddButtons([]string{"Ok"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		panels.HidePanel("modal")
	})

	panels.AddPanel("grid", grid, true, true)
	panels.AddPanel("modal", modal, false, false)

	return "Grid", "", panels
}
