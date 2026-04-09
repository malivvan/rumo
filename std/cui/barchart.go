package cui

import (
	"strconv"
	"sync"

	"github.com/gdamore/tcell/v3"
)

const (
	barChartYAxisLabelWidth = 2
	barGap                  = 2
	barWidth                = 3
)

// BarChartItem represents a single bar in bar chart.
type BarChartItem struct {
	label string
	value int
	color tcell.Color
}

// BarChart represents bar chart widget.
type BarChart struct {
	*Box

	// bar items
	bars []BarChartItem
	// maximum value of bars
	maxVal int
	// barGap gap between two bars
	barGap int
	// barWidth width of bars
	barWidth int
	// hasBorder true if widget has border
	hasBorder      bool
	axesColor      tcell.Color
	axesLabelColor tcell.Color

	sync.RWMutex
}

// NewBarChart returns a new bar chart widget.
func NewBarChart() *BarChart {
	chart := &BarChart{
		Box:            NewBox(),
		barGap:         barGap,
		barWidth:       barWidth,
		axesColor:      tcell.ColorDimGray,
		axesLabelColor: tcell.ColorDimGray,
	}

	return chart
}

// Focus is called when this widget receives focus.
func (c *BarChart) Focus(delegate func(w Widget)) {
	delegate(c.Box)
}

// HasFocus returns whether or not this widget has focus.
func (c *BarChart) HasFocus() bool {
	return c.Box.HasFocus()
}

// Draw draws this widget onto the screen.
func (c *BarChart) Draw(screen tcell.Screen) { //nolint:funlen,cyclop
	c.Box.Draw(screen)

	c.RLock()
	bars := append([]BarChartItem(nil), c.bars...)
	maxVal := c.maxVal
	barGap := c.barGap
	barWidth := c.barWidth
	hasBorder := c.hasBorder
	axesColor := c.axesColor
	axesLabelColor := c.axesLabelColor
	c.RUnlock()

	x, y, width, height := c.GetInnerRect()

	maxValY := y + 1
	xAxisStartY := y + height - 2 //nolint:mnd
	barStartY := y + height - 3   //nolint:mnd
	borderPadding := 0

	if hasBorder {
		borderPadding = 1
	}
	if maxVal == 0 {
		for _, b := range bars {
			if b.value > maxVal {
				maxVal = b.value
			}
		}
	}
	if maxVal == 0 {
		return
	}
	maxValueSr := strconv.Itoa(maxVal)
	maxValLenght := len(maxValueSr) + 1

	if maxValLenght < barChartYAxisLabelWidth {
		maxValLenght = barChartYAxisLabelWidth
	}

	// draw Y axis line
	drawLine(screen,
		x+maxValLenght,
		y+borderPadding,
		height-borderPadding-1,
		verticalLine, axesColor)

	// draw X axis line
	drawLine(screen,
		x+maxValLenght+1,
		xAxisStartY,
		width-borderPadding-maxValLenght-1,
		horizontalLine, axesColor)

	PrintJoinedSemigraphics(screen,
		x+maxValLenght,
		xAxisStartY,
		BoxDrawingsLightUpAndRight, axesColor)

	PrintJoinedSemigraphics(screen, x+maxValLenght-1, xAxisStartY, '0', axesLabelColor)

	mxValRune := []rune(maxValueSr)
	for i := range mxValRune {
		PrintJoinedSemigraphics(screen, x+borderPadding+i, maxValY, mxValRune[i], axesLabelColor)
	}

	// draw bars
	startX := x + maxValLenght + barGap
	labelY := y + height - 1
	valueMaxHeight := barStartY - maxValY

	for _, item := range bars {
		if startX > x+width {
			return
		}
		// set labels
		r := []rune(item.label)
		for j := range r {
			PrintJoinedSemigraphics(screen, startX+j, labelY, r[j], axesLabelColor)
		}
		// bar style
		barHeight := getBarChartHeight(valueMaxHeight, item.value, maxVal)

		for k := range barHeight {
			for l := range barWidth {
				PrintJoinedSemigraphics(screen, startX+l, barStartY-k, fullBlockRune, item.color)
			}
		}
		// bar value
		vSt := strconv.Itoa(item.value)
		vRune := []rune(vSt)

		for i := range vRune {
			PrintJoinedSemigraphics(screen, startX+i, barStartY-barHeight, vRune[i], item.color)
		}

		// calculate next startX for next bar
		rWidth := len(r)
		if rWidth < barWidth {
			rWidth = barWidth
		}

		startX = startX + barGap + rWidth
	}
}

// SetBorder sets border for this widget.
func (c *BarChart) SetBorder(status bool) *BarChart {
	c.Lock()
	defer c.Unlock()

	c.hasBorder = status
	c.Box.SetBorder(status)
	return c
}

// GetRect return widget current rect.
func (c *BarChart) GetRect() (int, int, int, int) {
	return c.Box.GetRect()
}

// SetRect sets rect for this widget.
func (c *BarChart) SetRect(x, y, width, height int) Widget {
	c.Box.SetRect(x, y, width, height)
	return c
}

// SetMaxValue sets maximum value of bars.
func (c *BarChart) SetMaxValue(maxValue int) *BarChart {
	c.Lock()
	defer c.Unlock()

	c.maxVal = maxValue
	return c
}

// SetAxesColor sets axes x and y lines color.
func (c *BarChart) SetAxesColor(color tcell.Color) *BarChart {
	c.Lock()
	defer c.Unlock()

	c.axesColor = color
	return c
}

// SetAxesLabelColor sets axes x and y label color.
func (c *BarChart) SetAxesLabelColor(color tcell.Color) *BarChart {
	c.Lock()
	defer c.Unlock()

	c.axesLabelColor = color
	return c
}

// AddBar adds new bar item to the bar chart widget.
func (c *BarChart) AddBar(label string, value int, color tcell.Color) {
	c.Lock()
	defer c.Unlock()

	c.bars = append(c.bars, BarChartItem{
		label: label,
		value: value,
		color: color,
	})
}

// RemoveBar removes a bar item from the bar chart.
func (c *BarChart) RemoveBar(label string) {
	c.Lock()
	defer c.Unlock()

	bars := c.bars[:0]

	for _, barItem := range c.bars {
		if barItem.label != label {
			bars = append(bars, barItem)
		}
	}

	c.bars = bars
}

// SetBarValue sets bar values.
func (c *BarChart) SetBarValue(name string, value int) *BarChart {
	c.Lock()
	defer c.Unlock()

	for i := range c.bars {
		if c.bars[i].label == name {
			c.bars[i].value = value
		}
	}
	return c
}

func getBarChartHeight(maxHeight int, value int, maxValue int) int {
	if maxValue <= 0 {
		return 0
	}
	if value >= maxValue {
		return maxHeight
	}

	height := (value * maxHeight) / maxValue

	return height
}

