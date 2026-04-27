//go:build js && wasm

// main_js.go is the js/wasm entrypoint. It exposes a JavaScript API that
// mirrors the github.com/malivvan/rumo Go package.
//
// Architecture
//
//	page tab(s) ──connect──▶ rumo-coordinator (SharedWorker)
//	                            │
//	                            ├── shared in-mem FS
//	                            ├── module registry
//	                            ├── monitor / routine registry
//	                            └── chan registry
//
//	page tab    ──new Worker─▶ rumo-vm-<id>   (DedicatedWorker, one per VM run)
//	                            │
//	                            └─ on `go fn()` ─▶ rumo-vm-<childId>
//	                                                (DedicatedWorker child)
//
// The coordinator is a SharedWorker because its state is shared across
// every page tab on the same origin. Each VM run, however, lives in its
// own DedicatedWorker so that every running rumo VM appears as a separate
// worker in DevTools and as a distinct row in the live monitor.
// DedicatedWorker is also used for `go fn()` children because SharedWorker
// scopes are forbidden from constructing further SharedWorkers
// (https://crbug.com/1102827).
//
// The same API is also registered on the global object directly, so the
// binary also works when loaded into the main thread or a plain Worker for
// testing or single-context scripting (the "standalone" role).
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"syscall/js"
	"testing/fstest"
	"time"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/vm"
)

// ---------------------------------------------------------------------------
// shared filesystem
// ---------------------------------------------------------------------------

// fsStore is the in-memory filesystem owned by the coordinator and shared
// by every VM running on the page. Scripts launched through `rumo.run` see
// this map merged with their entrypoint source so that imports resolve
// against the same files the page placed via `rumo.fs.put`.
type fsStore struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func newFSStore() *fsStore { return &fsStore{files: map[string][]byte{}} }

func (s *fsStore) snapshot() fstest.MapFS {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := make(fstest.MapFS, len(s.files))
	for k, v := range s.files {
		c := make([]byte, len(v))
		copy(c, v)
		m[k] = &fstest.MapFile{Data: c}
	}
	return m
}

func (s *fsStore) put(path string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.files[path] = cp
}

func (s *fsStore) get(path string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.files[path]
	if !ok {
		return nil, false
	}
	cp := make([]byte, len(d))
	copy(cp, d)
	return cp, true
}

func (s *fsStore) list() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.files))
	for k := range s.files {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (s *fsStore) del(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.files[path]; !ok {
		return false
	}
	delete(s.files, path)
	return true
}

func (s *fsStore) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = map[string][]byte{}
}

var sharedFS = newFSStore()

// ---------------------------------------------------------------------------
// routine registry — JS-level routines
// ---------------------------------------------------------------------------

// At the script level, `routine.start(fn)` keeps using Go goroutines through
// the existing routinevm implementation. At the JS level, `rumo.spawn(src)`
// creates an independent VM run and tracks it here so the page can `cancel`,
// `wait`, `write` to its stdin, and stream its stdout.
// routine state codes used in the live monitor.
const (
	routineRunning = int32(0)
	routineDone    = int32(1)
	routineError   = int32(2)
	routineCancel  = int32(3)
)

type routine struct {
	id     int64
	ctx    context.Context
	cancel context.CancelFunc

	stdinR *io.PipeReader
	stdinW *io.PipeWriter

	outMu  sync.Mutex
	outBuf bytes.Buffer
	// onChunk is invoked for every stdout chunk. Callers attach it through
	// spawn options or, for coordinator port clients, the bridge plugs in
	// a chunk forwarder that posts {type:"output"} messages over the port.
	onChunk func(string)

	doneCh chan struct{}
	doneI  atomic.Int32
	err    error

	// monitoring fields exposed via rumo.routines()
	name       string
	kind       string // "run" | "spawn" | "runCompiled" | "remote" | "go"
	workerName string // DedicatedWorker name hosting this VM (empty for in-process routines)
	parentID   int64  // 0 == top-level (no parent); otherwise the spawning routine's id
	startedAt  time.Time
	endedAt    atomic.Int64 // unix nano; 0 while running
	bytesOut   atomic.Int64
	state      atomic.Int32 // routineRunning / routineDone / routineError / routineCancel
	errStr     atomic.Pointer[string]
}

// Write makes the routine an io.Writer for VM stdout.
func (r *routine) Write(p []byte) (int, error) {
	r.bytesOut.Add(int64(len(p)))
	r.outMu.Lock()
	r.outBuf.Write(p)
	cb := r.onChunk
	r.outMu.Unlock()
	if cb != nil {
		// guard against panics in the JS callback
		func() {
			defer func() { _ = recover() }()
			cb(string(p))
		}()
	}
	return len(p), nil
}

func (r *routine) finish(err error) {
	r.err = err
	if err != nil {
		s := err.Error()
		r.errStr.Store(&s)
		if r.ctx.Err() != nil {
			r.state.Store(routineCancel)
		} else {
			r.state.Store(routineError)
		}
	} else {
		r.state.Store(routineDone)
	}
	r.endedAt.Store(time.Now().UnixNano())
	if r.stdinW != nil {
		_ = r.stdinW.Close()
	}
	if r.doneI.CompareAndSwap(0, 1) {
		close(r.doneCh)
	}
	monitor.emitDone(r)
}

// stateString returns a stable human label for the JS monitor table.
func (r *routine) stateString() string {
	switch r.state.Load() {
	case routineDone:
		return "done"
	case routineError:
		return "error"
	case routineCancel:
		return "canceled"
	default:
		return "running"
	}
}

// snapshot returns a JS-friendly map of the routine's current state.
func (r *routine) snapshot(now time.Time) js.Value {
	obj := jsObject.New()
	obj.Set("id", r.id)
	obj.Set("name", r.name)
	obj.Set("kind", r.kind)
	obj.Set("workerName", r.workerName)
	obj.Set("parentId", r.parentID)
	obj.Set("state", r.stateString())
	obj.Set("bytesOut", r.bytesOut.Load())
	obj.Set("startedAtMs", r.startedAt.UnixMilli())
	if e := r.endedAt.Load(); e != 0 {
		ended := time.Unix(0, e)
		obj.Set("endedAtMs", ended.UnixMilli())
		obj.Set("durationMs", ended.Sub(r.startedAt).Milliseconds())
	} else {
		obj.Set("endedAtMs", js.Null())
		obj.Set("durationMs", now.Sub(r.startedAt).Milliseconds())
	}
	if e := r.errStr.Load(); e != nil {
		obj.Set("error", *e)
	} else {
		obj.Set("error", js.Null())
	}
	return obj
}

