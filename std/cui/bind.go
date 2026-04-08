package cui

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v3"
)

type eventHandler func(ev *tcell.EventKey) *tcell.EventKey

// BindConfig maps keys to event handlers and processes key events.
type BindConfig struct {
	handlers map[string]eventHandler
	mutex    *sync.RWMutex
}

// NewBindConfig returns a new input configuration.
func NewBindConfig() *BindConfig {
	c := BindConfig{
		handlers: make(map[string]eventHandler),
		mutex:    new(sync.RWMutex),
	}

	return &c
}

// Set sets the handler for a key event string.
func (c *BindConfig) Set(s string, handler func(ev *tcell.EventKey) *tcell.EventKey) error {
	mod, key, str, err := DecodeBind(s)
	if err != nil {
		return err
	}

	if key == tcell.KeyRune {
		c.SetRune(mod, []rune(str)[0], handler)
	} else {
		c.SetKey(mod, key, handler)
	}
	return nil
}

// SetKey sets the handler for a key.
func (c *BindConfig) SetKey(mod tcell.ModMask, key tcell.Key, handler func(ev *tcell.EventKey) *tcell.EventKey) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Convert KeyCtrlA-Z to rune format.
	if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
		mod |= tcell.ModCtrl
		r := 'a' + (key - tcell.KeyCtrlA)
		c.handlers[fmt.Sprintf("%d:%d", mod, r)] = handler
		return
	}

	// Convert Shift+Tab to Backtab.
	if mod&tcell.ModShift != 0 && key == tcell.KeyTab {
		mod ^= tcell.ModShift
		key = tcell.KeyBacktab
	}

	c.handlers[fmt.Sprintf("%d-%d", mod, key)] = handler
}

// SetRune sets the handler for a rune.
func (c *BindConfig) SetRune(mod tcell.ModMask, ch rune, handler func(ev *tcell.EventKey) *tcell.EventKey) {
	// Some runes are identical to named keys. Set the bind on the matching
	// named key instead.
	switch ch {
	case '\t':
		c.SetKey(mod, tcell.KeyTab, handler)
		return
	case '\n':
		c.SetKey(mod, tcell.KeyEnter, handler)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.handlers[fmt.Sprintf("%d:%d", mod, ch)] = handler
}

// Capture handles key events.
func (c *BindConfig) Capture(ev *tcell.EventKey) *tcell.EventKey {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if ev == nil {
		return nil
	}

	mod := ev.Modifiers()
	key := ev.Key()
	str := ev.Str()

	// Convert KeyCtrlA-Z to rune format.
	if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
		mod |= tcell.ModCtrl
		str = string(rune('a' + (key - tcell.KeyCtrlA)))
		key = tcell.KeyRune
	}

	var keyName string
	if key != tcell.KeyRune {
		keyName = fmt.Sprintf("%d-%d", mod, key)
	} else {
		keyName = fmt.Sprintf("%d:%d", mod, []rune(str)[0])
	}

	handler := c.handlers[keyName]
	if handler != nil {
		return handler(ev)
	}
	return ev
}

// Clear removes all handlers.
func (c *BindConfig) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.handlers = make(map[string]eventHandler)
}

// Modifier labels
const (
	LabelCtrl  = "ctrl"
	LabelAlt   = "alt"
	LabelMeta  = "meta"
	LabelShift = "shift"
)

// ErrInvalidKeyEvent is the error returned when encoding or decoding a key event fails.
var ErrInvalidKeyEvent = errors.New("invalid key event")

// UnifyEnterKeys is a flag that determines whether or not KPEnter (keypad
// enter) key events are interpreted as Enter key events. When enabled, Ctrl+J
// key events are also interpreted as Enter key events.
var UnifyEnterKeys = true

var fullKeyNames = map[string]string{
	"backspace2": "Backspace",
	"pgup":       "PageUp",
	"pgdn":       "PageDown",
	"esc":        "Escape",
}

