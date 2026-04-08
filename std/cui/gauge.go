package cui

import (
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v3"
)

// PercentageModeGauge represents percentage mode gauge permitive.
type PercentageModeGauge struct {
	*Box

	// maxValue value
	maxValue int
	// value is current value
	value int
	// pgBgColor: progress block background color
	pgBgColor tcell.Color

	sync.RWMutex
}

// NewPercentageModeGauge returns new percentage mode gauge permitive.
func NewPercentageModeGauge() *PercentageModeGauge {
	gauge := &PercentageModeGauge{
		Box:       NewBox(),
		value:     0,
		pgBgColor: tcell.ColorBlue,
	}

	return gauge
}

// Draw draws this widget onto the screen.
func (g *PercentageModeGauge) Draw(screen tcell.Screen) {
	g.Box.Draw(screen)

	g.RLock()
	maxValue := g.maxValue
	value := g.value
	pgBgColor := g.pgBgColor
	g.RUnlock()

	if maxValue == 0 {
		return
	}

	x, y, width, height := g.GetInnerRect()
	pcWidth := 3
	pc := value * gaugeMaxPc / maxValue
	pcString := fmt.Sprintf("%d%%", pc)
	tW := width - pcWidth
	tX := x + (tW / emptySpaceParts)
	tY := y + height/emptySpaceParts
	prgBlock := getPercentageGaugeProgressBlock(width, value, maxValue)
	backgroundColor := g.GetBackgroundColor()
	style := tcell.StyleDefault.Background(pgBgColor).Foreground(Styles.PrimaryTextColor)

	for i := range height {
		for j := range prgBlock {
			screen.SetContent(x+j, y+i, ' ', nil, style)
		}
	}

	// print percentage in middle of box

	pcRune := []rune(pcString)
	for j := range pcRune {
		style = tcell.StyleDefault.Background(backgroundColor).Foreground(Styles.PrimaryTextColor)
		if x+prgBlock >= tX+j {
			style = tcell.StyleDefault.Background(pgBgColor).Foreground(Styles.PrimaryTextColor)
		}

		for i := range height {
			screen.SetContent(tX+j, y+i, ' ', nil, style)
		}

		screen.SetContent(tX+j, tY, pcRune[j], nil, style)
	}
}

// Focus is called when this widget receives focus.
func (g *PercentageModeGauge) Focus(delegate func(w Widget)) {
}

// HasFocus returns whether or not this widget has focus.
func (g *PercentageModeGauge) HasFocus() bool {
	return g.Box.HasFocus()
}

// GetRect return widget current rect.
func (g *PercentageModeGauge) GetRect() (int, int, int, int) {
	return g.Box.GetRect()
}

// SetRect sets rect for this widget.
func (g *PercentageModeGauge) SetRect(x, y, width, height int) {
	g.Box.SetRect(x, y, width, height)
}

// SetPgBgColor sets progress block background color.
func (g *PercentageModeGauge) SetPgBgColor(color tcell.Color) {
	g.Lock()
	defer g.Unlock()

	g.pgBgColor = color
}

// SetValue update the gauge progress.
func (g *PercentageModeGauge) SetValue(value int) {
	g.Lock()
	defer g.Unlock()

	if value <= g.maxValue {
		g.value = value
	}
}

// GetValue returns current gauge value.
func (g *PercentageModeGauge) GetValue() int {
	g.RLock()
	defer g.RUnlock()

	return g.value
}

// SetMaxValue set maximum allows value for the gauge.
func (g *PercentageModeGauge) SetMaxValue(value int) {
	g.Lock()
	defer g.Unlock()

	if value > 0 {
		g.maxValue = value
	}
}

// GetMaxValue returns maximum allows value for the gauge.
func (g *PercentageModeGauge) GetMaxValue() int {
	g.RLock()
	defer g.RUnlock()

	return g.maxValue
}

// Reset resets the gauge counter (set to 0).
func (g *PercentageModeGauge) Reset() {
	g.Lock()
	defer g.Unlock()

	g.value = 0
}

func getPercentageGaugeProgressBlock(maxValue int, value int, total int) int {
	if total == 0 {
		return 0
	}

	pc := value * gaugeMaxPc / total
	filled := pc * maxValue / gaugeMaxPc

	return filled
}

// UtilisationGauge represents utilisation mode gauge permitive.
type UtilisationGauge struct {
	*Box

	// pc percentage value
	pc float64
	// warn percentage value
	warnPc float64
	// critical percentage value
	critPc float64
	// okColor ok color
	okColor tcell.Color
	// warnColor warning block color
	warnColor tcell.Color
	// critColor critical block color
	critColor tcell.Color
	// emptyColor empty block color
	emptyColor tcell.Color
	// label prints label on the left of the gauge
	label string
	// labelColor label and percentage text color
	labelColor tcell.Color

	sync.RWMutex
}