func (r *routine) waitDone(timeout time.Duration) bool {
	if r.doneI.Load() == 1 {
		return true
	}
	if timeout < 0 {
		<-r.doneCh
		return true
	}
	select {
	case <-r.doneCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

type routineRegistry struct {
	mu    sync.Mutex
	next  atomic.Int64
	items map[int64]*routine
}

func (r *routineRegistry) add(rt *routine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.items == nil {
		r.items = make(map[int64]*routine)
	}
	r.items[rt.id] = rt
}

func (r *routineRegistry) get(id int64) *routine {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.items[id]
}

func (r *routineRegistry) del(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.delLocked(id)
}

// delLocked removes the routine and re-parents any surviving children to the
// removed routine's own parent. This keeps the routine tree on the page-side
// monitor connected when an intermediate `go` routine finishes (and gets
// auto-pruned 2s later) while its own `go` child is still running. Without
// reparenting, the grandchild's parentId would dangle and renderRoutines
// would promote it to a top-level root, visually breaking the tree.
//
// Caller must hold r.mu.
func (r *routineRegistry) delLocked(id int64) {
	rt, ok := r.items[id]
	if !ok {
		return
	}
	grandparent := rt.parentID
	for _, child := range r.items {
		if child.parentID == id {
			child.parentID = grandparent
		}
	}
	delete(r.items, id)
}

func (r *routineRegistry) snapshot() js.Value {
	r.mu.Lock()
	all := make([]*routine, 0, len(r.items))
	for _, rt := range r.items {
		all = append(all, rt)
	}
	r.mu.Unlock()
	sort.Slice(all, func(i, j int) bool { return all[i].id < all[j].id })
	now := time.Now()
	arr := jsArray.New()
	for i, rt := range all {
		arr.SetIndex(i, rt.snapshot(now))
	}
	return arr
}

func (r *routineRegistry) prune() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	// Collect ids first so we can iterate ids in stable order and use the
	// reparenting delLocked helper without mutating during range.
	ids := make([]int64, 0, len(r.items))
	for id, rt := range r.items {
		if rt.state.Load() != routineRunning {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if _, ok := r.items[id]; !ok {
			continue
		}
		r.delLocked(id)
		n++
	}
	return n
}

var routines = &routineRegistry{}

// ---------------------------------------------------------------------------
// monitor bus — pushes lifecycle events to subscribed JS callbacks and ports
// ---------------------------------------------------------------------------

// monitorBus delivers `routine:spawned` / `routine:done` lifecycle messages
// to subscribers so that page UIs can react instantly without polling.
// Polling via rumo.routines() still works for byte/age stats.
type monitorBus struct {
	mu    sync.Mutex
	ports []js.Value
	cbs   []js.Value
}

func (b *monitorBus) addPort(p js.Value) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ports = append(b.ports, p)
}

func (b *monitorBus) addCB(cb js.Value) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cbs = append(b.cbs, cb)
}

func (b *monitorBus) emit(msg js.Value) {
	b.mu.Lock()
	ports := append([]js.Value{}, b.ports...)
	cbs := append([]js.Value{}, b.cbs...)
	b.mu.Unlock()
	for _, p := range ports {
		func() {
			defer func() { _ = recover() }()
			p.Call("postMessage", msg)
		}()
	}
	for _, cb := range cbs {
		func() {
			defer func() { _ = recover() }()
			cb.Invoke(msg)
		}()
	}
}

func (b *monitorBus) emitSpawned(rt *routine) {
	msg := jsObject.New()
	msg.Set("type", "routine:spawned")
	msg.Set("routine", rt.snapshot(time.Now()))
	b.emit(msg)
}

func (b *monitorBus) emitDone(rt *routine) {
	msg := jsObject.New()
	msg.Set("type", "routine:done")
	msg.Set("routine", rt.snapshot(time.Now()))
	b.emit(msg)
}

var monitor = &monitorBus{}

// ---------------------------------------------------------------------------
// js helpers
// ---------------------------------------------------------------------------

var (
	jsObject     = js.Global().Get("Object")
	jsUint8Array = js.Global().Get("Uint8Array")
	jsArray      = js.Global().Get("Array")
	jsError      = js.Global().Get("Error")
	jsPromise    = js.Global().Get("Promise")
)

func toBytes(v js.Value) []byte {
	if v.IsUndefined() || v.IsNull() {
		return nil
	}
	if v.InstanceOf(jsUint8Array) {
		b := make([]byte, v.Length())
		js.CopyBytesToGo(b, v)
		return b
	}
	if v.Type() == js.TypeString {
		return []byte(v.String())
	}
	return nil
}

func bytesToJS(b []byte) js.Value {
	arr := jsUint8Array.New(len(b))
	js.CopyBytesToJS(arr, b)
	return arr
}

func newError(format string, args ...any) js.Value {
	return jsError.New(fmt.Sprintf(format, args...))
}

// newPromise spawns a goroutine that performs work and resolves/rejects.
// resolve/reject capture js.Value (or any js-convertible) and forward it.
func newPromise(work func(resolve, reject func(any))) js.Value {
	executor := js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[1]
		go func() {
			defer func() {
				if perr := recover(); perr != nil {
					reject.Invoke(newError("rumo: panic: %v", perr))
				}
			}()
			work(
				func(v any) { resolve.Invoke(v) },
				func(v any) { reject.Invoke(v) },
			)
		}()
		return nil
	})
	// note: executor is intentionally not Released — the Promise constructor
	// invokes it once and we let the GC handle cleanup along with the func.
	p := jsPromise.New(executor)
	executor.Release() // safe: New() has already called the executor synchronously
	return p
}

func optString(v js.Value, key string) string {
	if v.IsUndefined() || v.IsNull() {
		return ""
	}
	w := v.Get(key)
	if w.Type() != js.TypeString {
		return ""
	}
	return w.String()
}

func optStringArray(v js.Value, key string) []string {
	if v.IsUndefined() || v.IsNull() {
		return nil
	}
	w := v.Get(key)
	if !w.InstanceOf(jsArray) {
		return nil
	}
	n := w.Length()
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = w.Index(i).String()
	}
	return out
}

func optFunc(v js.Value, key string) js.Value {
	if v.IsUndefined() || v.IsNull() {
		return js.Undefined()
	}
	w := v.Get(key)
	if w.Type() != js.TypeFunction {
		return js.Undefined()
	}
	return w
}

// ---------------------------------------------------------------------------
// rumo runner
// ---------------------------------------------------------------------------

// runSource compiles `source` against the shared FS and runs it.
func runSource(ctx context.Context, source []byte, path string, args []string, stdin io.Reader, stdout io.Writer) error {
	if path == "" {
		path = "main.rumo"
	}
	snap := sharedFS.snapshot()
	// place (or override) the entrypoint at `path`
	cp := make([]byte, len(source))
	copy(cp, source)
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
	return prog.RunContext(ctx)
}

