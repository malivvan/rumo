//go:build js && wasm

// SharedArrayBuffer-backed chan transport.
//
// The coordinator allocates a SharedArrayBuffer for every buffered chan
// created via `chan.create`. The buffer is sent (shared, not copied) to the
// requesting vm-host through postMessage and cached locally so that future
// send/recv calls on the chan operate directly on shared memory using
// Atomics.compareExchange / Atomics.add / Atomics.notify / Atomics.waitAsync.
// This bypasses the per-op postMessage round-trip to the coordinator that
// the original RPC-based ChanTransport required.
//
// Other vm-hosts that receive a chan id (e.g. via a marshaled value flowing
// through another chan or as an argument to `go fn()`) discover the SAB
// lazily via the `chan.lookup` op and cache it the same way. If
// SharedArrayBuffer or Atomics.waitAsync isn't available in the current
// scope (no cross-origin isolation, or unbuffered chans for which we still
// use the RPC path), we fall back to the original RPC-via-coordinator path
// transparently.
//
// Layout (Int32 atomic header followed by `cap` fixed-size byte slots):
//
//	[ 0] LOCK            simple mutex (0=free, 1=held)
//	[ 1] CLOSED          set to 1 on close; sticky
//	[ 2] WRITE_IDX       next slot a sender will fill (0..cap-1)
//	[ 3] READ_IDX        next slot a receiver will drain (0..cap-1)
//	[ 4] COUNT           number of buffered items in [0..cap]
//	[ 5] CAP             capacity in slots
//	[ 6] SLOT_BYTES      bytes per slot (incl. the 4-byte length prefix)
//	[ 7] SEND_TICK       bumped on every successful Recv (wakes blocked senders)
//	[ 8] RECV_TICK       bumped on every successful Send (wakes blocked receivers)
//
// Each slot is `[u32 len][bytes...]`. A payload bigger than `slot - 4` is
// rejected with ErrSABTooBig — callers (the chan transport) fall back to
// RPC for that single op, accepting that ordering across the slow/fast
// paths is not preserved (this matches the behaviour of unbuffered Go
// channels under heavy contention and is documented in the API surface).
package main

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"syscall/js"
)

var (
	jsAtomics    = js.Global().Get("Atomics")
	jsSAB        = js.Global().Get("SharedArrayBuffer")
	jsInt32Array = js.Global().Get("Int32Array")
	jsUint8Ctor  = js.Global().Get("Uint8Array")
)

const (
	sabIdxLock     = 0
	sabIdxClosed   = 1
	sabIdxWriteIdx = 2
	sabIdxReadIdx  = 3
	sabIdxCount    = 4
	sabIdxCap      = 5
	sabIdxSlot     = 6
	sabIdxSendTick = 7
	sabIdxRecvTick = 8

	sabHdrInts          = 9
	sabHdrBytes         = sabHdrInts * 4
	sabDefaultSlotBytes = 8 * 1024
)

// ErrSABTooBig is returned when a marshalled value won't fit in a SAB slot.
// The caller (remoteChanTransport) treats it as a signal to fall back to
// the RPC path for that single op.
var ErrSABTooBig = errors.New("rumo: chan value too large for SAB slot")

var (
	errSABClosedSend  = errors.New("send on closed chan")
	errSABClosedClose = errors.New("close of closed chan")
)

// sabSupported reports whether the current JS scope can construct
// SharedArrayBuffer and call Atomics.waitAsync. Both require a
// cross-origin-isolated context (COOP / COEP headers — see cmd/web/server.go).
func sabSupported() bool {
	if jsSAB.IsUndefined() || jsAtomics.IsUndefined() {
		return false
	}
	if jsAtomics.Get("waitAsync").Type() != js.TypeFunction {
		return false
	}
	return true
}

// sabRing is a Go-side wrapper around the JS shared header + slot region.
// It is safe for concurrent use across goroutines on the same wasm
// instance, and for concurrent use across wasm instances that share the
// same SAB (which is exactly the cross-worker case we need).
type sabRing struct {
	sab   js.Value
	i32   js.Value // Int32Array view of the header
	bytes js.Value // Uint8Array view of the slot region
	slot  int
	cap   int
}

func newSABRing(capSlots, slotBytes int) *sabRing {
	if capSlots <= 0 {
		capSlots = 1
	}
	if slotBytes <= 0 {
		slotBytes = sabDefaultSlotBytes
	}
	total := sabHdrBytes + capSlots*slotBytes
	sab := jsSAB.New(total)
	r := wrapSABRingWith(sab, capSlots, slotBytes)
	jsAtomics.Call("store", r.i32, sabIdxCap, capSlots)
	jsAtomics.Call("store", r.i32, sabIdxSlot, slotBytes)
	return r
}

