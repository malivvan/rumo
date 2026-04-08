// Demo code for the Box widget.
package main

import (
	"os/exec"

	"github.com/malivvan/rumo/std/cui"
	"github.com/malivvan/rumo/std/cui/vte"
)

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)
	app.EnableBracketedPaste(true)
	t1 := vte.NewTerminal(app, exec.Command("/bin/bash"))
	t1.SetBorder(true)
	t1.SetTitle("vte demo")
	t2 := vte.NewTerminal(app, exec.Command("/bin/bash"))
	t2.SetBorder(true)
	t2.SetTitle("vte demo 2")
	root := cui.NewFlex()
	root.SetDirection(cui.FlexRow)
	root.AddItem(t1, 0, 1, true)
	root.AddItem(t2, 0, 1, true)
	app.SetRoot(root, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