// runCompiled deserializes a precompiled bytecode blob and runs it.
func runCompiled(ctx context.Context, data []byte, args []string, stdin io.Reader, stdout io.Writer) error {
	p := &rumo.Program{}
	if err := p.UnmarshalWithModules(data, rumo.Modules()); err != nil {
		return err
	}
	p.SetArgs(append([]string{""}, args...))
	if stdin != nil {
		p.SetStdin(stdin)
	}
	if stdout != nil {
		p.SetStdout(stdout)
	}
	return p.RunContext(ctx)
}

// compileSource compiles `source` against the shared FS and returns the
// marshaled bytecode blob.
func compileSource(source []byte, path string) ([]byte, error) {
	if path == "" {
		path = "main.rumo"
	}
	snap := sharedFS.snapshot()
	cp := make([]byte, len(source))
	copy(cp, source)
	snap[path] = &fstest.MapFile{Data: cp}

	s := rumo.NewScript(snap, path)
	s.SetImports(rumo.Modules())
	s.SetPermissions(vm.UnrestrictedPermissions())
	prog, err := s.Compile()
	if err != nil {
		return nil, err
	}
	return prog.Marshal()
}

// newRoutine allocates a routine, assigns an ID, and registers it. It does
// NOT start the script; callers set rt.onChunk first and then call launch.
// `kind` is one of "run", "runCompiled", or "spawn"; `name` is the script
// path or label shown in the monitor.
func newRoutine(kind, name string) *routine {
	rt := allocRoutine(kind, name)
	monitor.emitSpawned(rt)
	return rt
}

// allocRoutine is the no-emit variant. Callers that need to set additional
// fields (e.g. parentID, workerName) before observers see the routine should
// allocate, mutate, then emit themselves via monitor.emitSpawned.
func allocRoutine(kind, name string) *routine {
	ctx, cancel := context.WithCancel(context.Background())
	pr, pw := io.Pipe()
	rt := &routine{
		ctx:       ctx,
		cancel:    cancel,
		stdinR:    pr,
		stdinW:    pw,
		doneCh:    make(chan struct{}),
		kind:      kind,
		startedAt: time.Now(),
	}
	rt.state.Store(routineRunning)
	rt.id = routines.next.Add(1)
	if name == "" {
		name = fmt.Sprintf("%s_%d.rumo", kind, rt.id)
	}
	rt.name = name
	routines.add(rt)
	return rt
}

// launchSource spawns the goroutine that compiles and runs `source`. Callers
// must set rt.onChunk before calling this so the first chunk doesn't race
// against the assignment.
func (rt *routine) launchSource(source []byte, args []string, stdinPreload string) {
	if stdinPreload != "" {
		go func() { _, _ = io.Copy(rt.stdinW, bytes.NewReader([]byte(stdinPreload))) }()
	}
	go func() {
		err := runSource(rt.ctx, source, rt.name, args, rt.stdinR, rt)
		rt.finish(err)
	}()
}

// launchCompiled is the precompiled-bytecode variant of launchSource.
func (rt *routine) launchCompiled(data []byte, args []string, stdinPreload string) {
	if stdinPreload != "" {
		go func() { _, _ = io.Copy(rt.stdinW, bytes.NewReader([]byte(stdinPreload))) }()
	}
	go func() {
		err := runCompiled(rt.ctx, data, args, rt.stdinR, rt)
		rt.finish(err)
	}()
}

// ---------------------------------------------------------------------------
// JS API: top-level methods
// ---------------------------------------------------------------------------

func jsVersion(this js.Value, args []js.Value) any { return rumo.Version() }
func jsCommit(this js.Value, args []js.Value) any  { return rumo.Commit() }

func jsModules(this js.Value, args []js.Value) any {
	names := rumo.AllModuleNames()
	sort.Strings(names)
	out := make([]any, len(names))
	for i, n := range names {
		out[i] = n
	}
	return js.ValueOf(out)
}

func jsExports(this js.Value, args []js.Value) any {
	exports := rumo.Exports()
	obj := jsObject.New()
	for modName, members := range exports {
		arr := jsArray.New()
		i := 0
		// stable order
		names := make([]string, 0, len(members))
		for k := range members {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			arr.SetIndex(i, k)
			i++
		}
		obj.Set(modName, arr)
	}
	return obj
}

func jsCompile(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsRejected(newError("rumo.compile: missing source argument"))
	}
	src := toBytes(args[0])
	path := ""
	if len(args) >= 2 && args[1].Type() == js.TypeString {
		path = args[1].String()
	}
	return newPromise(func(resolve, reject func(any)) {
		b, err := compileSource(src, path)
		if err != nil {
			reject(newError("%s", err.Error()))
			return
		}
		resolve(bytesToJS(b))
	})
}

// runViaRoutine creates and launches a routine for `rumo.run` /
// `rumo.runCompiled`, returning a Promise that resolves with the
// {output, error} envelope when the routine finishes. The routine is visible
// in the monitor for its lifetime and pruned by the caller via `rumo.routines.prune()`
// or removed automatically once it completes.
func runViaRoutine(rt *routine, source []byte, compiled []byte, scriptArgs []string, stdinPreload string) js.Value {
	return newPromise(func(resolve, reject func(any)) {
		if compiled != nil {
			rt.launchCompiled(compiled, scriptArgs, stdinPreload)
		} else {
			rt.launchSource(source, scriptArgs, stdinPreload)
		}
		rt.waitDone(-1)
		rt.outMu.Lock()
		out := rt.outBuf.String()
		rt.outMu.Unlock()
		res := jsObject.New()
		res.Set("output", out)
		if rt.err != nil {
			res.Set("error", rt.err.Error())
		} else {
			res.Set("error", js.Null())
		}
		// short-lived runs disappear from the monitor automatically; spawn
		// keeps the entry until the page calls .close().
		go func() {
			time.Sleep(2 * time.Second)
			routines.del(rt.id)
		}()
		resolve(res)
	})
}

func jsRun(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsRejected(newError("rumo.run: missing source argument"))
	}
	src := toBytes(args[0])
	var opts js.Value = js.Undefined()
	if len(args) >= 2 {
		opts = args[1]
	}
	path := optString(opts, "path")
	scriptArgs := optStringArray(opts, "args")
	stdinStr := optString(opts, "stdin")
	onOutput := optFunc(opts, "onOutput")

	rt := newRoutine("run", path)
	if onOutput.Type() == js.TypeFunction {
		f := onOutput
		rt.onChunk = func(s string) { f.Invoke(s) }
	}
	return runViaRoutine(rt, src, nil, scriptArgs, stdinStr)
}

