package vm_test

import "testing"

func TestGoto(t *testing.T) {
	// simple forward jump skips intermediate code
	expectRun(t, `
	out = 0
	goto end
	out = 1
end:
	out = 2
`, nil, 2)

	// backward jump forms a loop
	expectRun(t, `
	out = 0
	i := 0
loop:
	i++
	out += i
	if i < 3 {
		goto loop
	}
`, nil, 6) // 1+2+3

	// forward jump out of an if-block
	expectRun(t, `
	out = 0
	x := 5
	if x > 0 {
		out = 1
		goto done
	}
	out = 99
done:
`, nil, 1)

	// goto inside a function (function-scoped labels)
	expectRun(t, `
	f := func() {
		x := 0
	again:
		x++
		if x < 5 { goto again }
		return x
	}
	out = f()
`, nil, 5)

	// goto used to break out of a for loop
	expectRun(t, `
	out = 0
	for i := 0; i < 10; i++ {
		out = i
		if i == 3 { goto exit }
	}
	out = 99
exit:
`, nil, 3)
}

func TestGotoErrors(t *testing.T) {
	// undefined label
	expectError(t, `goto missing`, nil, "label 'missing' not defined")

	// duplicate label
	expectError(t, `
L:
	a := 1
L:
	a = 2
	goto L
`, nil, "label 'L' already defined")

	// labels are function-scoped: a label in the outer scope is not visible
	// from inside a nested function literal.
	expectError(t, `
outer:
	f := func() {
		goto outer
	}
	f()
`, nil, "label 'outer' not defined")
}

