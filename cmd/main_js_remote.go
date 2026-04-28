//go:build js && wasm

// Per-VM DedicatedWorker concurrency runtime for the js/wasm build.
//
// This file plugs into vm.Config to make `go fn()` and `chan(n)` spawn
// across DedicatedWorkers instead of using Go goroutines:
//
//   - vm.Config.Spawner: every `go fn()` allocates a fresh
//     `vm-<id>-<n>` DedicatedWorker, ships (fn, args, parent bytecode)
//     to it, and returns a RoutineHandle whose Result/Wait/Cancel route
//     through that worker's MessagePort.
//   - vm.Config.ChanFactory: every `chan(n)` registers a queue with the
//     coordinator SharedWorker and returns a vm.RemoteChan whose
//     send/recv/close hop through a vm.ChanTransport that also points at
//     the coordinator. The coordinator owns every queue; vm-hosts only
//     hold proxies.
//
// Why DedicatedWorker and not SharedWorker for the vm-host? A SharedWorker
// scope is forbidden from constructing further SharedWorkers
// (https://crbug.com/1102827), which would block any recursive `go fn()`
// fan-out beyond the first level. DedicatedWorker is constructable from
// every scope (Window, DedicatedWorker, SharedWorker) and supports
// MessagePort transfer — exactly what we need to also hand each child a
// private port to the coordinator.
//
// Because the JS thread doesn't allow synchronous blocking on
// postMessage replies the way Atomics.wait+SAB would, we instead rely on
// Go's goroutine scheduler: a vm-host goroutine that calls Send/Recv
// blocks on a Go channel; the worker port's onmessage handler
// (registered as a syscall/js Func) wakes it when the coordinator
// replies. Other goroutines remain runnable on the same JS thread.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"syscall/js"
	"testing/fstest"
	"time"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/vm"
)

// ---------------------------------------------------------------------------
// portCallClient — request/response over a MessagePort with a Go-channel
//                  futex per pending id
// ---------------------------------------------------------------------------

type portCallClient struct {
	port    js.Value
	nextID  atomic.Int64
	mu      sync.Mutex
	pending map[int64]chan js.Value
	subMu   sync.Mutex
	subs    []func(js.Value)
}

func newPortCallClient(port js.Value) *portCallClient {
	c := &portCallClient{port: port, pending: make(map[int64]chan js.Value)}
	port.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		c.dispatch(args[0].Get("data"))
		return nil
	}))
	// MessagePort and SharedWorker.port require .start(); Worker (and the
	// global self of a DedicatedWorkerGlobalScope) auto-starts and has no
	// such method. Call it only when present so we accept both shapes.
	if startFn := port.Get("start"); startFn.Type() == js.TypeFunction {
		port.Call("start")
	}
	return c
}

func (c *portCallClient) dispatch(data js.Value) {
	if data.IsUndefined() || data.IsNull() {
		return
	}
	if id := data.Get("id"); id.Type() == js.TypeNumber {
		iid := int64(id.Int())
		c.mu.Lock()
		ch, ok := c.pending[iid]
		if ok {
			delete(c.pending, iid)
		}
		c.mu.Unlock()
		if ok {
			ch <- data
			return
		}
	}
	c.subMu.Lock()
	subs := append([]func(js.Value){}, c.subs...)
	c.subMu.Unlock()
	for _, s := range subs {
		func() {
			defer func() { _ = recover() }()
			s(data)
		}()
	}
}

func (c *portCallClient) subscribe(fn func(js.Value)) {
	c.subMu.Lock()
	c.subs = append(c.subs, fn)
	c.subMu.Unlock()
}

func (c *portCallClient) call(ctx context.Context, op string, fill func(js.Value)) (js.Value, error) {
	id := c.nextID.Add(1)
	resCh := make(chan js.Value, 1)
	c.mu.Lock()
	c.pending[id] = resCh
	c.mu.Unlock()

	msg := jsObject.New()
	msg.Set("id", id)
	msg.Set("op", op)
	if fill != nil {
		fill(msg)
	}
	c.port.Call("postMessage", msg)

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return js.Undefined(), ctx.Err()
	case data := <-resCh:
		if e := data.Get("error"); e.Type() == js.TypeString {
			return js.Undefined(), errors.New(e.String())
		}
		return data.Get("result"), nil
	}
}

