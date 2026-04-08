package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
)

// TextView demonstrates the basic text view.
func TextView(nextSlide func()) (title string, info string, content cui.Widget) {
	textView := cui.NewTextView()
	textView.SetVerticalAlign(cui.AlignBottom)
	textView.SetTextColor(color.Yellow)
	textView.SetDoneFunc(func(key tcell.Key) {
		nextSlide()
	})
	textView.SetChangedFunc(func() {
		if textView.HasFocus() {
			app.Draw()
		}
	})
	go func() {
		var n int
		for {
			n++
			if n > 512 {
				n = 1
				textView.SetText("")
			}

			fmt.Fprintf(textView, "%d ", n)
			time.Sleep(75 * time.Millisecond)
		}
	}()
	textView.SetBorder(true)
	textView.SetTitle("TextView implements io.Writer")
	textView.ScrollToEnd()
	return "TextView", textViewInfo, Code(textView, 36, 13, "textview")
}
