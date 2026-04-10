// Demo showing how to run a one-shot command inside a cui terminal widget.
// This is the cui/vte equivalent of the pty/examples/command example:
// it runs "ls --color=auto -la /" in a bordered terminal widget.
package main

import (
	"os/exec"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
	"github.com/malivvan/rumo/std/cui/vte"
)

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	// Create a terminal widget that runs a single command.
	cmd := exec.Command("ls", "--color=auto", "-la", "/")
	term := vte.NewTerminal(app, cmd)
	term.SetBorder(true)
	term.SetTitle(" ls --color=auto -la / ")

	// Quit the application on Escape or q.
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Str() == "q" {
				app.Stop()
				return nil
			}
		}
		return event
	})

	app.SetRoot(term, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
