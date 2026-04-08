package main

import (
	"fmt"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

// Slider demonstrates the Slider.
func Slider(nextSlide func()) (title string, info string, content cui.Widget) {
	slider := cui.NewSlider()
	slider.SetLabel("Volume:   0%")
	slider.SetChangedFunc(func(value int) {
		slider.SetLabel(fmt.Sprintf("Volume: %3d%%", value))
	})
	slider.SetDoneFunc(func(key tcell.Key) {
		nextSlide()
	})
	return "Slider", sliderInfo, Code(slider, 30, 1, "slider")
}