// post sends a fire-and-forget message (used for output forwarding etc.).
func (c *portCallClient) post(fill func(js.Value)) {
	msg := jsObject.New()
	if fill != nil {
		fill(msg)
	}
	c.port.Call("postMessage", msg)
}

// callTransfer is identical to call but forwards a list of transferables
// (typically MessagePort instances) to the peer. Required by the Spawner
// to hand each child its own private coordinator port.
func (c *portCallClient) callTransfer(ctx context.Context, op string, fill func(js.Value), transfer []js.Value) (js.Value, error) {
	id := c.nextID.Add(1)
	resCh := make(chan js.Value, 1)
	c.mu.Lock()
	c.pending[id] = resCh
	c.mu.Unlock()

	msg := jsObject.New()
	msg.Set("id", id)
	msg.Set("op", op)
	if fill != nil {
		fill(msg)
	}
	if len(transfer) > 0 {
		arr := js.Global().Get("Array").New(len(transfer))
		for i, t := range transfer {
			arr.SetIndex(i, t)
		}
		c.port.Call("postMessage", msg, arr)
	} else {
		c.port.Call("postMessage", msg)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return js.Undefined(), ctx.Err()
	case data := <-resCh:
		if e := data.Get("error"); e.Type() == js.TypeString {
			return js.Undefined(), errors.New(e.String())
		}
		return data.Get("result"), nil
	}
}

// ---------------------------------------------------------------------------
// Coordinator client (vm-host → coordinator)
// ---------------------------------------------------------------------------

type coordClient struct {
	*portCallClient
	workerURL string
}

// vm-host package-globals set up at runVM time.
var (
	myCoord     *coordClient
	myWorkerURL string
	// myRoutineID is the coordinator-assigned id of the routine that this
	// vm-host process is currently executing. The Spawner passes it as
	// `parentId` when allocating child routines so that the monitor can
	// render the routine tree (page-level routines as roots, every `go fn()`
	// inside the script as a child of its enclosing routine, etc.).
	myRoutineID int64
)

// spawnChildWorker creates a per-`go fn()` child worker at workerURL with
// the requested name as a plain DedicatedWorker.
//
// We deliberately do NOT try SharedWorker here, even when the host scope
// allows it. SharedWorker scopes are forbidden from constructing further
// SharedWorkers (https://crbug.com/1102827), which would block any
// recursive `go fn()` fan-out beyond the first level. DedicatedWorker is
// constructable from every scope (Window, DedicatedWorker, SharedWorker)
// and supports MessagePort transfer — the only thing we actually need.
//
// The returned `port` is the Worker object itself. It exposes the same
// postMessage / onmessage shape that newPortCallClient consumes.
func spawnChildWorker(workerURL, workerName string) (port js.Value, err error) {
	defer func() {
		if r := recover(); r != nil && port.IsUndefined() {
			err = fmt.Errorf("spawnChildWorker: panic: %v", r)
		}
	}()
	workerCtor := js.Global().Get("Worker")
	if workerCtor.IsUndefined() {
		return js.Undefined(), errors.New("Worker constructor not available")
	}
	opts := js.Global().Get("Object").New()
	opts.Set("name", workerName)
	w := workerCtor.New(workerURL, opts)
	if w.IsUndefined() {
		return js.Undefined(), errors.New("spawnChildWorker: Worker constructor returned undefined")
	}
	return w, nil
}

func openCoordClient(workerURL string) (cc *coordClient, err error) {
	defer func() {
		if r := recover(); r != nil {
			cc = nil
			err = fmt.Errorf("openCoordClient: panic: %v", r)
		}
	}()
	swCtor := js.Global().Get("SharedWorker")
	if swCtor.IsUndefined() {
		return nil, errors.New("SharedWorker constructor not available")
	}
	sw := swCtor.New(workerURL, "rumo-coordinator")
	port := sw.Get("port")
	if port.IsUndefined() {
		return nil, errors.New("vm-host: failed to obtain coordinator port")
	}
	pc := newPortCallClient(port)
	return &coordClient{portCallClient: pc, workerURL: workerURL}, nil
}

