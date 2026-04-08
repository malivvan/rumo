// Demo code for the TabbedPanels widget.
package main

import (
	"fmt"

	"github.com/malivvan/rumo/std/cui"
)

const panelCount = 5

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	panels := cui.NewTabbedPanels()
	for panel := 0; panel < panelCount; panel++ {
		func(panel int) {
			form := cui.NewForm()
			form.SetBorder(true)
			form.SetTitle(fmt.Sprintf("This is tab %d. Choose another tab.", panel+1))
			form.AddButton("Next", func() {
				panels.SetCurrentTab(fmt.Sprintf("panel-%d", (panel+1)%panelCount))
			})
			form.AddButton("Quit", func() {
				app.Stop()
			})
			form.SetCancelFunc(func() {
				app.Stop()
			})

			panels.AddTab(fmt.Sprintf("panel-%d", panel), fmt.Sprintf("Panel #%d", panel), form)
		}(panel)
	}

	app.SetRoot(panels, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
