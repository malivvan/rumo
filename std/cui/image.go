package cui

import (
	"errors"
	"fmt"
	"image"
	gcolor "image/color"
	"image/gif"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

// Image is a box which displays animated gifs via Omnikron13's pixelview
// dynamic color rendering.  It automatically draws the right frame based on
// time elapsed since creation.  You can trigger re-drawing by executing
// Animate(App) in a goroutine.
type Image struct {
	sync.RWMutex
	*Box

	// Timing for the frames
	delay         []time.Duration
	frames        []string
	startTime     time.Time
	totalDuration time.Duration
}

// NewImage returns a new Image.
func NewImage() *Image {
	return &Image{
		Box:       NewBox(),
		startTime: time.Now(),
	}
}

// SetImagePath sets the image to a given GIF path
func (g *Image) SetImagePath(imagePath string) (*Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return g, fmt.Errorf("unable to open file: %v", err)
	}
	defer file.Close()

	image, err := gif.DecodeAll(file)
	if err != nil {
		return g, fmt.Errorf("unable to decode GIF: %v", err)
	}

	return g.SetImage(image)
}

// SetImage sets the content to a given gif.GIF
func (g *Image) SetImage(image *gif.GIF) (*Image, error) {
	g.Lock()
	defer g.Unlock()

	g.delay = nil
	g.frames = nil
	g.startTime = time.Now()

	// Store delay in milliseconds
	g.totalDuration = time.Duration(0)
	for _, i := range image.Delay {
		d := time.Duration(i*10) * time.Millisecond
		g.delay = append(g.delay, d)
		g.totalDuration += d
	}

	// Set height,width of the box
	g.SetRect(0, 0, image.Config.Width, image.Config.Height)

	// Convert images to text
	var frames []string
	for i, img := range image.Image {
		parsed, err := imageFromImage(img)
		if err != nil {
			return g, fmt.Errorf("unable to convert frame %d: %v", i, err)
		}
		frames = append(frames, parsed)
	}

	// Store the output
	g.frames = frames

	return g, nil
}

// GetCurrentFrame returns the current frame the GIF is on
func (g *Image) GetCurrentFrame() int {
	g.RLock()
	startTime := g.startTime
	totalDuration := g.totalDuration
	delay := append([]time.Duration(nil), g.delay...)
	g.RUnlock()

	// Always at frame 0
	if totalDuration == 0 {
		return 0
	}

	dur := time.Since(startTime) % totalDuration
	for i, d := range delay {
		dur -= d
		if dur < 0 {
			return i
		}
	}
	return 0
}

// Draw renders the current frame of the GIF
func (g *Image) Draw(screen tcell.Screen) {
	g.Box.Draw(screen)

	g.RLock()
	frames := append([]string(nil), g.frames...)
	g.RUnlock()
	if len(frames) == 0 {
		return
	}

	currentFrame := g.GetCurrentFrame()
	if currentFrame >= len(frames) {
		currentFrame = 0
	}

	frame := strings.Split(frames[currentFrame], "\n")
	x, y, w, _ := g.GetInnerRect()

	for i, line := range frame {
		Print(screen, []byte(line), x, y+i, w, AlignLeft, color.White)
	}
}

var globalAnimationMutex = &sync.Mutex{}

// Animate triggers the application to redraw every 50ms
func Animate(app *App) {
	globalAnimationMutex.Lock()
	defer globalAnimationMutex.Unlock()

	for {
		app.QueueUpdateDraw(func() {})
		time.Sleep(50 * time.Millisecond)
	}
}

// imageFromFile is a convenience function that converts a file on disk to a formatted string.
// See FromImage() for more details.
func imageFromFile(filename string) (encoded string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	return imageFromReader(io.Reader(f))
}

// imageFromReader is a convenience function that converts an io.Reader to a formatted string.
// See FromImage() for more details.
func imageFromReader(reader io.Reader) (encoded string, err error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return
	}
	return imageFromImage(img)
}