// newCoordFromPort wraps a transferable MessagePort that another peer
// (page or parent vm-host) handed to us as a coordClient. This is the
// preferred path: it requires no SharedWorker constructor in scope and
// thus works equally well from Window-, DedicatedWorker-, or
// SharedWorker-launched workers.
func newCoordFromPort(port js.Value, workerURL string) *coordClient {
	pc := newPortCallClient(port)
	return &coordClient{portCallClient: pc, workerURL: workerURL}
}

// isShipFriendly reports whether o can be marshalled with vm.MarshalLive
// AND meaningfully reconstructed on the other side via FixDecodedObject.
//
// Some VM-produced values are intentionally non-portable. The biggest
// offender is the routine handle returned by `go fn()`: it's a *Map
// containing three anonymous *BuiltinFunction entries (Name="") whose
// Value field is a closure over a local goroutine handle. Those marshal
// to a BuiltinFunction with empty Name; the child then can't find them
// in its own builtinFuncs table and aborts with `unknown builtin
// function: ""`. Detect that case (and the recursive variants) up front
// so the parent can ship Null for that global instead of poisoning the
// snapshot.
func isShipFriendly(o vm.Object) bool {
	switch v := o.(type) {
	case nil:
		return true
	case *vm.BuiltinFunction:
		// nameless builtins are runtime-bound closures; not portable.
		return v.Name != ""
	case *vm.Map:
		// Read without locking — the map's mu is unexported. The
		// snapshot we'd be racing against is the parent VM's; for
		// shipping-decisions on globals (which are seldom mutated
		// post-init by the VM after spawn-time) a best-effort scan is
		// acceptable.
		for _, e := range v.Value {
			if !isShipFriendly(e) {
				return false
			}
		}
		return true
	case *vm.ImmutableMap:
		// Modules are already keyed by moduleName so FixDecodedObject
		// rebinds them wholesale — no need to inspect children.
		if v.ModuleName() != "" {
			return true
		}
		for _, e := range v.Value {
			if !isShipFriendly(e) {
				return false
			}
		}
		return true
	case *vm.Array:
		for _, e := range v.Value {
			if !isShipFriendly(e) {
				return false
			}
		}
		return true
	case *vm.ImmutableArray:
		for _, e := range v.Value {
			if !isShipFriendly(e) {
				return false
			}
		}
		return true
	}
	return true
}

// ---------------------------------------------------------------------------
// remoteChanTransport — vm.ChanTransport via the coordinator
// ---------------------------------------------------------------------------

// remoteChanTransport routes chan ops to the coordinator. When the chan
// has an associated SharedArrayBuffer (advertised by the coordinator at
// chan.create time, or fetched lazily via chan.lookup), Send/Recv/Close
// operate directly on shared memory using Atomics.waitAsync — no
// postMessage round-trip per op. The RPC path remains as a fallback for
// unbuffered chans, environments without cross-origin isolation, and the
// (rare) oversized-payload case.
type remoteChanTransport struct {
	coord *coordClient
	sabs  *sabRegistry
}

func newRemoteChanTransport(coord *coordClient) *remoteChanTransport {
	return &remoteChanTransport{coord: coord, sabs: newSABRegistry()}
}

// sabFor returns the cached sabRing for chanID, performing a single
// chan.lookup RPC against the coordinator on the first miss. Returns nil
// if the chan is RPC-only (no SAB allocated, or SAB unsupported in this
// scope).
func (t *remoteChanTransport) sabFor(ctx context.Context, chanID int64) *sabRing {
	if !sabSupported() {
		return nil
	}
	if r := t.sabs.get(chanID); r != nil {
		return r
	}
	res, err := t.coord.call(ctx, "chan.lookup", func(m js.Value) {
		m.Set("chanId", chanID)
	})
	if err != nil {
		return nil
	}
	sab := res.Get("sab")
	if !sab.Truthy() {
		return nil
	}
	r := wrapSABRing(sab)
	t.sabs.put(chanID, r)
	return r
}

