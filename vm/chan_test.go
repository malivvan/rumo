package vm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
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

// TestChanWireFormat checks that a *Chan survives MarshalLive/UnmarshalLive
// with only its id; the receiving side resolves it via ResolveChans against
// a fake transport.
func TestChanWireFormat(t *testing.T) {
	c := NewLocalChan(0)
	buf, err := MarshalLive(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalLive(buf)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rc, ok := got.(*Chan)
	if !ok {
		t.Fatalf("expected *Chan, got %T", got)
	}
	if rc.ID() != c.ID() {
		t.Fatalf("id mismatch: got %d want %d", rc.ID(), c.ID())
	}
	if rc.core != nil {
		t.Fatalf("expected nil core post-unmarshal, got %T", rc.core)
	}

	// resolver should bind a RemoteChanCore through the fallback transport
	tr := &fakeChanTransport{}
	ResolveChans(rc, NewChanRegistry(), tr)
	if _, ok := rc.core.(*RemoteChanCore); !ok {
		t.Fatalf("expected RemoteChanCore, got %T", rc.core)
	}

	// send/recv go through the transport
	if err := rc.core.Send(context.Background(), &Int{Value: 9}); err != nil {
		t.Fatalf("remote send: %v", err)
	}
	v, err := rc.core.Recv(context.Background())
	if err != nil {
		t.Fatalf("remote recv: %v", err)
	}
	if i, ok := v.(*Int); !ok || i.Value != 9 {
		t.Fatalf("recv mismatch: %#v", v)
	}
}

// TestResolveChansLocalOwner ensures ResolveChans prefers a local core when
// the chan is registered locally (the receiver is the owner).
func TestResolveChansLocalOwner(t *testing.T) {
	owner := NewLocalChan(1)
	reg := NewChanRegistry()
	reg.Register(owner)

	// fake an unmarshaled placeholder with the same id, nil core
	placeholder := &Chan{id: owner.ID()}
	ResolveChans(placeholder, reg, &fakeChanTransport{}) // fallback should be ignored
	if _, ok := placeholder.core.(*LocalChanCore); !ok {
		t.Fatalf("expected LocalChanCore on owner side, got %T", placeholder.core)
	}
}

// fakeChanTransport is an in-memory transport used by chan tests.
type fakeChanTransport struct {
	mu  sync.Mutex
	buf []Object
}

func (f *fakeChanTransport) SendOp(ctx context.Context, _ int64, val Object) error {
	f.mu.Lock()
	f.buf = append(f.buf, val)
	f.mu.Unlock()
	return nil
}
func (f *fakeChanTransport) RecvOp(ctx context.Context, _ int64) (Object, error) {
	for i := 0; i < 50; i++ {
		f.mu.Lock()
		if len(f.buf) > 0 {
			v := f.buf[0]
			f.buf = f.buf[1:]
			f.mu.Unlock()
			return v, nil
		}
		f.mu.Unlock()
		time.Sleep(2 * time.Millisecond)
	}
	return nil, errors.New("fakeChanTransport: recv timeout")
}
func (f *fakeChanTransport) CloseOp(_ context.Context, _ int64) error { return nil }
