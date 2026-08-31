package vm

import (
	"context"
	"testing"
)

// TestLocalChanRoundTrip exercises the script-visible IndexGet → BuiltinFunction
// surface of *Chan against a local Go-channel backing.
func TestLocalChanRoundTrip(t *testing.T) {
	c := NewLocalChan(2)
	send, err := c.IndexGet(&String{Value: "send"})
	if err != nil {
		t.Fatalf("IndexGet send: %v", err)
	}
	recv, err := c.IndexGet(&String{Value: "recv"})
	if err != nil {
		t.Fatalf("IndexGet recv: %v", err)
	}
	ctx := context.Background()
	if _, err := send.(*BuiltinFunction).Value(ctx, &Int{Value: 7}); err != nil {
		t.Fatalf("send: %v", err)
	}
	got, err := recv.(*BuiltinFunction).Value(ctx)
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if i, ok := got.(*Int); !ok || i.Value != 7 {
		t.Fatalf("expected Int(7), got %#v", got)
	}
}

// TestLocalChanIdentity checks reference semantics: copying a chan yields the
// same chan and equality is identity-based.
func TestLocalChanIdentity(t *testing.T) {
	c := NewLocalChan(0)
	if c.Copy() != Object(c) {
		t.Fatalf("Copy must return the same chan")
	}
	other := NewLocalChan(0)
	if c.Equals(other) {
		t.Fatalf("distinct chans must not be equal")
	}
	if c.Equals(&Int{Value: 1}) {
		t.Fatalf("chan must not equal an int")
	}
}