func jsRunCompiled(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsRejected(newError("rumo.runCompiled: missing bytecode argument"))
	}
	data := toBytes(args[0])
	var opts js.Value = js.Undefined()
	if len(args) >= 2 {
		opts = args[1]
	}
	scriptArgs := optStringArray(opts, "args")
	stdinStr := optString(opts, "stdin")
	onOutput := optFunc(opts, "onOutput")
	name := optString(opts, "name")
	if name == "" {
		name = "compiled.bin"
	}

	rt := newRoutine("runCompiled", name)
	if onOutput.Type() == js.TypeFunction {
		f := onOutput
		rt.onChunk = func(s string) { f.Invoke(s) }
	}
	return runViaRoutine(rt, nil, data, scriptArgs, stdinStr)
}

// jsSpawn: rumo.spawn(source, opts?) -> handle object
//
//	handle.id          number
//	handle.cancel()    void
//	handle.wait(secs?) Promise<bool>     resolves true if completed within timeout
//	handle.result()    Promise<{value, error, output}>
//	handle.write(s)    void              push data into the script's stdin
//	handle.close()     void              close stdin and free resources after wait
func jsSpawn(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return newError("rumo.spawn: missing source argument")
	}
	src := toBytes(args[0])
	var opts js.Value = js.Undefined()
	if len(args) >= 2 {
		opts = args[1]
	}
	onOutput := optFunc(opts, "onOutput")
	path := optString(opts, "path")

	rt := newRoutine("spawn", path)
	if onOutput.Type() == js.TypeFunction {
		f := onOutput
		rt.onChunk = func(s string) { f.Invoke(s) }
	}
	rt.launchSource(src, optStringArray(opts, "args"), optString(opts, "stdin"))

	handle := jsObject.New()
	handle.Set("id", rt.id)
	handle.Set("name", rt.name)

	handle.Set("cancel", js.FuncOf(func(this js.Value, _ []js.Value) any {
		rt.cancel()
		return nil
	}))

	handle.Set("write", js.FuncOf(func(this js.Value, a []js.Value) any {
		if len(a) == 0 {
			return nil
		}
		b := toBytes(a[0])
		if len(b) == 0 {
			return nil
		}
		go func() { _, _ = rt.stdinW.Write(b) }()
		return nil
	}))

	handle.Set("close", js.FuncOf(func(this js.Value, _ []js.Value) any {
		_ = rt.stdinW.Close()
		routines.del(rt.id)
		return nil
	}))

	handle.Set("wait", js.FuncOf(func(this js.Value, a []js.Value) any {
		secs := -1.0
		if len(a) > 0 && a[0].Type() == js.TypeNumber {
			secs = a[0].Float()
		}
		return newPromise(func(resolve, reject func(any)) {
			d := time.Duration(-1)
			if secs >= 0 {
				d = time.Duration(secs * float64(time.Second))
			}
			ok := rt.waitDone(d)
			if ok {
				resolve(true)
			} else {
				resolve(false)
			}
		})
	}))

	handle.Set("result", js.FuncOf(func(this js.Value, _ []js.Value) any {
		return newPromise(func(resolve, reject func(any)) {
			rt.waitDone(-1)
			rt.outMu.Lock()
			out := rt.outBuf.String()
			rt.outMu.Unlock()
			res := jsObject.New()
			res.Set("output", out)
			if rt.err != nil {
				res.Set("error", rt.err.Error())
			} else {
				res.Set("error", js.Null())
			}
			resolve(res)
		})
	}))

	return handle
}

// jsRoutines returns the current snapshot of all known routines as an
// array suitable for rendering in a live monitor.
func jsRoutines(this js.Value, args []js.Value) any {
	return routines.snapshot()
}

// jsRoutinesPrune drops every non-running routine from the registry and
// returns the count removed.
func jsRoutinesPrune(this js.Value, args []js.Value) any {
	return routines.prune()
}

// jsMonitorSubscribe registers a JS callback to receive
// {type:"routine:spawned"|"routine:done"} events on the calling page.
func jsMonitorSubscribe(this js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeFunction {
		return newError("rumo.subscribe: callback function required")
	}
	monitor.addCB(args[0])
	return js.Undefined()
}

func jsRejected(err js.Value) js.Value {
	return jsPromise.Call("reject", err)
}

// ---------------------------------------------------------------------------
// JS API: rumo.fs
// ---------------------------------------------------------------------------

func jsFsPut(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return newError("rumo.fs.put: usage put(path, content)")
	}
	if args[0].Type() != js.TypeString {
		return newError("rumo.fs.put: path must be a string")
	}
	sharedFS.put(args[0].String(), toBytes(args[1]))
	return js.Undefined()
}

func jsFsGet(this js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return js.Null()
	}
	d, ok := sharedFS.get(args[0].String())
	if !ok {
		return js.Null()
	}
	return bytesToJS(d)
}

func jsFsList(this js.Value, args []js.Value) any {
	names := sharedFS.list()
	out := make([]any, len(names))
	for i, n := range names {
		out[i] = n
	}
	return js.ValueOf(out)
}

func jsFsDelete(this js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return false
	}
	return sharedFS.del(args[0].String())
}

func jsFsClear(this js.Value, args []js.Value) any {
	sharedFS.clear()
	return js.Undefined()
}

// ---------------------------------------------------------------------------
// API installer + worker bridges
// ---------------------------------------------------------------------------

func installAPI(target js.Value) {
	api := jsObject.New()
	api.Set("version", js.FuncOf(jsVersion))
	api.Set("commit", js.FuncOf(jsCommit))
	api.Set("modules", js.FuncOf(jsModules))
	api.Set("exports", js.FuncOf(jsExports))
	api.Set("compile", js.FuncOf(jsCompile))
	api.Set("run", js.FuncOf(jsRun))
	api.Set("runCompiled", js.FuncOf(jsRunCompiled))
	api.Set("spawn", js.FuncOf(jsSpawn))
	api.Set("routines", js.FuncOf(jsRoutines))
	api.Set("pruneRoutines", js.FuncOf(jsRoutinesPrune))
	api.Set("subscribe", js.FuncOf(jsMonitorSubscribe))

	fs := jsObject.New()
	fs.Set("put", js.FuncOf(jsFsPut))
	fs.Set("get", js.FuncOf(jsFsGet))
	fs.Set("list", js.FuncOf(jsFsList))
	fs.Set("delete", js.FuncOf(jsFsDelete))
	fs.Set("clear", js.FuncOf(jsFsClear))
	api.Set("fs", fs)

	// expose a "ready" sentinel so JS can poll for runtime availability
	api.Set("ready", true)

	target.Set("rumo", api)
}

