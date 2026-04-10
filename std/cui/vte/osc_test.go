package vte

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOSC8(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   string
		expectedID string
	}{
		{
			name:       "no semicolon in URI",
			input:      "8;;https://example.com",
			expected:   "https://example.com",
			expectedID: "",
		},
		{
			name:       "no semicolon in URI, with id",
			input:      "8;id=hello;https://example.com",
			expected:   "https://example.com",
			expectedID: "hello",
		},
		{
			name:       "semicolon in URI",
			input:      "8;;https://example.com/semi;colon",
			expected:   "https://example.com/semi;colon",
			expectedID: "",
		},
		{
			name:       "multiple semicolons in URI",
			input:      "8;;https://example.com/s;e;m;i;colon",
			expected:   "https://example.com/s;e;m;i;colon",
			expectedID: "",
		},
		{
			name:       "semicolon in URI, with id",
			input:      "8;id=hello;https://example.com/semi;colon",
			expected:   "https://example.com/semi;colon",
			expectedID: "hello",
		},
		{
			name:       "terminating sequence",
			input:      "8;;",
			expected:   "",
			expectedID: "",
		},
		{
			name:       "terminating sequence with id",
			input:      "8;id=hello;",
			expected:   "",
			expectedID: "hello",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Simulate vt.osc
			selector, val, found := cutString(test.input, ";")
			if !found {
				return
			}
			assert.Equal(t, "8", selector)
			// parse the result
			url, id := osc8(val)
			assert.Equal(t, test.expected, url)
			assert.Equal(t, test.expectedID, id)
		})
	}
}

// VTE-026: OSC 1 (Set Icon Name) should emit an EventTitle.
func TestOSC1_SetIconName(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	var gotTitle string
	// Use a direct call to osc since we can't easily capture events without a pty
	// The osc function processes the payload; we just verify it doesn't crash
	// and recognizes selector "1".
	vt.osc("1;my-icon")
	// The event was posted to the events channel; drain it
	select {
	case ev := <-vt.events:
		if te, ok := ev.(*EventTitle); ok {
			gotTitle = te.Title()
		}
	default:
	}
	assert.Equal(t, "my-icon", gotTitle)
}

// VTE-027: OSC 4 (Change/Query Color Number) should be recognized without panic.
func TestOSC4_ColorNumber(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	assert.NotPanics(t, func() {
		vt.osc("4;1;rgb:ff/00/00")
	})
}

// VTE-027: OSC 10 (Set/Query Default Foreground Color) should be recognized.
func TestOSC10_DefaultForeground(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	assert.NotPanics(t, func() {
		vt.osc("10;rgb:ff/ff/ff")
	})
}

// VTE-027: OSC 11 (Set/Query Default Background Color) should be recognized.
func TestOSC11_DefaultBackground(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	assert.NotPanics(t, func() {
		vt.osc("11;rgb:00/00/00")
	})
}

// VTE-027: OSC 12 (Set/Query Default Cursor Color) should be recognized.
func TestOSC12_CursorColor(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	assert.NotPanics(t, func() {
		vt.osc("12;rgb:ff/ff/00")
	})
}

// VTE-027: OSC 52 (Clipboard Access) should emit an EventClipboard.
func TestOSC52_Clipboard(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)

	vt.osc("52;c;dGVzdA==")

	var gotClipboard string
	select {
	case ev := <-vt.events:
		if ce, ok := ev.(*EventClipboard); ok {
			gotClipboard = ce.Data()
		}
	default:
	}
	assert.Equal(t, "dGVzdA==", gotClipboard, "OSC 52 should emit clipboard event with base64 data")
}

// VTE-027: OSC 104 (Reset Color Number) should be recognized without panic.
func TestOSC104_ResetColor(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	assert.NotPanics(t, func() {
		vt.osc("104;1")
	})
}

// VTE-027: All recognized OSC selectors should not fall through to unhandled.
func TestOSC_AllRecognized(t *testing.T) {
	vt := New()
	vt.Resize(4, 1)
	// These should all be consumed without panic
	selectors := []string{
		"4;0;rgb:00/00/00",
		"10;?",
		"11;?",
		"12;?",
		"52;c;",
		"104",
	}
	for _, s := range selectors {
		assert.NotPanics(t, func() { vt.osc(s) }, "OSC %q should be handled", s)
	}
}