func (t *remoteChanTransport) SendOp(ctx context.Context, chanID int64, val vm.Object) error {
	blob, err := vm.MarshalLive(val)
	if err != nil {
		return err
	}
	if r := t.sabFor(ctx, chanID); r != nil {
		sErr := r.Send(ctx, blob)
		if sErr == nil {
			return nil
		}
		if !errors.Is(sErr, ErrSABTooBig) {
			return sErr
		}
		// Oversized payload: best-effort RPC fallback. Ordering across
		// the SAB and RPC paths for the same chan isn't preserved, but
		// the coordinator still has a queue for any RPC-only producer
		// to land on. In practice scripts rarely send values bigger
		// than the per-slot budget; documented limitation.
	}
	_, err = t.coord.call(ctx, "chan.send", func(m js.Value) {
		m.Set("chanId", chanID)
		m.Set("val", bytesToJS(blob))
	})
	return err
}

func (t *remoteChanTransport) RecvOp(ctx context.Context, chanID int64) (vm.Object, error) {
	if r := t.sabFor(ctx, chanID); r != nil {
		blob, ok, err := r.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			// closed and empty
			return vm.UndefinedValue, nil
		}
		if len(blob) == 0 {
			return vm.UndefinedValue, nil
		}
		o, uErr := vm.UnmarshalLive(blob)
		if uErr != nil {
			return nil, uErr
		}
		// any chan that flowed through the queue keeps its core nil; bind it.
		vm.ResolveChans(o, nil, t)
		return o, nil
	}
	res, err := t.coord.call(ctx, "chan.recv", func(m js.Value) { m.Set("chanId", chanID) })
	if err != nil {
		return nil, err
	}
	if res.IsUndefined() || res.IsNull() {
		return vm.UndefinedValue, nil
	}
	blob := toBytes(res)
	if len(blob) == 0 {
		return vm.UndefinedValue, nil
	}
	o, err := vm.UnmarshalLive(blob)
	if err != nil {
		return nil, err
	}
	vm.ResolveChans(o, nil, t)
	return o, nil
}

func (t *remoteChanTransport) CloseOp(ctx context.Context, chanID int64) error {
	var sabErr error
	if r := t.sabFor(ctx, chanID); r != nil {
		sabErr = r.Close()
		t.sabs.del(chanID)
	}
	// Always notify the coordinator so its registry forgets the chan and
	// future chan.lookup calls return null.
	_, err := t.coord.call(ctx, "chan.close", func(m js.Value) { m.Set("chanId", chanID) })
	if sabErr != nil && !errors.Is(sabErr, errSABClosedClose) {
		return sabErr
	}
	return err
}

// makeChanFactory returns a vm.Config-compatible ChanFactory that allocates
// chans via the coordinator and primes tr's SAB cache from the create reply,
// so the very first send/recv on a freshly-created buffered chan already
// uses the fast path.
func makeChanFactory(coord *coordClient, tr *remoteChanTransport) func(int) (vm.Object, error) {
	return func(buf int) (vm.Object, error) {
		res, err := coord.call(context.Background(), "chan.create", func(m js.Value) {
			m.Set("buf", buf)
		})
		if err != nil {
			return nil, err
		}
		chanID := int64(res.Get("chanId").Int())
		if sab := res.Get("sab"); sab.Truthy() {
			tr.sabs.put(chanID, wrapSABRing(sab))
		}
		return vm.NewRemoteChan(chanID, tr), nil
	}
}

// ---------------------------------------------------------------------------
// Spawner
// ---------------------------------------------------------------------------

// remoteRoutineHandle implements vm.RoutineHandle backed by a child
// DedicatedWorker. The handle holds the child port-client so cancel/wait
// can reach the running VM directly.
type remoteRoutineHandle struct {
	id     int64
	child  *portCallClient
	parent *coordClient

	doneOnce sync.Once
	doneCh   chan struct{}
	result   atomic.Pointer[vm.Object]
	err      atomic.Pointer[error]
}

func (h *remoteRoutineHandle) markDone(val vm.Object, err error) {
	h.doneOnce.Do(func() {
		if val != nil {
			h.result.Store(&val)
		}
		if err != nil {
			h.err.Store(&err)
		}
		close(h.doneCh)
	})
}

func (h *remoteRoutineHandle) Result(ctx context.Context) (vm.Object, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.doneCh:
	}
	if e := h.err.Load(); e != nil {
		// surface as Error{} so scripts can treat it like the goroutine path
		return &vm.Error{Value: &vm.String{Value: (*e).Error()}}, nil
	}
	if r := h.result.Load(); r != nil {
		return *r, nil
	}
	return vm.UndefinedValue, nil
}

