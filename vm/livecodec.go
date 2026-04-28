package vm

import (
	"github.com/malivvan/rumo/vm/codec"
)

// MarshalLive serialises an Object for live IPC (across SharedWorkers, RPC,
// etc.). Today it is a thin wrapper over MarshalObject — *Chan values get
// their core stripped automatically by encoding.go (see _chan case there) so
// only the chan id travels.
//
// Use ResolveChans on the receiving side to upgrade nil-core *Chan back to a
// LocalChanCore (when the receiver owns the chan) or a RemoteChanCore (when
// it does not).
func MarshalLive(o Object) ([]byte, error) {
	if o == nil {
		buf := make([]byte, codec.SizeByte())
		_ = MarshalObject(0, buf, nil)
		return buf, nil
	}
	size := SizeOfObject(o)
	buf := make([]byte, size)
	written := MarshalObject(0, buf, o)
	if written != size {
		return nil, errLiveCodecLengthMismatch
	}
	return buf, nil
}

// UnmarshalLive deserialises an Object produced by MarshalLive. The returned
// object may contain *Chan values whose Core is nil — the caller must run
// ResolveChans before letting scripts call send/recv on them.
func UnmarshalLive(buf []byte) (Object, error) {
	_, o, err := UnmarshalObject(0, buf)
	return o, err
}

// errLiveCodecLengthMismatch indicates an internal inconsistency between
// SizeOfObject and MarshalObject. It is returned (rather than panicked)
// because live IPC must never crash the runtime on a peer-side bug.
var errLiveCodecLengthMismatch = newLiveCodecError("MarshalLive: encoded length mismatch")

type liveCodecError struct{ msg string }

func newLiveCodecError(s string) *liveCodecError { return &liveCodecError{msg: s} }

func (e *liveCodecError) Error() string { return e.msg }

// ResolveChans walks o and binds every *Chan with a nil Core to either a
// local core (when the chan is in registry) or a fresh RemoteChanCore using
// fallback. The walk handles arrays, maps, error wrappers, and
// CompiledFunction Free cells. ObjectPtr is followed.
//
// Cycles are tolerated through a visited-set keyed by *Chan/*Map/*Array to
// avoid infinite recursion on self-referential structures.
func ResolveChans(o Object, registry *ChanRegistry, fallback ChanTransport) {
	resolveChans(o, registry, fallback, make(map[Object]struct{}))
}

func resolveChans(o Object, registry *ChanRegistry, fb ChanTransport, seen map[Object]struct{}) {
	if o == nil {
		return
	}
	if _, ok := seen[o]; ok {
		return
	}
	switch v := o.(type) {
	case *Chan:
		seen[v] = struct{}{}
		if v.core != nil {
			return
		}
		if registry != nil {
			if local := registry.Lookup(v.id); local != nil && local.core != nil {
				v.core = local.core
				return
			}
		}
		if fb != nil {
			v.core = &RemoteChanCore{id: v.id, tr: fb}
		}
	case *Array:
		seen[v] = struct{}{}
		v.mu.RLock()
		snap := append([]Object(nil), v.Value...)
		v.mu.RUnlock()
		for _, e := range snap {
			resolveChans(e, registry, fb, seen)
		}
	case *Map:
		seen[v] = struct{}{}
		v.mu.RLock()
		snap := make(map[string]Object, len(v.Value))
		for k, e := range v.Value {
			snap[k] = e
		}
		v.mu.RUnlock()
		for _, e := range snap {
			resolveChans(e, registry, fb, seen)
		}
	case *Error:
		seen[v] = struct{}{}
		resolveChans(v.Value, registry, fb, seen)
	case *ObjectPtr:
		seen[v] = struct{}{}
		if v.Value != nil {
			resolveChans(*v.Value, registry, fb, seen)
		}
	case *CompiledFunction:
		seen[v] = struct{}{}
		for _, fp := range v.Free {
			resolveChans(fp, registry, fb, seen)
		}
	}
}
