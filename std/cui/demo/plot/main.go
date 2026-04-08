package main

import (
	"math"

	"github.com/gdamore/tcell/v3"
	"github.com/malivvan/rumo/std/cui"
)

func main() {

	app := cui.NewApp()

	// plot (line charts)
	sinData := func() [][]float64 {
		n := 220
		data := make([][]float64, 2)
		data[0] = make([]float64, n)
		data[1] = make([]float64, n)
		for i := 0; i < n; i++ {
			data[0][i] = 1 + math.Sin(float64(i)/5)
			data[1][i] = 1 + math.Cos(float64(i)/5)
		}
		return data
	}()

	bmLineChart := newBrailleModeLineChart()
	app.SetAfterResizeFunc(func(width int, height int) {
		bmLineChart.SetData(sinData)
		app.Draw()
	})
	app.SetRoot(bmLineChart, true)
	app.EnableMouse(true)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func newBrailleModeLineChart() *cui.Plot {
	bmLineChart := cui.NewPlot()
	bmLineChart.SetBorder(true)
	bmLineChart.SetTitle("line chart (braille mode)")
	bmLineChart.SetLineColor([]tcell.Color{
		tcell.ColorSteelBlue,
		tcell.ColorGreen,
	})
	bmLineChart.SetMarker(cui.PlotMarkerBraille)

	return bmLineChart
}
