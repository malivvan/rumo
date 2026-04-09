package cui

import (
	"math"
	"sync"

	"github.com/gdamore/tcell/v3"
)

// ProgressBar indicates the progress of an operation.
type ProgressBar struct {
	*Box

	// Rune to use when rendering the empty area of the progress bar.
	emptyRune rune

	// Color of the empty area of the progress bar.
	emptyColor tcell.Color

	// Rune to use when rendering the filled area of the progress bar.
	filledRune rune

	// Color of the filled area of the progress bar.
	filledColor tcell.Color

	// If set to true, instead of filling from left to right, the bar is filled
	// from bottom to top.
	vertical bool

	// Current progress.
	progress int

	// Progress required to fill the bar.
	max int

	sync.RWMutex
}

// NewProgressBar returns a new progress bar.
func NewProgressBar() *ProgressBar {
	p := &ProgressBar{
		Box:         NewBox(),
		emptyRune:   tcell.RuneBlock,
		emptyColor:  Styles.WidgetBackgroundColor,
		filledRune:  tcell.RuneBlock,
		filledColor: Styles.PrimaryTextColor,
		max:         100,
	}
	p.SetBackgroundColor(Styles.WidgetBackgroundColor)
	return p
}

// SetEmptyRune sets the rune used for the empty area of the progress bar.
func (p *ProgressBar) SetEmptyRune(empty rune) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.emptyRune = empty
	return p
}

// SetEmptyColor sets the color of the empty area of the progress bar.
func (p *ProgressBar) SetEmptyColor(empty tcell.Color) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.emptyColor = empty
	return p
}

// SetFilledRune sets the rune used for the filled area of the progress bar.
func (p *ProgressBar) SetFilledRune(filled rune) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.filledRune = filled
	return p
}

// SetFilledColor sets the color of the filled area of the progress bar.
func (p *ProgressBar) SetFilledColor(filled tcell.Color) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.filledColor = filled
	return p
}

// SetVertical sets the direction of the progress bar.
func (p *ProgressBar) SetVertical(vertical bool) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.vertical = vertical
	return p
}

// SetMax sets the progress required to fill the bar.
func (p *ProgressBar) SetMax(max int) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.max = max
	return p
}

// GetMax returns the progress required to fill the bar.
func (p *ProgressBar) GetMax() int {
	p.RLock()
	defer p.RUnlock()

	return p.max
}

// AddProgress adds to the current progress.
func (p *ProgressBar) AddProgress(progress int) {
	p.Lock()
	defer p.Unlock()

	p.progress += progress
	if p.progress < 0 {
		p.progress = 0
	} else if p.progress > p.max {
		p.progress = p.max
	}
}

// SetProgress sets the current progress.
func (p *ProgressBar) SetProgress(progress int) *ProgressBar {
	p.Lock()
	defer p.Unlock()

	p.progress = progress
	if p.progress < 0 {
		p.progress = 0
	} else if p.progress > p.max {
		p.progress = p.max
	}
	return p
}

// GetProgress gets the current progress.
func (p *ProgressBar) GetProgress() int {
	p.RLock()
	defer p.RUnlock()

	return p.progress
}

// Complete returns whether the progress bar has been filled.
func (p *ProgressBar) Complete() bool {
	p.RLock()
	defer p.RUnlock()

	return p.progress >= p.max
}

// Draw draws this widget onto the screen.
func (p *ProgressBar) Draw(screen tcell.Screen) {
	if !p.GetVisible() {
		return
	}

	p.Box.Draw(screen)
	backgroundColor := p.GetBackgroundColor()

	p.Lock()
	defer p.Unlock()

	x, y, width, height := p.GetInnerRect()

	barSize := height
	maxLength := width
	if p.vertical {
		barSize = width
		maxLength = height
	}

	barLength := int(math.RoundToEven(float64(maxLength) * (float64(p.progress) / float64(p.max))))
	if barLength > maxLength {
		barLength = maxLength
	}

	for i := 0; i < barSize; i++ {
		for j := 0; j < barLength; j++ {
			if p.vertical {
				screen.Put(x+i, y+(height-1-j), string(p.filledRune), tcell.StyleDefault.Foreground(p.filledColor).Background(backgroundColor))
			} else {
				screen.Put(x+j, y+i, string(p.filledRune), tcell.StyleDefault.Foreground(p.filledColor).Background(backgroundColor))
			}
		}
		for j := barLength; j < maxLength; j++ {
			if p.vertical {
				screen.Put(x+i, y+(height-1-j), string(p.emptyRune), tcell.StyleDefault.Foreground(p.emptyColor).Background(backgroundColor))
			} else {
				screen.Put(x+j, y+i, string(p.emptyRune), tcell.StyleDefault.Foreground(p.emptyColor).Background(backgroundColor))
			}
		}
	}
}
