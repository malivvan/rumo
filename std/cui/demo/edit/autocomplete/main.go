// Demo code for edit.View autocomplete.
package main

import (
	"log"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
	"github.com/malivvan/rumo/std/cui/edit"
	"github.com/malivvan/rumo/std/cui/edit/runtime"
)

type demoCompletion struct {
	Label        string
	InsertText   string
	InlineTarget string
	Detail       string
}

var demoCompletions = []demoCompletion{
	{Label: "Println", InsertText: "Println", InlineTarget: "Println(\"inline suggestion\")", Detail: "fmt function · ghost call preview"},
	{Label: "Printf", InsertText: "Printf", InlineTarget: "Printf(\"%s\\n\", name)", Detail: "fmt function · format hint"},
	{Label: "Sprint", InsertText: "Sprint", InlineTarget: "Sprint(name)", Detail: "fmt function"},
	{Label: "Sprintf", InsertText: "Sprintf", InlineTarget: "Sprintf(\"hello %s\", name)", Detail: "fmt function · ghost format preview"},
	{Label: "message", InsertText: "message", InlineTarget: "message", Detail: "local variable"},
	{Label: "name", InsertText: "name", InlineTarget: "name", Detail: "function parameter"},
	{Label: "greeting", InsertText: "greeting", InlineTarget: "greeting", Detail: "local variable"},
	{Label: "greet", InsertText: "greet", InlineTarget: "greet(name)", Detail: "function · call preview"},
	{Label: "func", InsertText: "func", InlineTarget: "func main() {", Detail: "Go keyword · snippet-like preview"},
	{Label: "for", InsertText: "for", InlineTarget: "for i := range items {", Detail: "Go keyword · loop preview"},
	{Label: "if", InsertText: "if", InlineTarget: "if err != nil {", Detail: "Go keyword · condition preview"},
	{Label: "range", InsertText: "range", InlineTarget: "range items", Detail: "Go keyword"},
}

const demoText = `// edit autocomplete + inline suggestion demo
//
// Try this:
//   - The popup opens on startup and shows ghost text after the cursor.
//   - Type to refresh suggestions automatically.
//   - Press Ctrl-Space to open autocomplete explicitly at the cursor.
//   - Notice that some ghost text previews differ from the accepted InsertText.
//   - Use Up/Down, PgUp/PgDn, Enter, Tab, Esc, or the mouse.
//   - Press Ctrl-Q to quit.
//
// Suggested examples:
//   fmt.Pr
//   fmt.Pri
//   fmt.Sp
//   mes
//   gre

package main

import "fmt"

func greet(name string) string {
	greeting := fmt.Sp
	return greeting + name
}

func main() {
	mes := greet("cui")
	fmt.Pr
	_ = mes
}
`

func loadColorscheme() edit.Colorscheme {
	var scheme edit.Colorscheme
	if monokai := runtime.Files.FindFile(edit.RTColorscheme, "monokai"); monokai != nil {
		if data, err := monokai.Data(); err == nil {
			scheme = edit.ParseColorscheme(string(data))
		}
	}

	if scheme == nil {
		scheme = edit.Colorscheme{
			"default": tcell.StyleDefault,
		}
	}
	if _, ok := scheme["suggestion"]; !ok {
		base := scheme.GetDefault()
		scheme["suggestion"] = base.Foreground(color.Gray).Dim(true)
	}
	return scheme
}

func demoProvider(ctx edit.CompletionContext) []edit.CompletionItem {
	if ctx.Prefix == "" {
		return nil
	}

	prefix := strings.ToLower(ctx.Prefix)
	items := make([]edit.CompletionItem, 0, len(demoCompletions))
	seen := make(map[string]struct{})
	inlineSuffix := func(full string) string {
		if full == "" {
			return ""
		}
		fullRunes := []rune(full)
		prefixRunes := []rune(ctx.Prefix)
		if len(prefixRunes) >= len(fullRunes) {
			return ""
		}
		if strings.HasPrefix(strings.ToLower(full), prefix) {
			return string(fullRunes[len(prefixRunes):])
		}
		if strings.HasPrefix(strings.ToLower(full), strings.ToLower(ctx.Word)) {
			wordRunes := []rune(ctx.Word)
			if len(wordRunes) < len(fullRunes) {
				return string(fullRunes[len(wordRunes):])
			}
		}
		return ""
	}
	appendItem := func(item edit.CompletionItem) {
		key := item.Label
		if key == "" {
			key = item.InsertText
		}
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}

	for _, item := range demoCompletions {
		candidate := item.Label
		if candidate == "" {
			candidate = item.InsertText
		}
		if strings.HasPrefix(strings.ToLower(candidate), prefix) {
			appendItem(edit.CompletionItem{
				Label:      item.Label,
				InsertText: item.InsertText,
				InlineText: inlineSuffix(item.InlineTarget),
				Detail:     item.Detail,
			})
		}
	}

	for _, item := range edit.DefaultAutocompleteProvider(ctx) {
		candidate := item.Label
		if candidate == "" {
			candidate = item.InsertText
		}
		if strings.HasPrefix(strings.ToLower(candidate), prefix) {
			if item.Detail == "" {
				item.Detail = "from buffer"
			}
			appendItem(item)
		}
	}

	return items
}

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()

	buffer := edit.NewBufferFromString(demoText, "autocomplete-demo.go")
	view := edit.NewView(buffer)
	view.SetBorder(true)
	view.SetTitle("edit autocomplete + inline suggestion demo")
	view.SetRuntimeFiles(runtime.Files)
	view.SetColorscheme(loadColorscheme())
	view.SetAutocompleteProvider(demoProvider)
	view.Cursor.GotoLoc(edit.Loc{X: 8, Y: 22})
	view.TriggerAutocomplete()
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyNUL:
			view.TriggerAutocomplete()
			return nil
		case event.Key() == tcell.KeyRune && event.Modifiers() == tcell.ModCtrl && event.Str() == " ":
			view.TriggerAutocomplete()
			return nil
		case event.Key() == tcell.KeyCtrlQ:
			app.Stop()
			return nil
		}
		return event
	})

	app.EnableMouse(true)
	app.SetRoot(view, true)
	if err := app.Run(); err != nil {
		log.Fatalf("%v", err)
	}
}
