package vte

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	tests := []struct {
		name     string
		event    *tcell.EventKey
		expected string
	}{
		{
			name: "rune",
			event: tcell.NewEventKey(
				tcell.KeyRune,
				"j",
				tcell.ModNone,
			),
			expected: "j",
		},
		{
			name: "F1",
			event: tcell.NewEventKey(
				tcell.KeyF1,
				"",
				tcell.ModNone,
			),
			expected: "\x1bOP",
		},
		{
			name: "Shift-right",
			event: tcell.NewEventKey(
				tcell.KeyRight,
				"",
				tcell.ModShift,
			),
			expected: "\x1b[1;2C",
		},
		{
			name: "Ctrl-Shift-right",
			event: tcell.NewEventKey(
				tcell.KeyRight,
				"",
				tcell.ModShift|tcell.ModCtrl,
			),
			expected: "\x1b[1;6C",
		},
		{
			name: "Alt-Shift-right",
			event: tcell.NewEventKey(
				tcell.KeyRight,
				"",
				tcell.ModShift|tcell.ModAlt,
			),
			expected: "\x1b[1;4C",
		},
		{
			name: "rune + mod alt",
			event: tcell.NewEventKey(
				tcell.KeyRune,
				"j",
				tcell.ModAlt,
			),
			expected: "\x1Bj",
		},
		{
			name: "rune + mod ctrl",
			event: tcell.NewEventKey(
				tcell.KeyCtrlJ,
				string([]byte{0x0A}),
				tcell.ModCtrl,
			),
			expected: "\n",
		},
		{
			name: "shift + f5",
			event: tcell.NewEventKey(
				tcell.KeyF5,
				"",
				tcell.ModShift,
			),
			expected: "\x1B[15;2~",
		},
		{
			name: "shift + arrow",
			event: tcell.NewEventKey(
				tcell.KeyRight,
				"",
				tcell.ModShift,
			),
			expected: "\x1B[1;2C",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := keyCode(test.event)
			assert.Equal(t, test.expected, actual)
		})
	}
}

// VTE-004: Alt+F4 should map to F52, not F53.
func TestKey_AltF4(t *testing.T) {
	ev := tcell.NewEventKey(tcell.KeyF4, "", tcell.ModAlt)
	actual := keyCode(ev)
	assert.Equal(t, info.KeyF52, actual, "Alt+F4 should produce F52 code")
}

// VTE-005: Meta+Shift+Left/Right escape codes should not be swapped.
func TestKey_MetaShfLeftRight(t *testing.T) {
	// Left should end with D, Right with C
	assert.Contains(t, info.KeyMetaShfLeft, "D", "Meta+Shift+Left should end with D")
	assert.Contains(t, info.KeyMetaShfRight, "C", "Meta+Shift+Right should end with C")
}

// VTE-021: Alt+Shift+F5 through F12 should produce key codes.
func TestKey_AltShiftF5(t *testing.T) {
	ev := tcell.NewEventKey(tcell.KeyF5, "", tcell.ModAlt|tcell.ModShift)
	actual := keyCode(ev)
	assert.NotEmpty(t, actual, "Alt+Shift+F5 should produce a key code")
}

// VTE-004 regression: Alt+F5 through F12 should produce correct codes after fix.
func TestKey_AltFKeys(t *testing.T) {
	tests := []struct {
		key      tcell.Key
		expected string
	}{
		{tcell.KeyF1, info.KeyF49},
		{tcell.KeyF2, info.KeyF50},
		{tcell.KeyF3, info.KeyF51},
		{tcell.KeyF4, info.KeyF52},
		{tcell.KeyF5, info.KeyF53},
		{tcell.KeyF6, info.KeyF54},
		{tcell.KeyF7, info.KeyF55},
		{tcell.KeyF8, info.KeyF56},
		{tcell.KeyF9, info.KeyF57},
		{tcell.KeyF10, info.KeyF58},
		{tcell.KeyF11, info.KeyF59},
		{tcell.KeyF12, info.KeyF60},
	}
	for _, tt := range tests {
		ev := tcell.NewEventKey(tt.key, "", tcell.ModAlt)
		actual := keyCode(ev)
		assert.Equal(t, tt.expected, actual, "Alt+F%d", int(tt.key-tcell.KeyF1)+1)
	}
}

// VTE-021 regression: Alt+Shift F5-F12 should produce non-empty key codes.
func TestKey_AltShiftFKeys(t *testing.T) {
	for k := tcell.KeyF5; k <= tcell.KeyF12; k++ {
		ev := tcell.NewEventKey(k, "", tcell.ModAlt|tcell.ModShift)
		actual := keyCode(ev)
		assert.NotEmpty(t, actual, "Alt+Shift+F%d should produce a key code", int(k-tcell.KeyF1)+1)
	}
}
