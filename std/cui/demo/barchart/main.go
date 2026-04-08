// Demo code for the bar chart widget.
package main

import (
	"github.com/gdamore/tcell/v3/color"
	"github.com/malivvan/rumo/std/cui"
)

func main() {
	app := cui.NewApp()
	flex := cui.NewFlex()
	barGraph := cui.NewBarChart()
	barGraph.SetRect(4, 2, 50, 20)

	barGraph.SetBorder(true)
	barGraph.SetTitle("System Resource Usage")
	// display system metric usage
	barGraph.AddBar("cpu", 80, color.Blue)
	barGraph.AddBar("mem", 20, color.Red)
	barGraph.AddBar("swap", 40, color.Green)
	barGraph.AddBar("disk", 40, color.Orange)
	barGraph.SetMaxValue(100)
	barGraph.SetAxesColor(color.AntiqueWhite)
	barGraph.SetAxesLabelColor(color.AntiqueWhite)

	flex.AddItem(barGraph, 0, 1, false)
	app.SetRoot(flex, true)
	app.EnableMouse(true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
