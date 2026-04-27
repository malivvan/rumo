package vm

import (
	"context"
	"sync"
	"sync/atomic"
)

// Chan is the script-visible channel object. It exposes the same surface as
// the legacy `*Map{send,recv,close}` shape (so existing scripts work without
// changes) but routes through a pluggable Core, which can be either:
//
//   - LocalChanCore: backed by a Go channel (the native default).
//   - RemoteChanCore: backed by a ChanTransport, used by the js/wasm runtime
//     so that channels created in one SharedWorker can be used by send/recv
//     calls running in a different SharedWorker.
//
// Each Chan has a globally unique int64 id within the running process. The
// id is what travels over the wire when a Chan is marshalled into a routine
// spawn payload. The receiving side resolves the id either to its own local
// core (if it owns the channel) or to a RemoteChanCore that posts back to
// the owner via a ChanTransport.
type Chan struct {
	ObjectImpl
	id   int64
	core ChanCore
}

// ChanCore is the abstract backing for a Chan. Native and remote chans both
// implement this. ctx is the caller's VM context — used to abort the call if
// the VM is being torn down.
type ChanCore interface {
	Send(ctx context.Context, val Object) error
	Recv(ctx context.Context) (Object, error)
	Close() error
	ID() int64
}

// nextChanID hands out unique chan ids. Chan ids are addressable across the
// transport layer; the coordinator's chan registry uses the same id space.
var nextChanID atomic.Int64

func newChanID() int64 { return nextChanID.Add(1) }

// NewLocalChan creates a buffered chan backed by a local Go channel. Used by
// the native runtime and by the coordinator SharedWorker (which owns chan
// queues and serves send/recv calls from remote vm-host workers).
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

// NewRemoteChan returns a Chan whose send/recv/close calls route through the
// supplied transport. Useful for vm-host SharedWorkers that need to interact
// with channels owned by the coordinator.
func NewRemoteChan(id int64, tr ChanTransport) *Chan {
	return &Chan{id: id, core: &RemoteChanCore{id: id, tr: tr}}
}

// ID returns the chan's globally-unique id. Used by the wire codec.
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
// that close over the Chan's core, so `c.send(v)` and `c.recv()` work the
// same on local and remote chans.
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

// LocalChanCore is the native, in-process backing for a Chan. It re-uses the
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

// ChanTransport is the abstract wire used by RemoteChanCore. It carries
// chan ops between SharedWorkers (in js/wasm) or between processes (any
// future RPC backend). Implementations must serialise the value via the live
// codec and may block until the remote side acks.
type ChanTransport interface {
	// SendOp blocks until the owner accepts the value or returns an error.
	SendOp(ctx context.Context, chanID int64, val Object) error
	// RecvOp blocks until the owner returns a value or an error.
	RecvOp(ctx context.Context, chanID int64) (Object, error)
	// CloseOp closes the chan on the owner side.
	CloseOp(ctx context.Context, chanID int64) error
}

// RemoteChanCore proxies ops to a ChanTransport. The id identifies the
// channel in the owner's registry.
type RemoteChanCore struct {
	id int64
	tr ChanTransport
}

func (r *RemoteChanCore) ID() int64 { return r.id }

func (r *RemoteChanCore) Send(ctx context.Context, val Object) error {
	return r.tr.SendOp(ctx, r.id, val)
}

func (r *RemoteChanCore) Recv(ctx context.Context) (Object, error) {
	return r.tr.RecvOp(ctx, r.id)
}

func (r *RemoteChanCore) Close() error {
	return r.tr.CloseOp(context.Background(), r.id)
}

// ChanRegistry tracks every locally-owned chan by id so that incoming
// transport messages can resolve the target. Both the coordinator (which
// hosts every queue) and any VM that creates a chan locally use this.
type ChanRegistry struct {
	mu     sync.RWMutex
	chans  map[int64]*Chan
}

// NewChanRegistry returns an empty registry.
func NewChanRegistry() *ChanRegistry { return &ChanRegistry{chans: map[int64]*Chan{}} }

// Register inserts a chan into the registry, keyed by its id.
func (r *ChanRegistry) Register(c *Chan) {
	r.mu.Lock()
	r.chans[c.ID()] = c
	r.mu.Unlock()
}

// Lookup returns the chan associated with id (or nil).
func (r *ChanRegistry) Lookup(id int64) *Chan {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.chans[id]
}

// Forget removes id from the registry. Called by Close paths.
func (r *ChanRegistry) Forget(id int64) {
	r.mu.Lock()
	delete(r.chans, id)
	r.mu.Unlock()
}
