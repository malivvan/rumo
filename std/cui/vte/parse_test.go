package vte

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "UTF-8",
			input: "🔥",
			expected: []Sequence{
				Print('🔥'),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := parse.Next()
				if seq == nil {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestIn(t *testing.T) {
	tests := []struct {
		name     string
		inRange  []rune
		input    rune
		expected bool
	}{
		{
			name:     "endpoint min",
			inRange:  []rune{0x00, 0x20},
			input:    0x00,
			expected: true,
		},
		{
			name:     "endpoint max",
			inRange:  []rune{0x00, 0x20},
			input:    0x20,
			expected: true,
		},
		{
			name:     "within",
			inRange:  []rune{0x00, 0x20},
			input:    0x19,
			expected: true,
		},
		{
			name:     "outside",
			inRange:  []rune{0x00, 0x20},
			input:    0x21,
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := in(test.input, test.inRange[0], test.inRange[1])
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		isVals   []rune
		input    rune
		expected bool
	}{
		{
			name:     "multiple",
			isVals:   []rune{0x00, 0x20},
			input:    0x00,
			expected: true,
		},
		{
			name:     "single",
			isVals:   []rune{0x00},
			input:    0x00,
			expected: true,
		},
		{
			name:     "false multiple",
			isVals:   []rune{0x00, 0x20},
			input:    0x19,
			expected: false,
		},
		{
			name:     "false single",
			isVals:   []rune{0x00},
			input:    0x21,
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := is(test.input, test.isVals...)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestAnywhere(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected stateFn
	}{
		{
			name:     "0x18",
			input:    0x18,
			expected: ground,
		},
		{
			name:     "0x1A",
			input:    0x1A,
			expected: ground,
		},
		{
			name:     "0x1B",
			input:    0x1B,
			expected: escape,
		},
		{
			name:     "eof",
			input:    eof,
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parse := &Parser{
				sequences: make(chan Sequence, 2),
				state:     ground,
			}
			called := false
			parse.SetExitFunc(func() {
				called = true
			})
			actual := anywhere(test.input, parse)
			act := reflect.ValueOf(actual).Pointer()
			exp := reflect.ValueOf(test.expected).Pointer()
			assert.Equal(t, exp, act, "wrong return function")
			if test.expected != nil {
				assert.True(t, called, "exit function not called")
			}
		})
	}
}

func TestCSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "CSI Entry + C0",
			input: "a\x1b[\x00",
			expected: []Sequence{
				Print('a'),
				C0(0x00),
			},
		},
		{
			name:  "CSI Entry + escape",
			input: "a\x1b[\x1b",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Entry + ignore",
			input: "a\x1b[\x7F",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Entry + dispatch",
			input: "a\x1b[c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Intermediate: []rune{},
					Parameters:   []int{},
				},
			},
		},
		{
			name:  "CSI Param with collect first",
			input: "a\x1b[<c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{},
					Intermediate: []rune{'<'},
				},
			},
		},
		{
			name:  "CSI Param with colorspace",
			input: "a\x1b[38:2::0:0:0m",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'm',
					Parameters:   []int{38, 2, 0, 0, 0},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param with colorspace fg and bg",
			input: "a\x1b[38:2::0:0:0;48:2::0:0:0m",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'm',
					Parameters:   []int{38, 2, 0, 0, 0, 48, 2, 0, 0, 0},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param SGR with semicolons",
			input: "a\x1b[38;2;0;0;0;48;2;0;0;0m",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'm',
					Parameters:   []int{38, 2, 0, 0, 0, 48, 2, 0, 0, 0},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param",
			input: "a\x1b[0c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{0},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param + eof",
			input: "a\x1b[0",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Param + eof",
			input: "a\x1b[0\x00",
			expected: []Sequence{
				Print('a'),
				C0(0x00),
			},
		},
		{
			name:  "CSI Param + eof",
			input: "a\x1b[0\x7F\x00",
			expected: []Sequence{
				Print('a'),
				C0(0x00),
			},
		},
		{
			name:  "CSI Param with long param",
			input: "a\x1b[9999c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{9999},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param with multiple",
			input: "a\x1b[0;0c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{0, 0},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param with multiple blank",
			input: "a\x1b[;c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{0, 0},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param with multiple filled or blank",
			input: "a\x1b[;1c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{0, 1},
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "CSI Param + csiIgnore",
			input: "a\x1b[;1\x3Cc",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Param + escape",
			input: "a\x1b[;1\x1b",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Intermediate",
			input: "a\x1b[\x20\x20c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{},
					Intermediate: []rune{' ', ' '},
				},
			},
		},
		{
			name:  "CSI Intermediate + escape",
			input: "a\x1b[\x20\x20\x1b",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Intermediate + c0",
			input: "a\x1b[\x20\x20\x00c",
			expected: []Sequence{
				Print('a'),
				C0(0x00),
				CSI{
					Final:        'c',
					Parameters:   []int{},
					Intermediate: []rune{' ', ' '},
				},
			},
		},
		{
			name:  "CSI Intermediate + 7f ignore",
			input: "a\x1b[\x20\x20\x7Fc",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{},
					Intermediate: []rune{' ', ' '},
				},
			},
		},
		{
			name:  "CSI Intermediate + eof",
			input: "a\x1b[\x20\x20",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Intermediate + param",
			input: "a\x1b[0\x20\x20c",
			expected: []Sequence{
				Print('a'),
				CSI{
					Final:        'c',
					Parameters:   []int{0},
					Intermediate: []rune{' ', ' '},
				},
			},
		},
		{
			name:  "CSI Intermediate + param + ignore",
			input: "a\x1b[0\x20\x20\x30c",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Ignore + eof",
			input: "a\x1b[0\x20\x20\x30\x3A",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Ignore + esc",
			input: "a\x1b[0\x20\x20\x30\x1B",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "CSI Ignore + c0",
			input: "a\x1b[0\x20\x20\x30\x00c",
			expected: []Sequence{
				Print('a'),
				C0(0x00),
			},
		},
		{
			name:  "CSI Ignore + 7F ignore",
			input: "a\x1b[0\x20\x20\x30\x7Fc",
			expected: []Sequence{
				Print('a'),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := parse.Next()
				if seq == nil {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				t.Logf("%T", seq)
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestDCS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "DCS Entry + C0",
			input: "a\x1bP\x00",
			expected: []Sequence{
				Print('a'),
			},
		},
		{
			name:  "DCS Entry + end",
			input: "a\x1bPq",
			expected: []Sequence{
				Print('a'),
				DCS{
					Final:        'q',
					Intermediate: []rune{},
					Parameters:   []int{},
				},
				DCSEndOfData{},
			},
		},
		{
			name:  "DCS Entry + data + end",
			input: "a\x1bPq#0;2;0;\x1b\\",
			expected: []Sequence{
				Print('a'),
				DCS{
					Final:        'q',
					Intermediate: []rune{},
					Parameters:   []int{},
				},
				DCSData('#'),
				DCSData('0'),
				DCSData(';'),
				DCSData('2'),
				DCSData(';'),
				DCSData('0'),
				DCSData(';'),
				DCSEndOfData{},
				ESC{
					Final:        '\\',
					Intermediate: []rune{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := parse.Next()
				if seq == nil {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "ESC W",
			input: "a\x1bDc",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        'D',
					Intermediate: []rune{},
				},
				Print('c'),
			},
		},
		{
			name:  "ESC W",
			input: "a\x1bWc",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        'W',
					Intermediate: []rune{},
				},
				Print('c'),
			},
		},
		{
			name:  "ESC W with a C0",
			input: "a\x1b\x00Wc",
			expected: []Sequence{
				Print('a'),
				C0(0x00),
				ESC{
					Final:        'W',
					Intermediate: []rune{},
				},
				Print('c'),
			},
		},
		{
			name:  "with ignore",
			input: "a\x1b\x7FWc",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        'W',
					Intermediate: []rune{},
				},
				Print('c'),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			lex := NewParser(r)
			i := 0
			for {
				seq := lex.Next()
				if seq == nil {
					assert.Equal(t, len(test.expected), i, "fewer sequences than expected")
					break
				}
				assert.Equal(t, test.expected[i], seq)
				i += 1
				assert.LessOrEqual(t, i, len(test.expected), "more sequences than expected")
			}
		})
	}
}

