package vm

import (
	"context"
	"sync/atomic"
)

// Chan is the script-visible channel object. It exposes the same surface as
// the legacy `*Map{send,recv,close}` shape (so existing scripts work without
// changes) but is a dedicated reference type so `select` and future
// operations can recognise channels reliably.
//
// Every Chan is backed by a LocalChanCore, which wraps the objchan
// implementation from routinevm.go so all close-state and abort semantics
// keep working unchanged. Channels are single-process objects: they are
// backed by a Go channel and never travel across process/worker boundaries.
type Chan struct {
	ObjectImpl
	id   int64
	core ChanCore
}

// ChanCore is the abstract backing for a Chan. LocalChanCore is the only
// implementation; ctx is the caller's VM context — used to abort the call if
// the VM is being torn down.
type ChanCore interface {
	Send(ctx context.Context, val Object) error
	Recv(ctx context.Context) (Object, error)
	Close() error
	ID() int64
}

// nextChanID hands out unique chan ids within the running process.
var nextChanID atomic.Int64

func newChanID() int64 { return nextChanID.Add(1) }

// NewLocalChan creates a buffered chan backed by a local Go channel.
func NewLocalChan(buf int) *Chan {
	id := newChanID()
	return &Chan{
		id: id,
		core: &LocalChanCore{
			id: id,
			oc: &objchan{ch: make(chan Object, buf)},
		},
	}
}

// ID returns the chan's unique id.
func (c *Chan) ID() int64 { return c.id }

// Core returns the underlying core. Mostly useful for tests.
func (c *Chan) Core() ChanCore { return c.core }

// TypeName returns "chan".
func (c *Chan) TypeName() string { return "chan" }

func (c *Chan) String() string { return "chan" }

func (c *Chan) IsFalsy() bool { return false }

// Copy returns the same chan; channels are reference-typed.
func (c *Chan) Copy() Object { return c }

// Equals returns true if the two chans share the same id.
func (c *Chan) Equals(other Object) bool {
	if oc, ok := other.(*Chan); ok {
		return c.id == oc.id
	}
	return false
}

// IndexGet exposes the script-visible methods. Returns BuiltinFunction values
// that close over the Chan's core, so `c.send(v)` and `c.recv()` work as
// before.
func (c *Chan) IndexGet(index Object) (Object, error) {
	name, ok := index.(*String)
	if !ok {
		return nil, ErrNotIndexable
	}
	switch name.Value {
	case "send":
		return &BuiltinFunction{Name: "chan.send", Value: c.scriptSend}, nil
	case "recv":
		return &BuiltinFunction{Name: "chan.recv", Value: c.scriptRecv}, nil
	case "close":
		return &BuiltinFunction{Name: "chan.close", Value: c.scriptClose}, nil
	}
	return UndefinedValue, nil
}

func (c *Chan) scriptSend(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if err := c.core.Send(ctx, args[0]); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Chan) scriptRecv(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}
	val, err := c.core.Recv(ctx)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return UndefinedValue, nil
	}
	return val, nil
}

func (c *Chan) scriptClose(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}
	return nil, c.core.Close()
}

// LocalChanCore is the in-process backing for a Chan. It re-uses the
// pre-existing objchan struct so that all the existing close-state and abort
// semantics keep working unchanged.
type LocalChanCore struct {
	id int64
	oc *objchan
}

func (l *LocalChanCore) ID() int64 { return l.id }

func (l *LocalChanCore) Send(ctx context.Context, val Object) error {
	_, err := l.oc.send(ctx, val)
	return err
}

func (l *LocalChanCore) Recv(ctx context.Context) (Object, error) {
	return l.oc.recv(ctx)
}

func (l *LocalChanCore) Close() error {
	_, err := l.oc.closeChan(context.Background())
	return err
}