func wrapSABRing(sab js.Value) *sabRing { return wrapSABRingWith(sab, 0, 0) }

func wrapSABRingWith(sab js.Value, capSlots, slotBytes int) *sabRing {
	i32 := jsInt32Array.New(sab, 0, sabHdrInts)
	if capSlots == 0 {
		capSlots = int(jsAtomics.Call("load", i32, sabIdxCap).Int())
	}
	if slotBytes == 0 {
		slotBytes = int(jsAtomics.Call("load", i32, sabIdxSlot).Int())
	}
	if slotBytes <= 0 {
		slotBytes = sabDefaultSlotBytes
	}
	if capSlots <= 0 {
		capSlots = 1
	}
	bytes := jsUint8Ctor.New(sab, sabHdrBytes, capSlots*slotBytes)
	return &sabRing{sab: sab, i32: i32, bytes: bytes, slot: slotBytes, cap: capSlots}
}

func (r *sabRing) load(idx int) int32 {
	return int32(jsAtomics.Call("load", r.i32, idx).Int())
}
func (r *sabRing) store(idx int, v int32) {
	jsAtomics.Call("store", r.i32, idx, v)
}
func (r *sabRing) add(idx int, d int32) int32 {
	return int32(jsAtomics.Call("add", r.i32, idx, d).Int())
}
func (r *sabRing) cas(idx int, expect, replace int32) int32 {
	return int32(jsAtomics.Call("compareExchange", r.i32, idx, expect, replace).Int())
}
func (r *sabRing) notify(idx int) {
	jsAtomics.Call("notify", r.i32, idx)
}

// waitAsyncMS blocks the calling goroutine (but NOT the JS event loop)
// until either the indexed cell changes from `expect`, or `timeoutMs`
// milliseconds have elapsed. timeoutMs<0 waits indefinitely. Returns the
// JS resolution string ("ok", "not-equal", "timed-out").
func (r *sabRing) waitAsyncMS(idx int, expect int32, timeoutMs float64) string {
	var res js.Value
	if timeoutMs < 0 {
		res = jsAtomics.Call("waitAsync", r.i32, idx, expect)
	} else {
		res = jsAtomics.Call("waitAsync", r.i32, idx, expect, timeoutMs)
	}
	// Synchronous resolution (the cell already differs, or timeout was 0).
	if !res.Get("async").Bool() {
		v := res.Get("value")
		if v.Type() == js.TypeString {
			return v.String()
		}
		return "ok"
	}
	done := make(chan string, 1)
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) any {
		s := "ok"
		if len(args) > 0 && args[0].Type() == js.TypeString {
			s = args[0].String()
		}
		select {
		case done <- s:
		default:
		}
		fn.Release()
		return nil
	})
	res.Get("value").Call("then", fn)
	return <-done
}

func (r *sabRing) lock(ctx context.Context) error {
	// Fast path.
	if r.cas(sabIdxLock, 0, 1) == 0 {
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r.cas(sabIdxLock, 0, 1) == 0 {
			return nil
		}
		// Block until the lock holder unlocks (notify) or 50ms elapses.
		// The short timeout serves both as a ctx-cancellation poll and as
		// a backstop against a missed notify (defensive only — unlock
		// always notifies).
		r.waitAsyncMS(sabIdxLock, 1, 50)
	}
}

func (r *sabRing) unlock() {
	r.store(sabIdxLock, 0)
	r.notify(sabIdxLock)
}

func (r *sabRing) writeSlot(slot int, blob []byte) {
	base := slot * r.slot
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(blob)))
	js.CopyBytesToJS(r.bytes.Call("subarray", base, base+4), hdr[:])
	if len(blob) > 0 {
		js.CopyBytesToJS(r.bytes.Call("subarray", base+4, base+4+len(blob)), blob)
	}
}

func (r *sabRing) readSlot(slot int) []byte {
	base := slot * r.slot
	var hdr [4]byte
	js.CopyBytesToGo(hdr[:], r.bytes.Call("subarray", base, base+4))
	n := int(binary.LittleEndian.Uint32(hdr[:]))
	if n <= 0 {
		return nil
	}
	out := make([]byte, n)
	js.CopyBytesToGo(out, r.bytes.Call("subarray", base+4, base+4+n))
	return out
}

