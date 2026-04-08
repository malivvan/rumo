package main

import (
	"time"

	"github.com/malivvan/rumo/std/cui"
)

func main() {
	app := cui.NewApp()
	grid := cui.NewGrid()
	grid.SetBorder(true)
	grid.SetTitle("Spinners")

	spinners := [][]*cui.Spinner{
		{
			cui.NewSpinner().SetStyle(cui.SpinnerDotsCircling),
			cui.NewSpinner().SetStyle(cui.SpinnerDotsUpDown),
			cui.NewSpinner().SetStyle(cui.SpinnerBounce),
			cui.NewSpinner().SetStyle(cui.SpinnerLine),
		},
		{
			cui.NewSpinner().SetStyle(cui.SpinnerCircleQuarters),
			cui.NewSpinner().SetStyle(cui.SpinnerSquareCorners),
			cui.NewSpinner().SetStyle(cui.SpinnerCircleHalves),
			cui.NewSpinner().SetStyle(cui.SpinnerCorners),
		},
		{
			cui.NewSpinner().SetStyle(cui.SpinnerArrows),
			cui.NewSpinner().SetStyle(cui.SpinnerHamburger),
			cui.NewSpinner().SetStyle(cui.SpinnerStack),
			cui.NewSpinner().SetStyle(cui.SpinnerStar),
		},
		{
			cui.NewSpinner().SetStyle(cui.SpinnerGrowHorizontal),
			cui.NewSpinner().SetStyle(cui.SpinnerGrowVertical),
			cui.NewSpinner().SetStyle(cui.SpinnerBoxBounce),
			cui.NewSpinner().SetCustomStyle([]rune{'🕛', '🕐', '🕑', '🕒', '🕓', '🕔', '🕕', '🕖', '🕗', '🕘', '🕙', '🕚'}),
		},
	}

	for rowIdx, row := range spinners {
		for colIdx, spinner := range row {
			grid.AddItem(spinner, rowIdx, colIdx, 1, 1, 1, 1, false)
		}
	}

	update := func() {
		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				for _, row := range spinners {
					for _, spinner := range row {
						spinner.Pulse()
					}
				}
				app.Draw()
			}
		}
	}
	go update()

	app.SetRoot(grid, false)
	app.EnableMouse(true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
