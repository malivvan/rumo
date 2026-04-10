// Demo showing how to run an interactive shell inside a cui terminal widget.
// This is the cui/vte equivalent of the pty/examples/shell example:
// it runs "bash" in a full-screen terminal widget with mouse support.
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

	// Create a terminal widget running an interactive shell.
	cmd := exec.Command("bash")
	term := vte.NewTerminal(app, cmd)
	term.SetBorder(true)
	term.SetTitle(" bash ")

	app.SetRoot(term, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

