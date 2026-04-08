// Demo code for the DropDown widget.
package main

import "github.com/malivvan/rumo/std/cui"

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	dropdown := cui.NewDropDown()
	dropdown.SetLabel("Select an option (hit Enter): ")
	dropdown.SetOptions(nil,
		cui.NewDropDownOption("First"),
		cui.NewDropDownOption("Second"),
		cui.NewDropDownOption("Third"),
		cui.NewDropDownOption("Fourth"),
		cui.NewDropDownOption("Fifth"))

	app.SetRoot(dropdown, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
