package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
	"github.com/malivvan/rumo/std/cui/edit"
	"github.com/malivvan/rumo/std/cui/edit/runtime"
)

func saveBuffer(b *edit.Buffer, path string) error {
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: femto [filename]\n")
		os.Exit(1)
	}
	path := os.Args[1]

	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read %v: %v", path, err)
	}

	var colorscheme edit.Colorscheme
	if monokai := runtime.Files.FindFile(edit.RTColorscheme, "monokai"); monokai != nil {
		if data, err := monokai.Data(); err == nil {
			colorscheme = edit.ParseColorscheme(string(data))
		}
	}

	app := cui.NewApp()
	buffer := edit.NewBufferFromString(string(content), path)
	root := edit.NewView(buffer)
	root.SetRuntimeFiles(runtime.Files)
	root.SetColorscheme(colorscheme)
	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlS:
			saveBuffer(buffer, path)
			return nil
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		}
		return event
	})
	app.SetRoot(root, true)

	if err := app.Run(); err != nil {
		log.Fatalf("%v", err)
	}
}