// Send copies blob into the next free slot, blocking (cooperatively, via
// Atomics.waitAsync) until space is available. Returns ErrSABTooBig if the
// payload doesn't fit a single slot — the caller should fall back to RPC
// for that op.
func (r *sabRing) Send(ctx context.Context, blob []byte) error {
	if len(blob) > r.slot-4 {
		return ErrSABTooBig
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.lock(ctx); err != nil {
			return err
		}
		if r.load(sabIdxClosed) != 0 {
			r.unlock()
			return errSABClosedSend
		}
		count := r.load(sabIdxCount)
		if int(count) < r.cap {
			wIdx := r.load(sabIdxWriteIdx)
			r.writeSlot(int(wIdx), blob)
			r.store(sabIdxWriteIdx, (wIdx+1)%int32(r.cap))
			r.add(sabIdxCount, 1)
			r.add(sabIdxRecvTick, 1)
			r.unlock()
			r.notify(sabIdxRecvTick)
			return nil
		}
		// Full. Sleep on send-tick changes (incremented by Recv / Close).
		snap := r.load(sabIdxSendTick)
		r.unlock()
		r.waitAsyncMS(sabIdxSendTick, snap, 100)
	}
}

// Recv blocks until a value is available (returns ok=true), or the chan is
// closed with no buffered items (returns ok=false). The blob is the
// vm.MarshalLive payload originally written by a Send; the caller is
// responsible for UnmarshalLive + ResolveChans.
func (r *sabRing) Recv(ctx context.Context) (blob []byte, ok bool, err error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		if err := r.lock(ctx); err != nil {
			return nil, false, err
		}
		count := r.load(sabIdxCount)
		if count > 0 {
			rIdx := r.load(sabIdxReadIdx)
			b := r.readSlot(int(rIdx))
			r.store(sabIdxReadIdx, (rIdx+1)%int32(r.cap))
			r.add(sabIdxCount, -1)
			r.add(sabIdxSendTick, 1)
			r.unlock()
			r.notify(sabIdxSendTick)
			return b, true, nil
		}
		if r.load(sabIdxClosed) != 0 {
			r.unlock()
			return nil, false, nil
		}
		snap := r.load(sabIdxRecvTick)
		r.unlock()
		r.waitAsyncMS(sabIdxRecvTick, snap, 100)
	}
}

// Close marks the ring as closed and wakes every waiter. Idempotency:
// returns errSABClosedClose if the chan was already closed.
func (r *sabRing) Close() error {
	if err := r.lock(context.Background()); err != nil {
		return err
	}
	if r.load(sabIdxClosed) != 0 {
		r.unlock()
		return errSABClosedClose
	}
	r.store(sabIdxClosed, 1)
	r.add(sabIdxSendTick, 1)
	r.add(sabIdxRecvTick, 1)
	r.unlock()
	r.notify(sabIdxSendTick)
	r.notify(sabIdxRecvTick)
	return nil
}

// sabRegistry is a per-vm-host cache of chan id → sabRing wrappers, so
// repeat send/recv on the same chan don't re-construct Int32Array views.
type sabRegistry struct {
	mu    sync.Mutex
	rings map[int64]*sabRing
}

func newSABRegistry() *sabRegistry { return &sabRegistry{rings: map[int64]*sabRing{}} }

func (s *sabRegistry) get(id int64) *sabRing {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rings[id]
}

func (s *sabRegistry) put(id int64, r *sabRing) {
	s.mu.Lock()
	s.rings[id] = r
	s.mu.Unlock()
}

func (s *sabRegistry) del(id int64) {
	s.mu.Lock()
	delete(s.rings, id)
	s.mu.Unlock()
}

// coordSABStore is the coordinator-side registry that maps chan ids to the
// JS SharedArrayBuffer values it allocated. Used to serve `chan.lookup`
// requests from vm-hosts that received a chan id by other means (e.g. via
// a marshalled value flowing through another chan, or as a `go fn()` arg).
type coordSABStore struct {
	mu   sync.Mutex
	sabs map[int64]js.Value
}

func newCoordSABStore() *coordSABStore { return &coordSABStore{sabs: map[int64]js.Value{}} }

func (s *coordSABStore) put(id int64, sab js.Value) {
	s.mu.Lock()
	s.sabs[id] = sab
	s.mu.Unlock()
}

func (s *coordSABStore) get(id int64) (js.Value, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sabs[id]
	return v, ok
}

func (s *coordSABStore) del(id int64) {
	s.mu.Lock()
	delete(s.sabs, id)
	s.mu.Unlock()
}

// coordSABs is the coordinator-process singleton store. Nil-safe: only the
// coordinator role mutates it (under inSharedWorker()).
var coordSABs = newCoordSABStore()

