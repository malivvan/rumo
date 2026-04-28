package vm_test

import "testing"

func TestSelectBasic(t *testing.T) {
	// Single recv case fires after the channel receives a value.
	expectRun(t, `
	c := chan(1)
	c.send(42)
	out = 0
	select {
	case v := c.recv():
		out = v
	}
	`, nil, 42)

	// Single send case fires when the buffered chan has room.
	expectRun(t, `
	c := chan(1)
	out = 0
	select {
	case c.send(7):
		out = 1
	}
	if c.recv() == 7 {
		out = out + 10
	}
	`, nil, 11)

	// Recv with discard (no LHS).
	expectRun(t, `
	c := chan(1)
	c.send(99)
	out = 0
	select {
	case c.recv():
		out = 1
	}
	`, nil, 1)
}

func TestSelectDefault(t *testing.T) {
	// Default fires when no case is ready.
	expectRun(t, `
	c := chan(0)
	out = "none"
	select {
	case v := c.recv():
		out = string(v)
	default:
		out = "default"
	}
	`, nil, "default")

	// Default with send: full buffered chan triggers default.
	expectRun(t, `
	c := chan(1)
	c.send(1)
	out = "none"
	select {
	case c.send(2):
		out = "sent"
	default:
		out = "default"
	}
	`, nil, "default")
}

func TestSelectMultiCase(t *testing.T) {
	// Two recv cases — only one is ready, so it must fire.
	expectRun(t, `
	a := chan(1)
	b := chan(1)
	b.send(2)
	out = 0
	select {
	case v := a.recv():
		out = v
	case v := b.recv():
		out = v + 100
	}
	`, nil, 102)

	// Mixed send/recv — only the send is ready.
	expectRun(t, `
	a := chan(0)
	b := chan(1)
	out = 0
	select {
	case v := a.recv():
		out = v
	case b.send(5):
		out = 9
	}
	`, nil, 9)
}

func TestSelectRecvOk(t *testing.T) {
	// v, ok form with closed channel: recv returns undefined and ok==false.
	expectRun(t, `
	c := chan(0)
	c.close()
	out = ""
	select {
	case v, ok := c.recv():
		if ok {
			out = "open"
		} else {
			out = "closed"
		}
		v = v
	}
	`, nil, "closed")

	// v, ok form with a sent value: ok==true.
	expectRun(t, `
	c := chan(1)
	c.send(11)
	out = 0
	select {
	case v, ok := c.recv():
		if ok {
			out = v
		}
	}
	`, nil, 11)
}

func TestSelectBlockingViaRoutine(t *testing.T) {
	// No case is ready initially; a goroutine sends after a moment.  The
	// select should block until the value arrives.
	expectRun(t, `
	c := chan(0)
	g := go func() {
		c.send(123)
	}()
	out = 0
	select {
	case v := c.recv():
		out = v
	}
	g.wait()
	`, nil, 123)
}

func TestSelectBreak(t *testing.T) {
	// `break` exits the select (not the surrounding loop).
	expectRun(t, `
	out = 0
	for i := 0; i < 3; i++ {
		c := chan(1)
		c.send(i)
		select {
		case v := c.recv():
			out = out + v
			break
		}
	}
	`, nil, 3) // 0 + 1 + 2
}

func TestSelectErrors(t *testing.T) {
	// Two `default` clauses are not allowed.
	expectError(t, `
	c := chan(1)
	select {
	default:
		_ = c
	default:
		_ = c
	}
	`, nil, "multiple default clauses in select")

	// A case must be a chan send/recv expression.
	expectError(t, `
	select {
	case 1 + 1:
	}
	`, nil, "select case")
}

