package cui

import (
	"math"
	"sync"

	"github.com/gdamore/tcell/v3"
)

// Sparkline represents a sparkline widgets.
type Sparkline struct {
	*Box

	data           []float64
	dataTitle      string
	dataTitlecolor tcell.Color
	lineColor      tcell.Color
	mu             sync.RWMutex
}

// NewSparkline returns a new sparkline widget.
func NewSparkline() *Sparkline {
	return &Sparkline{
		Box: NewBox(),
	}
}

// Draw draws this widget onto the screen.
func (sl *Sparkline) Draw(screen tcell.Screen) {
	sl.Box.Draw(screen)

	sl.mu.RLock()
	data := append([]float64(nil), sl.data...)
	dataTitle := sl.dataTitle
	dataTitleColor := sl.dataTitlecolor
	lineColor := sl.lineColor
	sl.mu.RUnlock()

	x, y, width, height := sl.GetInnerRect()
	barHeight := height

	// print label
	if dataTitle != "" {
		Print(screen, []byte(dataTitle), x, y, width, AlignLeft, dataTitleColor)

		barHeight--
	}

	maxVal := getMaxFloat64FromSlice(data)
	if maxVal < 0 {
		return
	}

	// print lines
	for i := 0; i < len(data) && i+x < x+width; i++ {
		value := data[i]

		if math.IsNaN(value) {
			continue
		}

		dHeight := int((value / maxVal) * float64(barHeight))

		sparkChar := barsRune[len(barsRune)-1]

		for j := range dHeight {
			PrintJoinedSemigraphics(screen, i+x, y-1+height-j, sparkChar, lineColor)
		}

		if dHeight == 0 {
			sparkChar = barsRune[1]
			PrintJoinedSemigraphics(screen, i+x, y-1+height, sparkChar, lineColor)
		}
	}
}

// SetRect sets rect for this widget.
func (sl *Sparkline) SetRect(x, y, width, height int) {
	sl.Box.SetRect(x, y, width, height)
}

// GetRect return widget current rect.
func (sl *Sparkline) GetRect() (int, int, int, int) {
	return sl.Box.GetRect()
}

// HasFocus returns whether or not this widget has focus.
func (sl *Sparkline) HasFocus() bool {
	return sl.Box.HasFocus()
}

// SetData sets sparkline data.
func (sl *Sparkline) SetData(data []float64) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.data = data
}

// SetDataTitle sets sparkline data title.
func (sl *Sparkline) SetDataTitle(title string) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.dataTitle = title
}

// SetDataTitleColor sets sparkline data title color.
func (sl *Sparkline) SetDataTitleColor(color tcell.Color) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.dataTitlecolor = color
}

// SetLineColor sets sparkline line color.
func (sl *Sparkline) SetLineColor(color tcell.Color) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.lineColor = color
}