func (h *remoteRoutineHandle) Wait(ctx context.Context, seconds int64) bool {
	if seconds < 0 {
		select {
		case <-h.doneCh:
			return true
		case <-ctx.Done():
			return false
		}
	}
	select {
	case <-h.doneCh:
		return true
	case <-time.After(time.Duration(seconds) * time.Second):
		return false
	case <-ctx.Done():
		return false
	}
}

func (h *remoteRoutineHandle) Cancel() {
	// best-effort: post a cancel op to the child; ignore reply
	defer func() { _ = recover() }()
	h.child.post(func(m js.Value) {
		m.Set("op", "cancel")
	})
}

// newSpawner builds a vm.Config.Spawner closure for the current vm-host. It
// captures the coordinator client and the parent's bytecode (which the child
// needs to resolve constant references inside the supplied fn).
func newSpawner(coord *coordClient, parentBytecode []byte) func(context.Context, vm.Object, []vm.Object) (vm.RoutineHandle, error) {
	return func(ctx context.Context, fn vm.Object, args []vm.Object) (vm.RoutineHandle, error) {
		// 1) ask the coordinator to allocate a routine + worker name
		alloc, err := coord.call(ctx, "routine.allocate", func(m js.Value) {
			m.Set("kind", "go")
			m.Set("parent", workerSelfName())
			m.Set("parentId", myRoutineID)
		})
		if err != nil {
			return nil, err
		}
		routineID := int64(alloc.Get("routineId").Int())
		workerName := alloc.Get("workerName").String()

		// 2) marshal closure + args
		fnBlob, err := vm.MarshalLive(fn)
		if err != nil {
			return nil, err
		}
		argsArr := js.Global().Get("Array").New(len(args))
		for i, a := range args {
			b, err := vm.MarshalLive(a)
			if err != nil {
				return nil, err
			}
			argsArr.SetIndex(i, bytesToJS(b))
		}

		// 2b) snapshot the parent VM's current globals and ship them so the
		// child can resolve OpGetGlobal references (e.g. `fmt`, `times`
		// imports set during top-level execution). Without this the child's
		// globals slice is empty and the routine body crashes on the first
		// module access.
		//
		// Some globals are intentionally non-portable — most notably the
		// *Map produced by wrapRoutineHandle, which contains anonymous
		// *BuiltinFunctions (Name="") whose Value field closes over local
		// goroutine state. Those would round-trip as nameless builtins and
		// fail FixDecodedObject on the child with `unknown builtin
		// function: ""`. The child never references its siblings'
		// handles anyway, so we substitute Null for any global that
		// isShipFriendly() rejects (or that MarshalLive itself rejects).
		globalsArr := js.Global().Get("Array").New(0)
		if pv, ok := ctx.Value(vm.ContextKey("vm")).(*vm.VM); ok && pv != nil {
			gs := pv.Globals()
			globalsArr = js.Global().Get("Array").New(len(gs))
			for i, g := range gs {
				if g == nil || !isShipFriendly(g) {
					globalsArr.SetIndex(i, js.Null())
					continue
				}
				gb, gErr := vm.MarshalLive(g)
				if gErr != nil {
					// non-portable global; ship Null and let the child
					// resolve OpGetGlobal at its own peril if the routine
					// happens to read this slot.
					globalsArr.SetIndex(i, js.Null())
					continue
				}
				globalsArr.SetIndex(i, bytesToJS(gb))
			}
		}

		// 3) spin up the child worker. We use a DedicatedWorker (plain
		//    `new Worker`) for every child: DedicatedWorker construction
		//    is allowed from every context (Window, DedicatedWorker,
		//    SharedWorker), supports the same postMessage / onmessage
		//    surface we need, and — critically — accepts transferable
		//    MessagePorts. SharedWorker would NOT work as a child here,
		//    because a SharedWorker scope is forbidden from constructing
		//    further SharedWorkers (https://crbug.com/1102827) which
		//    would block grandchild fan-out.
		childPort, spawnErr := spawnChildWorker(coord.workerURL, workerName)
		if spawnErr != nil {
			return nil, spawnErr
		}
		childClient := newPortCallClient(childPort)

		// 3b) create a private coord channel for this child and hand one
		//     end to the coordinator (via our own coord client) and the
		//     other end to the child as a transferable on the runVMRoutine
		//     message. This propagates coord access recursively without
		//     ever needing the child to construct a SharedWorker itself.
		mcCtor := js.Global().Get("MessageChannel")
		var childCoordPort js.Value
		var hasChildCoordPort bool
		if !mcCtor.IsUndefined() {
			ch := mcCtor.New()
			p1 := ch.Get("port1")
			p2 := ch.Get("port2")
			// transfer p1 to the coordinator (no reply expected)
			tr := js.Global().Get("Array").New(1)
			tr.SetIndex(0, p1)
			attachMsg := jsObject.New()
			attachMsg.Set("op", "coord.attach")
			attachMsg.Set("port", p1)
			coord.port.Call("postMessage", attachMsg, tr)
			childCoordPort = p2
			hasChildCoordPort = true
		}

		handle := &remoteRoutineHandle{
			id:     routineID,
			child:  childClient,
			parent: coord,
			doneCh: make(chan struct{}),
		}

		// forward streamed output from child → parent's page-facing port,
		// and bump the coordinator's monitor byte counter.
		childClient.subscribe(func(data js.Value) {
			if t := data.Get("type"); t.Type() == js.TypeString && t.String() == "output" {
				if g_vmHostPort.Truthy() {
					g_vmHostPort.Call("postMessage", data)
				}
			}
		})

		// 4) issue runVMRoutine asynchronously. The reply carries the routine's
		//    return value (encoded with vm.MarshalLive) or an error string.
		go func() {
			res, err := childClient.callTransfer(context.Background(), "runVMRoutine", func(m js.Value) {
				m.Set("bytecode", bytesToJS(parentBytecode))
				m.Set("fn", bytesToJS(fnBlob))
				m.Set("args", argsArr)
				m.Set("globals", globalsArr)
				m.Set("routineId", routineID)
				m.Set("workerName", workerName)
				m.Set("workerURL", coord.workerURL)
				if hasChildCoordPort {
					m.Set("coordPort", childCoordPort)
				}
			}, func() []js.Value {
				if hasChildCoordPort {
					return []js.Value{childCoordPort}
				}
				return nil
			}())
			if err != nil {
				handle.markDone(nil, err)
				_, _ = coord.call(context.Background(), "routine.done", func(m js.Value) {
					m.Set("routineId", routineID)
					m.Set("error", err.Error())
				})
				return
			}
			// res is { value: <Uint8Array>, error: <string|null> }
			if e := res.Get("error"); e.Type() == js.TypeString {
				handle.markDone(nil, errors.New(e.String()))
				_, _ = coord.call(context.Background(), "routine.done", func(m js.Value) {
					m.Set("routineId", routineID)
					m.Set("error", e.String())
				})
				return
			}
			var val vm.Object = vm.UndefinedValue
			if vb := res.Get("value"); vb.Truthy() {
				blob := toBytes(vb)
				if len(blob) > 0 {
					if v, err := vm.UnmarshalLive(blob); err == nil {
						val = v
					}
				}
			}
			handle.markDone(val, nil)
			_, _ = coord.call(context.Background(), "routine.done", func(m js.Value) {
				m.Set("routineId", routineID)
				m.Set("error", js.Null())
			})
		}()

		return handle, nil
	}
}