// inSharedWorker reports whether the global scope of this WASM instance is a
// SharedWorkerGlobalScope. Only the coordinator role runs in a SharedWorker.
func inSharedWorker() bool {
	self := js.Global()
	c := self.Get("constructor")
	if c.IsUndefined() {
		return false
	}
	name := c.Get("name")
	if name.Type() != js.TypeString {
		return false
	}
	return name.String() == "SharedWorkerGlobalScope"
}

// inDedicatedWorker reports whether the global scope of this WASM instance
// is a DedicatedWorkerGlobalScope. Both top-level vm-hosts (one per
// rumo.run / runCompiled / spawn call) and per-`go fn()` child workers
// run in this mode.
func inDedicatedWorker() bool {
	self := js.Global()
	c := self.Get("constructor")
	if c.IsUndefined() {
		return false
	}
	name := c.Get("name")
	if name.Type() != js.TypeString {
		return false
	}
	return name.String() == "DedicatedWorkerGlobalScope"
}

// workerSelfName returns the SharedWorker `self.name` the current instance
// was created with, or "" if this WASM instance isn't running in a
// SharedWorker scope. Only the coordinator uses this (its name is
// "rumo-coordinator").
func workerSelfName() string {
	self := js.Global()
	if !inSharedWorker() {
		return ""
	}
	if n := self.Get("name"); n.Type() == js.TypeString {
		return n.String()
	}
	return ""
}

// installVMHostBridgeDedicated wires self.onmessage for a vm-host running
// in a DedicatedWorker. Every VM run lives in its own such worker — both
// top-level page-launched VMs and per-`go fn()` children. The bridge
// speaks the same runVM / runVMCompiled / runVMRoutine / cancel protocol
// using `self` itself as the message port: DedicatedWorkers receive via
// self.onmessage and reply via self.postMessage(...) without per-connection
// ports.
//
// Messages from page (or parent vm-host) → vm-host:
//
//	{ id, op:"runVM",         source, args, stdin, fs, coordPort }
//	{ id, op:"runVMCompiled", bytecode, args, stdin, coordPort }
//	{ id, op:"runVMRoutine",  bytecode, fn, args, globals, coordPort }
//	{ id, op:"cancel" }
//
// Replies (matched on id):
//
//	{ id, result: { output, bytes, error } }            // runVM / runVMCompiled
//	{ id, result: { value, error } }                    // runVMRoutine
//
// Streamed output is forwarded as `{type:"output", chunk, bytes}` messages
// while the VM is running.
func installVMHostBridgeDedicated() {
	if !inDedicatedWorker() {
		return
	}
	self := js.Global()
	var (
		running atomic.Bool
		cancel  context.CancelFunc
		cancelM sync.Mutex
	)

	self.Set("onmessage", js.FuncOf(func(_ js.Value, ma []js.Value) any {
		if len(ma) == 0 {
			return nil
		}
		data := ma[0].Get("data")
		op := data.Get("op")
		if op.Type() != js.TypeString {
			return nil
		}
		switch op.String() {
		case "runVM", "runVMCompiled":
			if !running.CompareAndSwap(false, true) {
				portError(self, data.Get("id"), "vm-host: already running")
				return nil
			}
			go runOneVM(self, data, op.String() == "runVMCompiled", &cancel, &cancelM)
		case "runVMRoutine":
			if !running.CompareAndSwap(false, true) {
				portError(self, data.Get("id"), "vm-host: already running")
				return nil
			}
			go runOneVMRoutine(self, data, &cancel, &cancelM)
		case "cancel":
			cancelM.Lock()
			if cancel != nil {
				cancel()
			}
			cancelM.Unlock()
		}
		return nil
	}))

	hello := jsObject.New()
	hello.Set("type", "ready")
	hello.Set("role", "vm-host")
	hello.Set("version", rumo.Version())
	self.Call("postMessage", hello)
}

// runOneVM executes a single VM in this DedicatedWorker. Output is streamed
// back as `{type:"output",chunk}` messages and the final `{id,result}`
// envelope carries the buffered output and error string (or null).
func runOneVM(port js.Value, data js.Value, compiled bool, cancelOut *context.CancelFunc, cancelMu *sync.Mutex) {
	id := data.Get("id")

	// stage shared-FS contents (sent by the page) into the local sharedFS so
	// the compiler can resolve imports.
	if fs := data.Get("fs"); !fs.IsUndefined() && !fs.IsNull() {
		keys := js.Global().Get("Object").Call("keys", fs)
		for i := 0; i < keys.Length(); i++ {
			path := keys.Index(i).String()
			b := toBytes(fs.Get(path))
			sharedFS.put(path, b)
		}
	}

	// Establish coordinator client + record the page-facing port so that the
	// Spawner can forward child output upstream and that chan ops can hop
	// to the coordinator.
	g_vmHostPort = port
	if v := data.Get("workerURL"); v.Type() == js.TypeString {
		myWorkerURL = v.String()
	} else if myWorkerURL == "" {
		myWorkerURL = "./worker.js"
	}
	// Preferred: the page (or parent vm-host) transferred a private
	// MessagePort to us via the runVM message — use it. This is the only
	// path that always works, since DedicatedWorker scopes can't always
	// construct SharedWorkers directly in every browser.
	if myCoord == nil {
		if cp := data.Get("coordPort"); cp.Truthy() {
			myCoord = newCoordFromPort(cp, myWorkerURL)
		}
	}
	// Fallback: try opening a SharedWorker connection ourselves. This works
	// from DedicatedWorker contexts in most browsers; if it fails, `go fn()`
	// would be forced back to local goroutines (no monitor visibility).
	if myCoord == nil {
		if c, err := openCoordClient(myWorkerURL); err == nil {
			myCoord = c
		}
	}
	// remember our own routine id so the Spawner can record parent->child
	// edges via routine.allocate (see newSpawner).
	if v := data.Get("routineId"); v.Type() == js.TypeNumber {
		myRoutineID = int64(v.Int())
	}

	scriptArgs := jsValueStringArray(data.Get("args"))
	stdinStr := ""
	if v := data.Get("stdin"); v.Type() == js.TypeString {
		stdinStr = v.String()
	}
	path := ""
	if v := data.Get("path"); v.Type() == js.TypeString {
		path = v.String()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelMu.Lock()
	*cancelOut = cancel
	cancelMu.Unlock()

	var buf bytes.Buffer
	streamW := &vmHostStream{port: port}
	w := io.MultiWriter(&buf, streamW)
	var r io.Reader
	if stdinStr != "" {
		r = bytes.NewReader([]byte(stdinStr))
	}

	var err error
	if compiled {
		err = runCompiledRemote(ctx, toBytes(data.Get("bytecode")), scriptArgs, r, w)
	} else {
		err = runSourceRemote(ctx, toBytes(data.Get("source")), path, scriptArgs, r, w)
	}

	result := jsObject.New()
	result.Set("output", buf.String())
	result.Set("bytes", buf.Len())
	if err != nil {
		result.Set("error", err.Error())
	} else {
		result.Set("error", js.Null())
	}
	// Match the standard {id, result|error} envelope used by PortClient on
	// the page side. We always resolve (never reject) and surface the script
	// error inside the result object so streaming output is preserved.
	msg := jsObject.New()
	msg.Set("id", id)
	msg.Set("result", result)
	port.Call("postMessage", msg)
}

// attachCoordPort installs the same handlePortMessage dispatcher on a
// fresh MessagePort that some other peer (the page, or a parent vm-host)
// transferred to us. Used by the `coord.attach` op so that vm-hosts can
// reach the coordinator over a private channel without constructing a
// SharedWorker themselves.
func attachCoordPort(p js.Value) {
	p.Set("onmessage", js.FuncOf(func(_ js.Value, ma []js.Value) any {
		if len(ma) == 0 {
			return nil
		}
		handlePortMessage(p, ma[0].Get("data"))
		return nil
	}))
	if startFn := p.Get("start"); startFn.Type() == js.TypeFunction {
		p.Call("start")
	}
}

// installCoordinatorBridge registers `onconnect` for the rumo-coordinator
// SharedWorker so each connecting page tab gets its own MessagePort
// speaking the protocol below.
//
// Page → coordinator messages have shape:
//
//	{ id: <number>, op: <string>, ...payload }
//
// Replies are
//
//	{ id, result }                      for sync results
//	{ id, error: "..." }                for failures
//	{ type:"output", routineId, chunk } streamed routine output
//	{ id, done: true }                  for routine completion notifications
func installCoordinatorBridge() {
	if !inSharedWorker() {
		return
	}
	self := js.Global()
	self.Set("onconnect", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		ports := args[0].Get("ports")
		if ports.IsUndefined() || ports.Length() == 0 {
			return nil
		}
		port := ports.Index(0)
		port.Set("onmessage", js.FuncOf(func(_ js.Value, ma []js.Value) any {
			if len(ma) == 0 {
				return nil
			}
			handlePortMessage(port, ma[0].Get("data"))
			return nil
		}))
		port.Call("start")
		// announce readiness so the page side can resolve its connect promise
		hello := jsObject.New()
		hello.Set("type", "ready")
		hello.Set("role", "coordinator")
		hello.Set("version", rumo.Version())
		port.Call("postMessage", hello)
		return nil
	}))
}

