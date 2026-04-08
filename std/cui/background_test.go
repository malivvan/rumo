package cui

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

var (
	testBoxBackgroundColor         = tcell.NewHexColor(0x800000)
	testProgressBarBackgroundColor = tcell.NewHexColor(0x008080)
	testModalBackgroundColor       = tcell.NewHexColor(0x808000)
)

func TestBoxDrawUsesConfiguredBackgroundColor(t *testing.T) {
	t.Parallel()

	b := NewBox()
	b.SetRect(0, 0, 4, 2)
	b.SetBackgroundColor(testBoxBackgroundColor)

	app, err := newTestApp(b)
	if err != nil {
		t.Fatalf("failed to initialize App: %v", err)
	}

	b.Draw(app.screen)

	_, style, _ := app.screen.Get(1, 1)
	if got := style.GetBackground(); got != testBoxBackgroundColor {
		t.Fatalf("expected box background %v, got %v", testBoxBackgroundColor, got)
	}
}

func TestProgressBarDrawUsesConfiguredBackgroundColor(t *testing.T) {
	t.Parallel()

	p := NewProgressBar()
	p.SetRect(0, 0, 4, 1)
	p.SetBackgroundColor(testProgressBarBackgroundColor)
	p.SetProgress(50)

	app, err := newTestApp(p)
	if err != nil {
		t.Fatalf("failed to initialize App: %v", err)
	}

	p.Draw(app.screen)

	for x := 0; x < 4; x++ {
		_, style, _ := app.screen.Get(x, 0)
		if got := style.GetBackground(); got != testProgressBarBackgroundColor {
			t.Fatalf("expected progress bar background %v at x=%d, got %v", testProgressBarBackgroundColor, x, got)
		}
	}
}

func TestModalSetBackgroundColorPropagatesToEmbeddedWidgets(t *testing.T) {
	t.Parallel()

	m := NewModal()
	m.SetBackgroundColor(testModalBackgroundColor)

	if got := m.GetBackgroundColor(); got != testModalBackgroundColor {
		t.Fatalf("expected modal background %v, got %v", testModalBackgroundColor, got)
	}
	if got := m.GetFrame().GetBackgroundColor(); got != testModalBackgroundColor {
		t.Fatalf("expected modal frame background %v, got %v", testModalBackgroundColor, got)
	}
	if got := m.GetForm().GetBackgroundColor(); got != testModalBackgroundColor {
		t.Fatalf("expected modal form background %v, got %v", testModalBackgroundColor, got)
	}
}
