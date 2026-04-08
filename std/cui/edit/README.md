## femto, an editor component for tview

`femto` is a text editor component for tview. The vast majority of the code is derived from
[the micro editor](github.com/zyedidia/micro).

**Note** The shape of the `femto` API is a work-in-progress, and should not be considered stable.

This is a fork of https://github.com/pgavlin/femto

### Default keybindings
```
Up:             CursorUp
Down:           CursorDown
Right:          CursorRight
Left:           CursorLeft
ShiftUp:        SelectUp
ShiftDown:      SelectDown
ShiftLeft:      SelectLeft
ShiftRight:     SelectRight
AltLeft:        WordLeft
AltRight:       WordRight
AltUp:          MoveLinesUp
AltDown:        MoveLinesDown
AltShiftRight:  SelectWordRight
AltShiftLeft:   SelectWordLeft
CtrlLeft:       StartOfLine
CtrlRight:      EndOfLine
CtrlShiftLeft:  SelectToStartOfLine
ShiftHome:      SelectToStartOfLine
CtrlShiftRight: SelectToEndOfLine
ShiftEnd:       SelectToEndOfLine
CtrlUp:         CursorStart
CtrlDown:       CursorEnd
CtrlShiftUp:    SelectToStart
CtrlShiftDown:  SelectToEnd
Alt-{:          ParagraphPrevious
Alt-}:          ParagraphNext
Enter:          InsertNewline
CtrlH:          Backspace
Backspace:      Backspace
Alt-CtrlH:      DeleteWordLeft
Alt-Backspace:  DeleteWordLeft
Tab:            IndentSelection,InsertTab
Backtab:        OutdentSelection,OutdentLine
CtrlZ:          Undo
CtrlY:          Redo
CtrlC:          Copy
CtrlX:          Cut
CtrlK:          CutLine
CtrlD:          DuplicateLine
CtrlV:          Paste
CtrlA:          SelectAll
Home:           StartOfLine
End:            EndOfLine
CtrlHome:       CursorStart
CtrlEnd:        CursorEnd
PageUp:         CursorPageUp
PageDown:       CursorPageDown
CtrlR:          ToggleRuler
Delete:         Delete
Insert:         ToggleOverwriteMode
Alt-f:          WordRight
Alt-b:          WordLeft
Alt-a:          StartOfLine
Alt-e:          EndOfLine
Esc:            Escape
Alt-n:          SpawnMultiCursor
Alt-m:          SpawnMultiCursorSelect
Alt-p:          RemoveMultiCursor
Alt-c:          RemoveAllMultiCursors
Alt-x:          SkipMultiCursor
```

### Example Usage

The code below (also found in `cmd/femto/edit.go`) creates a `tview` application with a single full-screen editor
that operates on one file at a time. Ctrl-s saves any edits; Ctrl-q quits.

```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/mekansm/internal/cui/edit"
	"github.com/malivvan/mekansm/internal/cui/edit/runtime"
	"github.com/malivvan/mekansm/internal/nuview"
)

func saveBuffer(b *edit.Buffer, path string) error {
	return ioutil.WriteFile(path, []byte(b.String()), 0600)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: femto [filename]\n")
		os.Exit(1)
	}
	path := os.Args[1]

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read %v: %v", path, err)
	}

	var colorscheme edit.Colorscheme
	if monokai := runtime.Files.FindFile(edit.RTColorscheme, "monokai"); monokai != nil {
		if data, err := monokai.Data(); err == nil {
			colorscheme = edit.ParseColorscheme(string(data))
		}
	}

	app := nuview.NewApplication()
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
```