func portReply(port js.Value, id js.Value, result any) {
	msg := jsObject.New()
	msg.Set("id", id)
	msg.Set("result", result)
	port.Call("postMessage", msg)
}

func portError(port js.Value, id js.Value, err string) {
	msg := jsObject.New()
	msg.Set("id", id)
	msg.Set("error", err)
	port.Call("postMessage", msg)
}

// handlePortMessage routes a SharedWorker port message to the appropriate
// runtime operation. Output streams flow back as separate `output` messages.
func handlePortMessage(port js.Value, data js.Value) {
	id := data.Get("id")
	op := data.Get("op")
	if op.Type() != js.TypeString {
		portError(port, id, "missing op")
		return
	}
	switch op.String() {
	case "version":
		portReply(port, id, rumo.Version())
	case "commit":
		portReply(port, id, rumo.Commit())
	case "modules":
		portReply(port, id, jsModules(js.Undefined(), nil))
	case "exports":
		portReply(port, id, jsExports(js.Undefined(), nil))
	case "compile":
		go func() {
			src := toBytes(data.Get("source"))
			path := ""
			if p := data.Get("path"); p.Type() == js.TypeString {
				path = p.String()
			}
			b, err := compileSource(src, path)
			if err != nil {
				portError(port, id, err.Error())
				return
			}
			portReply(port, id, bytesToJS(b))
		}()
	case "run":
		go portRun(port, id, data, false)
	case "runCompiled":
		go portRun(port, id, data, true)
	case "spawn":
		go portSpawn(port, id, data)
	case "routine.cancel":
		rid := int64(data.Get("routineId").Int())
		if rt := routines.get(rid); rt != nil {
			rt.cancel()
		}
		portReply(port, id, true)
	case "routine.wait":
		rid := int64(data.Get("routineId").Int())
		secs := -1.0
		if v := data.Get("seconds"); v.Type() == js.TypeNumber {
			secs = v.Float()
		}
		go func() {
			rt := routines.get(rid)
			if rt == nil {
				portError(port, id, "unknown routine")
				return
			}
			d := time.Duration(-1)
			if secs >= 0 {
				d = time.Duration(secs * float64(time.Second))
			}
			portReply(port, id, rt.waitDone(d))
		}()
	case "routine.result":
		rid := int64(data.Get("routineId").Int())
		go func() {
			rt := routines.get(rid)
			if rt == nil {
				portError(port, id, "unknown routine")
				return
			}
			rt.waitDone(-1)
			rt.outMu.Lock()
			out := rt.outBuf.String()
			rt.outMu.Unlock()
			res := jsObject.New()
			res.Set("output", out)
			if rt.err != nil {
				res.Set("error", rt.err.Error())
			} else {
				res.Set("error", js.Null())
			}
			portReply(port, id, res)
		}()
	case "routine.write":
		rid := int64(data.Get("routineId").Int())
		rt := routines.get(rid)
		if rt == nil {
			portError(port, id, "unknown routine")
			return
		}
		b := toBytes(data.Get("chunk"))
		if len(b) > 0 {
			go func() { _, _ = rt.stdinW.Write(b) }()
		}
		portReply(port, id, true)
	case "routine.close":
		rid := int64(data.Get("routineId").Int())
		if rt := routines.get(rid); rt != nil {
			_ = rt.stdinW.Close()
			routines.del(rid)
		}
		portReply(port, id, true)
	case "routines.list":
		portReply(port, id, routines.snapshot())
	case "routines.prune":
		portReply(port, id, routines.prune())
	case "monitor.subscribe":
		monitor.addPort(port)
		portReply(port, id, true)
	case "coord.attach":
		// A vm-host (DedicatedWorker) is asking us to listen on a
		// transferable MessagePort it received from the page. Hook the
		// same handlePortMessage dispatcher up so that routine.allocate
		// / chan.* / etc. all work over this private channel — no reply
		// expected.
		cp := data.Get("port")
		if cp.IsUndefined() || cp.IsNull() {
			return
		}
		attachCoordPort(cp)
	case "fs.put":
		path := data.Get("path").String()
		sharedFS.put(path, toBytes(data.Get("content")))
		portReply(port, id, true)
	case "fs.get":
		path := data.Get("path").String()
		if d, ok := sharedFS.get(path); ok {
			portReply(port, id, bytesToJS(d))
		} else {
			portReply(port, id, js.Null())
		}
	case "fs.list":
		names := sharedFS.list()
		out := make([]any, len(names))
		for i, n := range names {
			out[i] = n
		}
		portReply(port, id, js.ValueOf(out))
	case "fs.delete":
		portReply(port, id, sharedFS.del(data.Get("path").String()))
	case "fs.clear":
		sharedFS.clear()
		portReply(port, id, true)
	case "fs.snapshot":
		// Returns a JS object {path: Uint8Array} of every file in the shared FS.
		// Used by the page to bundle the FS into a per-VM DedicatedWorker
		// before dispatching a runVM call.
		obj := jsObject.New()
		for _, name := range sharedFS.list() {
			if d, ok := sharedFS.get(name); ok {
				obj.Set(name, bytesToJS(d))
			}
		}
		portReply(port, id, obj)
	case "routine.register":
		// Register a remote routine (one running in a different DedicatedWorker)
		// in the monitor. The page reports per-VM lifecycle; the coordinator
		// just tracks it.
		rt := registerRemoteRoutine(data)
		portReply(port, id, rt.id)
	case "routine.update":
		rid := int64(data.Get("routineId").Int())
		rt := routines.get(rid)
		if rt == nil {
			portError(port, id, "unknown routine")
			return
		}
		if b := data.Get("bytes"); b.Type() == js.TypeNumber {
			rt.bytesOut.Store(int64(b.Int()))
		}
		portReply(port, id, true)
	case "routine.done":
		rid := int64(data.Get("routineId").Int())
		rt := routines.get(rid)
		if rt == nil {
			portError(port, id, "unknown routine")
			return
		}
		errStr := ""
		if e := data.Get("error"); e.Type() == js.TypeString {
			errStr = e.String()
		}
		var err error
		if errStr != "" {
			err = fmt.Errorf("%s", errStr)
		}
		rt.finish(err)
		go func() {
			time.Sleep(2 * time.Second)
			routines.del(rt.id)
		}()
		portReply(port, id, true)
	case "routine.allocate":
		// Called by a vm-host's Spawner to obtain a fresh routineId + worker
		// name for a new `go fn()` DedicatedWorker. Also creates the monitor row.
		rt := allocRoutine("go", "")
		wn := fmt.Sprintf("rumo-vm-%d", rt.id)
		rt.workerName = wn
		if v := data.Get("parentId"); v.Type() == js.TypeNumber {
			rt.parentID = int64(v.Int())
		}
		// Emit only after parentID + workerName are populated so that any
		// subscriber (e.g. the page's live monitor) renders the tree edge
		// on the first event instead of on the next poll.
		monitor.emitSpawned(rt)
		out := jsObject.New()
		out.Set("routineId", rt.id)
		out.Set("workerName", wn)
		portReply(port, id, out)
	case "chan.create":
		// vm-host asks the coordinator to allocate a backing queue. When
		// SharedArrayBuffer is available and buf>0 we hand the worker a
		// shared ring directly so subsequent send/recv calls bypass
		// postMessage entirely. Otherwise we fall back to the original
		// goroutine-backed LocalChan + RPC path.
		buf := 0
		if v := data.Get("buf"); v.Type() == js.TypeNumber {
			buf = v.Int()
		}
		out := jsObject.New()
		if sabSupported() && buf > 0 {
			ring := newSABRing(buf, sabDefaultSlotBytes)
			cid := vm.NewChanID()
			coordSABs.put(cid, ring.sab)
			out.Set("chanId", cid)
			out.Set("sab", ring.sab)
		} else {
			c := vm.NewLocalChan(buf)
			coordChans.Register(c)
			out.Set("chanId", c.ID())
			out.Set("sab", js.Null())
		}
		portReply(port, id, out)
	case "chan.lookup":
		// Workers that received a chan id by other means (marshalled
		// value, go fn() arg) ask the coordinator to hand them the SAB
		// so they can join the fast path. Replies with sab:null when the
		// chan is RPC-only (no SAB allocated for it).
		cid := int64(data.Get("chanId").Int())
		out := jsObject.New()
		if sab, ok := coordSABs.get(cid); ok {
			out.Set("sab", sab)
		} else {
			out.Set("sab", js.Null())
		}
		portReply(port, id, out)
	case "chan.send":
		go coordChanSend(port, id, data)
	case "chan.recv":
		go coordChanRecv(port, id, data)
	case "chan.close":
		go coordChanClose(port, id, data)
	default:
		portError(port, id, fmt.Sprintf("unknown op: %s", op.String()))
	}
}

