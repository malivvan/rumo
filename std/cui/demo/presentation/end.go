package main

import (
	"fmt"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

// End shows the final slide.
func End(nextSlide func()) (title string, info string, content cui.Widget) {
	textView := cui.NewTextView()
	textView.SetDoneFunc(func(key tcell.Key) {
		nextSlide()
	})
	url := "https://github.com/malivvan/rumo/std/cui"
	fmt.Fprint(textView, url)
	return "End", "", Center(len(url), 1, textView)
}