// NewUtilModeGauge returns new utilisation mode gauge permitive.
func NewUtilModeGauge() *UtilisationGauge {
	gauge := &UtilisationGauge{
		Box:        NewBox(),
		pc:         gaugeMinPc,
		warnPc:     gaugeWarnPc,
		critPc:     gaugeCritPc,
		warnColor:  tcell.ColorOrange,
		critColor:  tcell.ColorRed,
		okColor:    tcell.ColorGreen,
		emptyColor: tcell.ColorWhite,
		labelColor: Styles.PrimaryTextColor,
		label:      "",
	}

	return gauge
}

// SetLabel sets label for this widget.
func (g *UtilisationGauge) SetLabel(label string) {
	g.Lock()
	defer g.Unlock()

	g.label = label
}

// SetLabelColor sets label text color.
func (g *UtilisationGauge) SetLabelColor(color tcell.Color) {
	g.Lock()
	defer g.Unlock()

	g.labelColor = color
}

// Focus is called when this widget receives focus.
func (g *UtilisationGauge) Focus(delegate func(w Widget)) {
}

// HasFocus returns whether or not this widget has focus.
func (g *UtilisationGauge) HasFocus() bool {
	return g.Box.HasFocus()
}

// GetRect return widget current rect.
func (g *UtilisationGauge) GetRect() (int, int, int, int) {
	return g.Box.GetRect()
}

// SetRect sets rect for this widget.
func (g *UtilisationGauge) SetRect(x, y, width, height int) {
	g.Box.SetRect(x, y, width, height)
}

// SetValue update the gauge progress.
func (g *UtilisationGauge) SetValue(value float64) {
	g.Lock()
	defer g.Unlock()

	if value <= float64(gaugeMaxPc) {
		g.pc = value
	}
}

// GetValue returns current gauge value.
func (g *UtilisationGauge) GetValue() float64 {
	g.RLock()
	defer g.RUnlock()

	return g.pc
}

// Draw draws this widget onto the screen.
func (g *UtilisationGauge) Draw(screen tcell.Screen) {
	g.Box.Draw(screen)

	g.RLock()
	pc := g.pc
	warnPc := g.warnPc
	critPc := g.critPc
	okColor := g.okColor
	warnColor := g.warnColor
	critColor := g.critColor
	emptyColor := g.emptyColor
	label := g.label
	labelColor := g.labelColor
	g.RUnlock()

	x, y, width, height := g.GetInnerRect()
	labelPCWidth := 7
	labelWidth := len(label)
	barWidth := width - labelPCWidth - labelWidth

	for i := range barWidth {
		for j := range height {
			value := float64(i * 100 / barWidth)
			color := getUtilisationGaugeBarColor(value, warnPc, critPc, okColor, warnColor, critColor)

			if value > pc {
				color = emptyColor
			}

			Print(screen, []byte(prgCell), x+labelWidth+i, y+j, 1, AlignCenter, color)
		}
	}
	// draw label
	tY := y + (height / emptySpaceParts)
	if labelWidth > 0 {
		Print(screen, []byte(label), x, tY, labelWidth, AlignLeft, labelColor)
	}

	// draw percentage text
	Print(screen, []byte(fmt.Sprintf("%6.2f%%", pc)),
		x+barWidth+labelWidth,
		tY,
		labelPCWidth,
		AlignLeft,
		Styles.PrimaryTextColor)
}

// SetWarnPercentage sets warning percentage start range.
func (g *UtilisationGauge) SetWarnPercentage(percentage float64) {
	g.Lock()
	defer g.Unlock()

	if percentage > 0 && percentage < 100 {
		g.warnPc = percentage
	}
}

// SetCritPercentage sets critical percentage start range.
func (g *UtilisationGauge) SetCritPercentage(percentage float64) {
	g.Lock()
	defer g.Unlock()

	if percentage > 0 && percentage < 100 && percentage > g.warnPc {
		g.critPc = percentage
	}
}

// SetEmptyColor sets empty gauge color.
func (g *UtilisationGauge) SetEmptyColor(color tcell.Color) {
	g.Lock()
	defer g.Unlock()

	g.emptyColor = color
}

func getUtilisationGaugeBarColor(percentage, warnPc, critPc float64, okColor, warnColor, critColor tcell.Color) tcell.Color {
	if percentage < warnPc {
		return okColor
	} else if percentage < critPc {
		return warnColor
	}

	return critColor
}
