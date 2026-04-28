package vm_test

import "testing"

func TestSwitch(t *testing.T) {
	// Basic tag switch
	expectRun(t, `
	x := 2
	out = ""
	switch x {
	case 1:
		out = "one"
	case 2:
		out = "two"
	case 3:
		out = "three"
	}
	`, nil, "two")

	// Default branch
	expectRun(t, `
	x := 99
	out = ""
	switch x {
	case 1:
		out = "one"
	default:
		out = "other"
	}
	`, nil, "other")

	// Multi-expression case
	expectRun(t, `
	x := 5
	out = ""
	switch x {
	case 1, 2:
		out = "small"
	case 3, 4, 5:
		out = "med"
	default:
		out = "big"
	}
	`, nil, "med")

	// First of multi-expression case matches
	expectRun(t, `
	x := 3
	out = ""
	switch x {
	case 3, 4, 5:
		out = "med"
	}
	`, nil, "med")

	// Tagless switch (boolean cases)
	expectRun(t, `
	out = ""
	switch {
	case 1 > 2:
		out = "gt"
	case 1 < 2:
		out = "lt"
	default:
		out = "eq"
	}
	`, nil, "lt")

	// Init statement
	expectRun(t, `
	out = ""
	switch x := 7; x {
	case 7:
		out = "seven"
	default:
		out = "other"
	}
	`, nil, "seven")

	// Init with tagless switch
	expectRun(t, `
	out = ""
	switch x := 3; {
	case x < 5:
		out = "low"
	default:
		out = "high"
	}
	`, nil, "low")

	// Auto-break (no implicit fallthrough)
	expectRun(t, `
	x := 1
	out = ""
	switch x {
	case 1:
		out += "a"
	case 2:
		out += "b"
	}
	`, nil, "a")

	// Explicit fallthrough
	expectRun(t, `
	x := 1
	out = ""
	switch x {
	case 1:
		out += "a"
		fallthrough
	case 2:
		out += "b"
	case 3:
		out += "c"
	}
	`, nil, "ab")

	// Fallthrough chain
	expectRun(t, `
	x := 1
	out = ""
	switch x {
	case 1:
		out += "a"
		fallthrough
	case 2:
		out += "b"
		fallthrough
	case 3:
		out += "c"
	case 4:
		out += "d"
	}
	`, nil, "abc")

	// Fallthrough into default
	expectRun(t, `
	x := 1
	out = ""
	switch x {
	case 1:
		out += "a"
		fallthrough
	default:
		out += "d"
	}
	`, nil, "ad")

	// break inside switch exits switch (not enclosing loop)
	expectRun(t, `
	out = 0
	for i := 0; i < 5; i++ {
		switch i {
		case 2:
			break
		}
		out += i
	}
	`, nil, 10) // 0+1+2+3+4

	// Per-case lexical scope
	expectRun(t, `
	x := 1
	out = ""
	switch x {
	case 1:
		y := "a"
		out = y
	case 2:
		y := "b"
		out = y
	}
	`, nil, "a")

	// Nested switch
	expectRun(t, `
	x := 1
	y := 2
	out = ""
	switch x {
	case 1:
		switch y {
		case 1:
			out = "1-1"
		case 2:
			out = "1-2"
		}
	case 2:
		out = "2"
	}
	`, nil, "1-2")

	// Empty case body
	expectRun(t, `
	x := 1
	out = "ok"
	switch x {
	case 1:
	case 2:
		out = "no"
	}
	`, nil, "ok")

	// Switch over strings
	expectRun(t, `
	s := "hello"
	out = 0
	switch s {
	case "hi":
		out = 1
	case "hello":
		out = 2
	case "hey":
		out = 3
	}
	`, nil, 2)
}

func TestSwitchErrors(t *testing.T) {
	expectError(t, `fallthrough`, nil,
		"fallthrough not allowed outside switch")
	expectError(t, `switch 1 { case 1: fallthrough }`, nil,
		"cannot fallthrough final case in switch")
	// Multiple defaults
	expectError(t, `switch 1 { default: a := 1; default: a := 2 }`, nil,
		"multiple default clauses in switch")
}

