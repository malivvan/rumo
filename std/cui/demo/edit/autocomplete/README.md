# Edit autocomplete + inline suggestion demo

This demo showcases autocomplete and inline ghost suggestions in `github.com/malivvan/rumo/std/cui/edit`.

## Features
- custom completion provider with `Detail` text
- inline ghost text rendered after the cursor from `CompletionItem.InlineText`
- examples where the ghost preview differs from the accepted `InsertText`
- fallback suggestions from `edit.DefaultAutocompleteProvider`
- explicit popup trigger with `Ctrl+Space`
- startup suggestion so the feature is visible immediately
- live refresh while typing
- keyboard and mouse selection

## Run

```bash
go run ./demo/edit/autocomplete
```

## Keys
- `Ctrl+Space`: open autocomplete at the cursor
- `Up` / `Down` / `PgUp` / `PgDn`: move through suggestions
- `Enter` or `Tab`: accept
- `Esc`: close popup
- `Ctrl-Q`: quit

## What to try
- Start on the `fmt.Pr` line and observe the ghost text after the cursor.
- Type `i` to narrow `Println` / `Printf` and watch the inline suggestion update.
- Compare the ghost preview with what is actually inserted when you press `Enter`.

