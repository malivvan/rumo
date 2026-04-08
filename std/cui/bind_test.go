package cui

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
)

const pressTimes = 7

func TestBindConfig(t *testing.T) {
	t.Parallel()

	wg := make([]*sync.WaitGroup, len(testCases))

	config := NewBindConfig()
	for i, c := range testCases {
		wg[i] = new(sync.WaitGroup)
		wg[i].Add(pressTimes)

		i := i // Capture
		err := config.Set(c.encoded, func(ev *tcell.EventKey) *tcell.EventKey {
			wg[i].Done()
			return nil
		})
		if err != nil {
			t.Fatalf("failed to set keybind for %s: %s", c.encoded, err)
		}
	}

	done := make(chan struct{})
	timeout := time.After(5 * time.Second)

	go func() {
		for i := range testCases {
			wg[i].Wait()
		}

		done <- struct{}{}
	}()

	errs := make(chan error)
	for j := 0; j < pressTimes; j++ {
		for i, c := range testCases {
			i, c := i, c // Capture
			go func() {
				ev := config.Capture(tcell.NewEventKey(c.key, c.str, c.mod))
				if ev != nil {
					errs <- fmt.Errorf("failed to test capturing keybinds: failed to register case %d event %d %d %s", i, c.mod, c.key, c.str)
				}
			}()
		}
	}

	select {
	case err := <-errs:
		t.Fatal(err)
	case <-timeout:
		t.Fatal("timeout")
	case <-done:
	}
}

// Example of creating and using an input configuration.
func ExampleNewBindConfig() {
	// Create a new input configuration to store the key bindings.
	c := NewBindConfig()

	handleSave := func(ev *tcell.EventKey) *tcell.EventKey {
		// Save
		return nil
	}

	handleOpen := func(ev *tcell.EventKey) *tcell.EventKey {
		// Open
		return nil
	}

	handleExit := func(ev *tcell.EventKey) *tcell.EventKey {
		// Exit
		return nil
	}

	// Bind Alt+s.
	if err := c.Set("Alt+s", handleSave); err != nil {
		log.Fatalf("failed to set keybind: %s", err)
	}

	// Bind Alt+o.
	c.SetRune(tcell.ModAlt, 'o', handleOpen)

	// Bind Escape.
	c.SetKey(tcell.ModNone, tcell.KeyEscape, handleExit)

	// Capture input. This will differ based on the framework in use (if any).
	// When using tview or cui, call App.SetInputCapture before calling
	// App.Run.
	// app.SetInputCapture(c.Capture)
}

type testCase struct {
	mod     tcell.ModMask
	key     tcell.Key
	str     string
	encoded string
}

func (c testCase) String() string {
	var str string
	if c.str != "" {
		str = "-" + str
	}
	return fmt.Sprintf("%d-%d%s-%s", c.mod, c.key, str, c.encoded)
}

var testCases = []testCase{
	{mod: tcell.ModNone, key: tcell.KeyRune, str: "a", encoded: "a"},
	{mod: tcell.ModNone, key: tcell.KeyRune, str: "+", encoded: "+"},
	{mod: tcell.ModNone, key: tcell.KeyRune, str: ";", encoded: ";"},
	{mod: tcell.ModNone, key: tcell.KeyTab, str: string(rune(tcell.KeyTab)), encoded: "Tab"},
	{mod: tcell.ModNone, key: tcell.KeyEnter, str: string(rune(tcell.KeyEnter)), encoded: "Enter"},
	{mod: tcell.ModNone, key: tcell.KeyPgDn, str: "", encoded: "PageDown"},
	{mod: tcell.ModAlt, key: tcell.KeyRune, str: "a", encoded: "Alt+a"},
	{mod: tcell.ModAlt, key: tcell.KeyRune, str: "+", encoded: "Alt++"},
	{mod: tcell.ModAlt, key: tcell.KeyRune, str: ";", encoded: "Alt+;"},
	{mod: tcell.ModAlt, key: tcell.KeyRune, str: " ", encoded: "Alt+Space"},
	{mod: tcell.ModAlt, key: tcell.KeyRune, str: "1", encoded: "Alt+1"},
	{mod: tcell.ModAlt, key: tcell.KeyTab, str: string(rune(tcell.KeyTab)), encoded: "Alt+Tab"},
	{mod: tcell.ModAlt, key: tcell.KeyEnter, str: string(rune(tcell.KeyEnter)), encoded: "Alt+Enter"},
	{mod: tcell.ModAlt, key: tcell.KeyDelete, str: "", encoded: "Alt+Delete"},
	{mod: tcell.ModCtrl, key: tcell.KeyRune, str: "c", encoded: "Ctrl+c"},
	{mod: tcell.ModCtrl, key: tcell.KeyRune, str: "d", encoded: "Ctrl+d"},
	{mod: tcell.ModCtrl | tcell.ModAlt, key: tcell.KeyRune, str: "c", encoded: "Ctrl+Alt+c"},
	{mod: tcell.ModCtrl, key: tcell.KeyRune, str: " ", encoded: "Ctrl+Space"},
	{mod: tcell.ModCtrl | tcell.ModAlt, key: tcell.KeyRune, str: "+", encoded: "Ctrl+Alt++"},
	{mod: tcell.ModCtrl | tcell.ModShift, key: tcell.KeyRune, str: "+", encoded: "Ctrl+Shift++"},
}

func TestEncodeBind(t *testing.T) {
	t.Parallel()

	for _, c := range testCases {
		encoded, err := EncodeBind(c.mod, c.key, c.str)
		if err != nil {
			t.Errorf("failed to encode key %d %d %s: %s", c.mod, c.key, c.str, err)
		}
		if encoded != c.encoded {
			t.Errorf("failed to encode key %d %d %s: got %s, want %s", c.mod, c.key, c.str, encoded, c.encoded)
		}
	}
}

func TestDecodeBind(t *testing.T) {
	t.Parallel()

	for _, c := range testCases {
		mod, key, str, err := DecodeBind(c.encoded)
		if err != nil {
			t.Errorf("failed to decode key %s: %s", c.encoded, err)
		}
		if mod != c.mod {
			t.Errorf("failed to decode key %s: invalid modifiers: got %d, want %d", c.encoded, mod, c.mod)
		}
		if key != c.key {
			t.Errorf("failed to decode key %s: invalid key: got %d, want %d", c.encoded, key, c.key)
		}
		if str != c.str {
			t.Errorf("failed to decode key %s: invalid rune: got %s, want %s", c.encoded, str, c.str)
		}
	}
}