// imageFromImage is the primary function of this package,
// It takes an image.Image and converts it to a string formatted for tview.
// The unicode half-block character (▀) with a fg & bg colour set will represent
// pixels in the returned string.
// Because each character represents two pixels, it is not possible to convert an
// image if its height is uneven. Attempts to do so will return an error.
func imageFromImage(img image.Image) (encoded string, err error) {
	if (img.Bounds().Max.Y-img.Bounds().Min.Y)%2 != 0 {
		err = errors.New("pixelview: Can't process image with uneven height")
		return
	}

	switch v := img.(type) {
	default:
		return imageFromGeneric(img)
	case *image.Paletted:
		return imageFromPaletted(v)
	case *image.NRGBA:
		return imageFromNRGBA(v)
	}
}

// imageFromGeneric is the fallback function for processing images.
// It will be used for more exotic image formats than png or gif.
func imageFromGeneric(img image.Image) (encoded string, err error) {
	var sb strings.Builder
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 2 {
		var prevfg, prevbg gcolor.Color
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			fg := img.At(x, y)
			bg := img.At(x, y+1)
			encodeImage(fg, bg, &prevfg, &prevbg, &sb)
		}
		sb.WriteRune('\n')
	}
	encoded = sb.String()
	return
}

// imageFromPaletted saves a few μs when working with paletted images.
// These are what PNG8 images are decoded as.
func imageFromPaletted(img *image.Paletted) (encoded string, err error) {
	var sb strings.Builder
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y += 2 {
		var prevfg, prevbg gcolor.Color
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			i := (y-img.Rect.Min.Y)*img.Stride + (x - img.Rect.Min.X)
			fg := img.Palette[img.Pix[i]]
			bg := img.Palette[img.Pix[i+img.Stride]]
			encodeImage(fg, bg, &prevfg, &prevbg, &sb)
		}
		sb.WriteRune('\n')
	}
	encoded = sb.String()
	return
}

// imageFromNRGBA saves a handful of μs when working with NRGBA images.
// These are what PNG24 images are decoded as.
func imageFromNRGBA(img *image.NRGBA) (encoded string, err error) {
	var sb strings.Builder
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y += 2 {
		var prevfg, prevbg gcolor.Color
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			i := (y-img.Rect.Min.Y)*img.Stride + (x-img.Rect.Min.X)*4
			fg := gcolor.NRGBA{R: img.Pix[i], G: img.Pix[i+1], B: img.Pix[i+2], A: img.Pix[i+3]}
			i += img.Stride
			bg := gcolor.NRGBA{R: img.Pix[i], G: img.Pix[i+1], B: img.Pix[i+2], A: img.Pix[i+3]}
			encodeImage(fg, bg, &prevfg, &prevbg, &sb)
		}
		sb.WriteRune('\n')
	}
	encoded = sb.String()
	return
}

// encode converts a fg & bg colour into a formatted pair of 'pixels',
// using the prevfg & prevbg colours to perform something akin to run-length encoding
func encodeImage(fg, bg gcolor.Color, prevfg, prevbg *gcolor.Color, sb *strings.Builder) {
	if fg == *prevfg && bg == *prevbg {
		sb.WriteRune('▀')
		return
	}
	if fg == *prevfg {
		sb.WriteString(fmt.Sprintf(
			"[:%s]▀",
			imageHexColour(bg),
		))
		*prevbg = bg
		return
	}
	if bg == *prevbg {
		sb.WriteString(fmt.Sprintf(
			"[%s:]▀",
			imageHexColour(fg),
		))
		*prevfg = fg
		return
	}
	sb.WriteString(fmt.Sprintf(
		"[%s:%s]▀",
		imageHexColour(fg),
		imageHexColour(bg),
	))
	*prevfg = fg
	*prevbg = bg
	return
}

func imageHexColour(c gcolor.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%.2x%.2x%.2x", r>>8, g>>8, b>>8)
}