// ---------------------------------------------------------------------------
// installRemoteConcurrency — wire vm.Config for a vm-host
// ---------------------------------------------------------------------------

// installRemoteConcurrency populates Spawner + ChanFactory on cfg using the
// coordinator client owned by this vm-host. It must be called before
// runSource / runCompiled. Safe to call when myCoord is nil — does nothing.
func installRemoteConcurrency(cfg *vm.Config, parentBytecode []byte) {
	if myCoord == nil {
		return
	}
	tr := newRemoteChanTransport(myCoord)
	cfg.ChanFactory = makeChanFactory(myCoord, tr)
	cfg.Spawner = newSpawner(myCoord, parentBytecode)
}

// g_vmHostPort is the page-facing (or parent vm-host-facing) port of the
// current vm-host. Set in runOneVM / runOneVMRoutine. Used by the Spawner
// to forward child stdout chunks toward the page.
var g_vmHostPort js.Value

// ---------------------------------------------------------------------------
// runSourceRemote / runCompiledRemote
//
// Drop-in replacements for runSource / runCompiled that install Spawner and
// ChanFactory on the compiled program. The Spawner needs the program's
// serialised bytecode to ship to every child SharedWorker so OpConstant
// references inside the supplied CompiledFunction continue to resolve.
// ---------------------------------------------------------------------------

