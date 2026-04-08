// Demo code for the bar chart widget.
package main

import (
	"math/rand"
	"time"

	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
)

func main() {
	app := cui.NewApp()
	gauge := cui.NewUtilModeGauge()
	gauge.SetLabel("cpu usage:")
	gauge.SetLabelColor(color.LightSkyBlue)
	gauge.SetRect(10, 4, 50, 3)
	gauge.SetWarnPercentage(65)
	gauge.SetCritPercentage(80)
	gauge.SetBorder(true)

	update := func() {
		tick := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				randNum := float64(rand.Float64() * 100)
				gauge.SetValue(randNum)
				app.Draw()
			}
		}
	}
	go update()

	app.SetRoot(gauge, false)
	app.EnableMouse(true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