// DecodeBind decodes a string as a key or combination of keys.
func DecodeBind(s string) (mod tcell.ModMask, key tcell.Key, str string, err error) {
	if len(s) == 0 {
		return 0, 0, "", ErrInvalidKeyEvent
	}

	// Special case for plus rune decoding
	if s[len(s)-1:] == "+" {
		key = tcell.KeyRune
		str = "+"

		if len(s) == 1 {
			return mod, key, str, nil
		} else if len(s) == 2 {
			return 0, 0, "", ErrInvalidKeyEvent
		} else {
			s = s[:len(s)-2]
		}
	}

	split := strings.Split(s, "+")
DECODEPIECE:
	for _, piece := range split {
		// DecodeBind modifiers
		pieceLower := strings.ToLower(piece)
		switch pieceLower {
		case LabelCtrl:
			mod |= tcell.ModCtrl
			continue
		case LabelAlt:
			mod |= tcell.ModAlt
			continue
		case LabelMeta:
			mod |= tcell.ModMeta
			continue
		case LabelShift:
			mod |= tcell.ModShift
			continue
		}

		// DecodeBind key
		for shortKey, fullKey := range fullKeyNames {
			if pieceLower == strings.ToLower(fullKey) {
				pieceLower = shortKey
				break
			}
		}
		switch pieceLower {
		case "backspace":
			key = tcell.KeyBackspace2
			continue
		case "space", "spacebar":
			key = tcell.KeyRune
			str = " "
			continue
		}
		for k, keyName := range tcell.KeyNames {
			if pieceLower == strings.ToLower(strings.ReplaceAll(keyName, "-", "+")) {
				key = k
				if key < 0x80 {
					str = string(rune(k))
				}
				continue DECODEPIECE
			}
		}

		// DecodeBind rune
		if len(piece) > 1 {
			return 0, 0, "", ErrInvalidKeyEvent
		}

		key = tcell.KeyRune
		str = string(rune(piece[0]))
	}

	// Normalize Ctrl+A-Z to lowercase
	if mod&tcell.ModCtrl != 0 && key == tcell.KeyRune {
		str = strings.ToLower(str)
	}

	return mod, key, str, nil
}

// EncodeBind encodes a key or combination of keys a string.
func EncodeBind(mod tcell.ModMask, key tcell.Key, str string) (string, error) {
	var b strings.Builder
	var wrote bool

	if mod&tcell.ModCtrl != 0 {
		if key == tcell.KeyBackspace || key == tcell.KeyTab || key == tcell.KeyEnter {
			mod ^= tcell.ModCtrl
		} else {
			// Convert KeyCtrlA-Z to rune format.
			if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
				mod |= tcell.ModCtrl
				str = string(rune('a' + (key - tcell.KeyCtrlA)))
				key = tcell.KeyRune
			}
		}
	}

	if key != tcell.KeyRune {
		if UnifyEnterKeys && key == tcell.KeyCtrlJ {
			key = tcell.KeyEnter
		} else if key < 0x80 {
			str = string(rune(key))
		}
	}

	// EncodeBind modifiers
	if mod&tcell.ModCtrl != 0 {
		b.WriteString(upperFirst(LabelCtrl))
		wrote = true
	}
	if mod&tcell.ModAlt != 0 {
		if wrote {
			b.WriteRune('+')
		}
		b.WriteString(upperFirst(LabelAlt))
		wrote = true
	}
	if mod&tcell.ModMeta != 0 {
		if wrote {
			b.WriteRune('+')
		}
		b.WriteString(upperFirst(LabelMeta))
		wrote = true
	}
	if mod&tcell.ModShift != 0 {
		if wrote {
			b.WriteRune('+')
		}
		b.WriteString(upperFirst(LabelShift))
		wrote = true
	}

	if key == tcell.KeyRune && str == " " {
		if wrote {
			b.WriteRune('+')
		}
		b.WriteString("Space")
	} else if key != tcell.KeyRune {
		// EncodeBind key
		keyName := tcell.KeyNames[key]
		if keyName == "" {
			return "", ErrInvalidKeyEvent
		}
		keyName = strings.ReplaceAll(keyName, "-", "+")
		fullKeyName := fullKeyNames[strings.ToLower(keyName)]
		if fullKeyName != "" {
			keyName = fullKeyName
		}

		if wrote {
			b.WriteRune('+')
		}
		b.WriteString(keyName)
	} else {
		// EncodeBind rune
		if wrote {
			b.WriteRune('+')
		}
		b.WriteString(str)
	}

	return b.String(), nil
}

func upperFirst(s string) string {
	if len(s) <= 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