func TestEscapeIntermediate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "ESC SP F",
			input: "a\x1b Fc",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        'F',
					Intermediate: []rune{' '},
				},
				Print('c'),
			},
		},
		{
			name:  "ESC # 3",
			input: "a\x1b#3c",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        '3',
					Intermediate: []rune{'#'},
				},
				Print('c'),
			},
		},
		{
			name:  "ESC ( B",
			input: "a\x1b(Bc",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        'B',
					Intermediate: []rune{'('},
				},
				Print('c'),
			},
		},
		{
			name:  "ESC ( B with C0",
			input: "a\x1b(\tBc",
			expected: []Sequence{
				Print('a'),
				C0('\t'),
				ESC{
					Final:        'B',
					Intermediate: []rune{'('},
				},
				Print('c'),
			},
		},
		{
			name:  "ESC ( B with ignore",
			input: "a\x1b(\x7FBc",
			expected: []Sequence{
				Print('a'),
				ESC{
					Final:        'B',
					Intermediate: []rune{'('},
				},
				Print('c'),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := parse.Next()
				if seq == nil {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

func TestGround(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "printables",
			input: "abc",
			expected: []Sequence{
				Print('a'),
				Print('b'),
				Print('c'),
			},
		},
		{
			name:  "printable with c0",
			input: string([]rune{'a', 0x00, 'c'}),
			expected: []Sequence{
				Print('a'),
				C0(0x00),
				Print('c'),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			lex := NewParser(r)
			i := 0
			for {
				seq := lex.Next()
				if seq == nil {
					break
				}
				assert.Equal(t, test.expected[i], seq)
				i += 1
			}
		})
	}
}

func TestOSC(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Sequence
	}{
		{
			name:  "OSC entry",
			input: "a\x1b\x5D",
			expected: []Sequence{
				Print('a'),
				OSC{},
			},
		},
		{
			name:  "OSC end ST",
			input: "a\x1B\x5D\x1B\x5C",
			expected: []Sequence{
				Print('a'),
				OSC{},
				ESC{
					Final:        0x5C,
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "OSC end CAN",
			input: "a\x1B\x5D\x1B\x18",
			expected: []Sequence{
				Print('a'),
				OSC{},
				C0(0x18),
			},
		},
		{
			name:  "OSC end SUB",
			input: "a\x1B\x5D\x1B\x1A",
			expected: []Sequence{
				Print('a'),
				OSC{},
				C0(0x1A),
			},
		},
		{
			name:  "OSC 8 ;; http://example.com",
			input: "a\x1B\x5D8;;http://example.com\x1b\x5CLink\x1b\x5D8;;\x1b\x5C",
			expected: []Sequence{
				Print('a'),
				OSC{
					Payload: []rune{
						'8',
						';',
						';',
						'h',
						't',
						't',
						'p',
						':',
						'/',
						'/',
						'e',
						'x',
						'a',
						'm',
						'p',
						'l',
						'e',
						'.',
						'c',
						'o',
						'm',
					},
				},
				ESC{
					Final:        '\\',
					Intermediate: []rune{},
				},
				Print('L'),
				Print('i'),
				Print('n'),
				Print('k'),
				OSC{
					Payload: []rune{
						'8',
						';',
						';',
					},
				},
				ESC{
					Final:        '\\',
					Intermediate: []rune{},
				},
			},
		},
		{
			name:  "OSC bell terminated",
			input: "a\x1B\x5D\ab",
			expected: []Sequence{
				Print('a'),
				OSC{},
				Print('b'),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := strings.NewReader(test.input)
			parse := NewParser(r)
			i := 0
			for {
				seq := parse.Next()
				if seq == nil {
					assert.Equal(t, len(test.expected), i, "wrong amount of sequences")
					break
				}
				if i < len(test.expected) {
					assert.Equal(t, test.expected[i], seq)
				}
				i += 1
			}
		})
	}
}

// VTE-025: 8-bit C1 code 0x9B should enter CSI state (same as ESC [).
func TestC1_CSI_8bit(t *testing.T) {
	// 0x9B followed by 'H' is CSI H (cursor home)
	input := string([]byte{0x9B}) + "H"
	r := strings.NewReader(input)
	parse := NewParser(r)
	seq := parse.Next()
	csi, ok := seq.(CSI)
	assert.True(t, ok, "expected CSI sequence from 8-bit C1 0x9B")
	assert.Equal(t, 'H', csi.Final)
}

// VTE-025: 8-bit C1 code 0x9D should enter OSC state (same as ESC ]).
func TestC1_OSC_8bit(t *testing.T) {
	// 0x9D followed by OSC data terminated by BEL
	input := string([]byte{0x9D}) + "2;title\a"
	r := strings.NewReader(input)
	parse := NewParser(r)
	seq := parse.Next()
	osc, ok := seq.(OSC)
	assert.True(t, ok, "expected OSC sequence from 8-bit C1 0x9D")
	assert.Equal(t, "2;title", string(osc.Payload))
}

// VTE-017: DCS passthrough should not redundantly set exit on every character.
// Verify that hook sets the exit function and passthrough data is received correctly.
func TestDCS_Passthrough(t *testing.T) {
	// DCS q (final char) followed by data "abc" then ST (ESC \)
	input := "\x1BPq" + "abc" + "\x1B\\"
	r := strings.NewReader(input)
	parse := NewParser(r)

	// First should be the DCS hook
	seq := parse.Next()
	dcs, ok := seq.(DCS)
	assert.True(t, ok, "expected DCS sequence")
	assert.Equal(t, 'q', dcs.Final)

	// Then the passthrough data
	for _, expected := range []rune{'a', 'b', 'c'} {
		seq = parse.Next()
		data, ok := seq.(DCSData)
		assert.True(t, ok, "expected DCSData")
		assert.Equal(t, expected, rune(data))
	}

	// Then end of data
	seq = parse.Next()
	_, ok = seq.(DCSEndOfData)
	assert.True(t, ok, "expected DCSEndOfData")
}