// coordChans is the chan registry hosted by the coordinator. Every chan
// created by any vm-host (via the chan.create op) is registered here so that
// chan.send / chan.recv / chan.close can resolve the queue.
var coordChans = vm.NewChanRegistry()

func coordChanSend(port, id, data js.Value) {
	cid := int64(data.Get("chanId").Int())
	c := coordChans.Lookup(cid)
	if c == nil {
		portError(port, id, fmt.Sprintf("chan.send: unknown chanId %d", cid))
		return
	}
	blob := toBytes(data.Get("val"))
	val, err := vm.UnmarshalLive(blob)
	if err != nil {
		portError(port, id, "chan.send: "+err.Error())
		return
	}
	// chans embedded in the value get re-bound to the coordinator's registry
	// (which is the canonical owner) so receivers see them as local cores.
	vm.ResolveChans(val, coordChans, nil)
	if err := c.Core().Send(context.Background(), val); err != nil {
		portError(port, id, "chan.send: "+err.Error())
		return
	}
	portReply(port, id, true)
}

func coordChanRecv(port, id, data js.Value) {
	cid := int64(data.Get("chanId").Int())
	c := coordChans.Lookup(cid)
	if c == nil {
		portError(port, id, fmt.Sprintf("chan.recv: unknown chanId %d", cid))
		return
	}
	val, err := c.Core().Recv(context.Background())
	if err != nil {
		portError(port, id, "chan.recv: "+err.Error())
		return
	}
	if val == nil {
		portReply(port, id, js.Null())
		return
	}
	blob, err := vm.MarshalLive(val)
	if err != nil {
		portError(port, id, "chan.recv: "+err.Error())
		return
	}
	portReply(port, id, bytesToJS(blob))
}