func runSourceRemote(ctx context.Context, source []byte, path string, args []string, stdin io.Reader, stdout io.Writer) error {
	if path == "" {
		path = "main.rumo"
	}
	snap := sharedFS.snapshot()
	cp := append([]byte(nil), source...)
	snap[path] = &fstest.MapFile{Data: cp}

	s := rumo.NewScript(snap, path)
	s.SetImports(rumo.Modules())
	s.SetPermissions(vm.UnrestrictedPermissions())
	prog, err := s.Compile()
	if err != nil {
		return err
	}
	prog.SetArgs(append([]string{path}, args...))
	if stdin != nil {
		prog.SetStdin(stdin)
	}
	if stdout != nil {
		prog.SetStdout(stdout)
	}
	if myCoord != nil {
		bcBlob, err := prog.Marshal()
		if err != nil {
			return err
		}
		installRemoteSpawnerOnProgram(prog, bcBlob)
	}
	return prog.RunContext(ctx)
}

func runCompiledRemote(ctx context.Context, data []byte, args []string, stdin io.Reader, stdout io.Writer) error {
	prog := &rumo.Program{}
	if err := prog.UnmarshalWithModules(data, rumo.Modules()); err != nil {
		return err
	}
	prog.SetArgs(append([]string{""}, args...))
	if stdin != nil {
		prog.SetStdin(stdin)
	}
	if stdout != nil {
		prog.SetStdout(stdout)
	}
	if myCoord != nil {
		// `data` IS the marshaled bytecode; pass it straight to the spawner.
		installRemoteSpawnerOnProgram(prog, data)
	}
	return prog.RunContext(ctx)
}

// installRemoteSpawnerOnProgram attaches Spawner + ChanFactory to a rumo.Program
// using myCoord as the coordinator client and bcBlob as the bytecode that
// children should ship for closure resolution.
func installRemoteSpawnerOnProgram(prog *rumo.Program, bcBlob []byte) {
	tr := newRemoteChanTransport(myCoord)
	prog.SetChanFactory(makeChanFactory(myCoord, tr))
	prog.SetSpawner(newSpawner(myCoord, bcBlob))
}

// ---------------------------------------------------------------------------
// runOneVMRoutine — vm-host receive side of the spawn protocol
//
// Triggered when the parent vm-host posts a `runVMRoutine` message. Decodes
// the bytecode + fn + args, installs Spawner + ChanFactory pointing at the
// same coordinator (so this child's own `go` and `chan` calls also fan out
// to fresh DedicatedWorkers), and runs vm.RunCompiled(fn, args).
// ---------------------------------------------------------------------------