func coordChanClose(port, id, data js.Value) {
	cid := int64(data.Get("chanId").Int())
	// SAB-backed chans live only in the SAB store on the coordinator —
	// the worker has already performed the in-memory close via the ring
	// header. We just drop our reference so chan.lookup stops handing it
	// out to fresh workers.
	if _, ok := coordSABs.get(cid); ok {
		coordSABs.del(cid)
		portReply(port, id, true)
		return
	}
	c := coordChans.Lookup(cid)
	if c == nil {
		portError(port, id, fmt.Sprintf("chan.close: unknown chanId %d", cid))
		return
	}
	if err := c.Core().Close(); err != nil {
		portError(port, id, "chan.close: "+err.Error())
		return
	}
	coordChans.Forget(cid)
	portReply(port, id, true)
}

// registerRemoteRoutine creates a routine entry whose execution lives in a
// different DedicatedWorker. The coordinator only tracks state changes that
// the orchestrating page reports via routine.update / routine.done messages.
func registerRemoteRoutine(data js.Value) *routine {
	kind := "remote"
	if v := data.Get("kind"); v.Type() == js.TypeString {
		kind = v.String()
	}
	name := ""
	if v := data.Get("name"); v.Type() == js.TypeString {
		name = v.String()
	}
	rt := allocRoutine(kind, name)
	if v := data.Get("workerName"); v.Type() == js.TypeString {
		rt.workerName = v.String()
	}
	if v := data.Get("parentId"); v.Type() == js.TypeNumber {
		rt.parentID = int64(v.Int())
	}
	// emit after parentID/workerName so subscribers see the full edge
	monitor.emitSpawned(rt)
	return rt
}

func portRun(port js.Value, id js.Value, data js.Value, compiled bool) {
	scriptArgs := jsValueStringArray(data.Get("args"))
	stdinStr := ""
	if v := data.Get("stdin"); v.Type() == js.TypeString {
		stdinStr = v.String()
	}
	streamID := data.Get("streamId")
	stream := streamID.Truthy()
	path := ""
	if p := data.Get("path"); p.Type() == js.TypeString {
		path = p.String()
	}

	kind := "run"
	if compiled {
		kind = "runCompiled"
		if path == "" {
			path = "compiled.bin"
		}
	}
	rt := newRoutine(kind, path)
	if stream {
		sid := streamID
		rt.onChunk = func(s string) {
			msg := jsObject.New()
			msg.Set("type", "output")
			msg.Set("streamId", sid)
			msg.Set("chunk", s)
			defer func() { _ = recover() }()
			port.Call("postMessage", msg)
		}
	}

	if compiled {
		rt.launchCompiled(toBytes(data.Get("bytecode")), scriptArgs, stdinStr)
	} else {
		rt.launchSource(toBytes(data.Get("source")), scriptArgs, stdinStr)
	}
	rt.waitDone(-1)
	rt.outMu.Lock()
	out := jsObject.New()
	out.Set("output", rt.outBuf.String())
	rt.outMu.Unlock()
	if rt.err != nil {
		out.Set("error", rt.err.Error())
	} else {
		out.Set("error", js.Null())
	}
	portReply(port, id, out)
	go func() {
		time.Sleep(2 * time.Second)
		routines.del(rt.id)
	}()
}

func portSpawn(port js.Value, id js.Value, data js.Value) {
	src := toBytes(data.Get("source"))
	path := ""
	if p := data.Get("path"); p.Type() == js.TypeString {
		path = p.String()
	}
	rt := newRoutine("spawn", path)
	rt.onChunk = func(s string) {
		msg := jsObject.New()
		msg.Set("type", "output")
		msg.Set("routineId", rt.id)
		msg.Set("chunk", s)
		defer func() { _ = recover() }()
		port.Call("postMessage", msg)
	}
	rt.launchSource(src, jsValueStringArray(data.Get("args")), func() string {
		if v := data.Get("stdin"); v.Type() == js.TypeString {
			return v.String()
		}
		return ""
	}())
	portReply(port, id, rt.id)

	// also notify completion so the page can resolve a wait/result without
	// polling
	go func() {
		rt.waitDone(-1)
		msg := jsObject.New()
		msg.Set("type", "done")
		msg.Set("routineId", rt.id)
		if rt.err != nil {
			msg.Set("error", rt.err.Error())
		} else {
			msg.Set("error", js.Null())
		}
		port.Call("postMessage", msg)
	}()
}

type portStream struct {
	port     js.Value
	streamID js.Value
}

func (p *portStream) Write(b []byte) (int, error) {
	msg := jsObject.New()
	msg.Set("type", "output")
	msg.Set("streamId", p.streamID)
	msg.Set("chunk", string(b))
	defer func() { _ = recover() }()
	p.port.Call("postMessage", msg)
	return len(b), nil
}

// vmHostStream is the streaming writer used by per-VM DedicatedWorkers.
// Unlike portStream it doesn't need a streamId since the page uses a fresh
// worker per VM and treats every output message on that port as belonging
// to it.
type vmHostStream struct{ port js.Value }

func (v *vmHostStream) Write(b []byte) (int, error) {
	msg := jsObject.New()
	msg.Set("type", "output")
	msg.Set("chunk", string(b))
	msg.Set("bytes", len(b))
	defer func() { _ = recover() }()
	v.port.Call("postMessage", msg)
	return len(b), nil
}

func jsValueStringArray(v js.Value) []string {
	if !v.InstanceOf(jsArray) {
		return nil
	}
	n := v.Length()
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = v.Index(i).String()
	}
	return out
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

// main dispatches one of two roles based on the worker scope:
//
//   - DedicatedWorkerGlobalScope → vm-host (runs exactly one VM and streams
//     it; used both for top-level rumo.run/runCompiled/spawn calls and for
//     per-`go fn()` children)
//   - SharedWorkerGlobalScope    → coordinator (FS, monitor, registry, chans);
//                                  the SharedWorker named "rumo-coordinator"
//   - any other scope (Window / standalone) → installs the in-process API
//     directly on the global object (used for tests / direct page loads
//     without a worker)
//
// Each `rumo.run` / `rumo.runCompiled` / `rumo.spawn` call from the page
// creates its own dedicated `rumo-vm-<id>` DedicatedWorker and reports
// lifecycle to the coordinator's monitor — so every running VM appears as
// a distinct worker in the browser's DevTools and as a row in the live
// monitor table.
func main() {
	switch {
	case inDedicatedWorker():
		// Every VM run (top-level or `go fn()` child) lives in its own
		// DedicatedWorker. Dispatch via self.onmessage; no onconnect.
		installVMHostBridgeDedicated()
	case inSharedWorker():
		// Coordinator (named "rumo-coordinator").
		installCoordinatorBridge()
	default:
		// Standalone: page loaded the wasm directly without any worker.
		installAPI(js.Global())
	}
	// Keep the runtime alive — without this the Go program would exit and
	// every registered js.Func would become invalid.
	select {}
}