func runOneVMRoutine(port js.Value, data js.Value, cancelOut *context.CancelFunc, cancelMu *sync.Mutex) {
	id := data.Get("id")
	g_vmHostPort = port

	// workerURL is needed to spawn nested children.
	if v := data.Get("workerURL"); v.Type() == js.TypeString {
		myWorkerURL = v.String()
	} else if myWorkerURL == "" {
		myWorkerURL = "./worker.js"
	}
	// Preferred: parent vm-host transferred a private coordPort.
	if myCoord == nil {
		if cp := data.Get("coordPort"); cp.Truthy() {
			myCoord = newCoordFromPort(cp, myWorkerURL)
		}
	}
	// Fallback to constructing one ourselves (only works in some scopes).
	if myCoord == nil {
		if c, err := openCoordClient(myWorkerURL); err == nil {
			myCoord = c
		}
	}
	// remember our own routine id so the nested Spawner can record
	// parent->child edges for `go fn()` calls inside this routine.
	if v := data.Get("routineId"); v.Type() == js.TypeNumber {
		myRoutineID = int64(v.Int())
	}

	// Decode bytecode → Program (gives us the constants pool fn refers to).
	bcBlob := toBytes(data.Get("bytecode"))
	prog := &rumo.Program{}
	if err := prog.UnmarshalWithModules(bcBlob, rumo.Modules()); err != nil {
		replyVMRoutine(port, id, nil, err)
		return
	}

	// Decode fn + args.
	fnBlob := toBytes(data.Get("fn"))
	fnObj, err := vm.UnmarshalLive(fnBlob)
	if err != nil {
		replyVMRoutine(port, id, nil, err)
		return
	}
	fn, ok := fnObj.(*vm.CompiledFunction)
	if !ok {
		replyVMRoutine(port, id, nil, errors.New("runVMRoutine: fn is not a CompiledFunction"))
		return
	}

	// any *Chan inside fn's free vars or args needs binding
	tr := newRemoteChanTransport(myCoord)
	vm.ResolveChans(fn, nil, tr)

	argsArr := data.Get("args")
	n := argsArr.Length()
	scriptArgs := make([]vm.Object, n)
	for i := 0; i < n; i++ {
		blob := toBytes(argsArr.Index(i))
		o, err := vm.UnmarshalLive(blob)
		if err != nil {
			replyVMRoutine(port, id, nil, err)
			return
		}
		vm.ResolveChans(o, nil, tr)
		scriptArgs[i] = o
	}

	// Wire VM with our Spawner + ChanFactory so this child also fans out.
	cfg := vm.UnlimitedConfig()
	cfg.Permissions = vm.UnrestrictedPermissions()
	cfg.ChanFactory = makeChanFactory(myCoord, tr)
	cfg.Spawner = newSpawner(myCoord, bcBlob)

	bc := prog.Bytecode()
	globals := make([]vm.Object, cfg.GlobalsSize)
	// If the parent shipped its globals snapshot, decode them into the
	// child VM's globals slice (after rebinding modules / builtins). This
	// makes OpGetGlobal references (e.g. imported `fmt` / `times`) resolve
	// correctly inside `go fn()` bodies on the DedicatedWorker path.
	if g := data.Get("globals"); g.InstanceOf(jsArray) {
		mods := rumo.Modules()
		n := g.Length()
		if n > len(globals) {
			n = len(globals)
		}
		for i := 0; i < n; i++ {
			gv := g.Index(i)
			if gv.IsUndefined() || gv.IsNull() {
				continue
			}
			blob := toBytes(gv)
			if len(blob) == 0 {
				continue
			}
			obj, uErr := vm.UnmarshalLive(blob)
			if uErr != nil {
				// One bad slot must not poison the whole routine —
				// the body may never read it. Leave Undefined and
				// keep going.
				continue
			}
			fixed, fErr := vm.FixDecodedObject(obj, mods)
			if fErr != nil {
				// e.g. *Map of anonymous *BuiltinFunction values
				// (a routine handle from the parent — never used
				// by the child). Leave the slot as Undefined.
				continue
			}
			vm.ResolveChans(fixed, nil, tr)
			globals[i] = fixed
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelMu.Lock()
	*cancelOut = cancel
	cancelMu.Unlock()

	v := vm.NewVM(ctx, bc, globals, cfg)
	v.Out = &vmHostStream{port: port}
	v.Args = []string{fn.TypeName()}

	val, runErr := v.RunCompiled(fn, scriptArgs...)
	replyVMRoutine(port, id, val, runErr)
}

// replyVMRoutine emits the standard {id, result|error} envelope back to the
// parent vm-host.
func replyVMRoutine(port, id js.Value, val vm.Object, err error) {
	out := jsObject.New()
	if err != nil {
		out.Set("error", err.Error())
		out.Set("value", js.Null())
	} else {
		out.Set("error", js.Null())
		var blob []byte
		if val != nil {
			b, mErr := vm.MarshalLive(val)
			if mErr == nil {
				blob = b
			}
		}
		if blob == nil {
			out.Set("value", js.Null())
		} else {
			out.Set("value", bytesToJS(blob))
		}
	}
	msg := jsObject.New()
	msg.Set("id", id)
	msg.Set("result", out)
	port.Call("postMessage", msg)
}

