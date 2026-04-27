//go:build js && wasm

// main_js.go is the js/wasm entrypoint. It exposes a JavaScript API that
// mirrors the github.com/malivvan/rumo Go package and is intended to run
// inside a SharedWorker so that all page contexts that connect to the worker
// share a single rumo runtime, a single in-memory filesystem, and a single
// routine registry.
//
// Architecture
//
//	page tab(s)  ──connect──▶  SharedWorker  ──hosts──▶  WASM runtime (this binary)
//	                              port                     │
//	                                                       ├── shared in-mem FS
//	                                                       ├── module registry
//	                                                       └── routine registry
//
// The same API is also registered on the global object directly, so the binary
// also works when loaded into the main thread or a dedicated Worker for
// testing or single-context scripting. The SharedWorker `onconnect` hook adds
// a MessagePort bridge that forwards `postMessage` calls to the same API.
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

// fsStore is the in-memory filesystem shared by every VM running in this
// SharedWorker. Scripts launched through `rumo.run` see this map merged with
// their entrypoint source so that imports resolve against the same files the
// page placed via `rumo.fs.put`.
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
type routine struct {
	id     int64
	ctx    context.Context
	cancel context.CancelFunc

	stdinR *io.PipeReader
	stdinW *io.PipeWriter

	outMu  sync.Mutex
	outBuf bytes.Buffer
	// onChunk is invoked for every stdout chunk. Callers attach it through
	// spawn options or, for SharedWorker clients, the bridge plugs in a
	// chunk forwarder that posts {type:"output"} messages over the port.
	onChunk func(string)

	doneCh chan struct{}
	doneI  atomic.Int32
	err    error
}

// Write makes the routine an io.Writer for VM stdout.
func (r *routine) Write(p []byte) (int, error) {
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
	if r.stdinW != nil {
		_ = r.stdinW.Close()
	}
	if r.doneI.CompareAndSwap(0, 1) {
		close(r.doneCh)
	}
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
	delete(r.items, id)
}

var routines = &routineRegistry{}

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
func newRoutine() *routine {
	ctx, cancel := context.WithCancel(context.Background())
	pr, pw := io.Pipe()
	rt := &routine{
		ctx:    ctx,
		cancel: cancel,
		stdinR: pr,
		stdinW: pw,
		doneCh: make(chan struct{}),
	}
	rt.id = routines.next.Add(1)
	routines.add(rt)
	return rt
}

// launch spawns the goroutine that runs the script. Callers must set
// rt.onChunk before calling this — otherwise the first chunk(s) of output
// can race against the assignment.
func (rt *routine) launch(source []byte, opts js.Value) {
	args := optStringArray(opts, "args")
	path := optString(opts, "path")
	if path == "" {
		path = fmt.Sprintf("routine_%d.rumo", rt.id)
	}
	if s := optString(opts, "stdin"); s != "" {
		go func() { _, _ = io.Copy(rt.stdinW, bytes.NewReader([]byte(s))) }()
	}
	go func() {
		err := runSource(rt.ctx, source, path, args, rt.stdinR, rt)
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

	return newPromise(func(resolve, reject func(any)) {
		var buf bytes.Buffer
		w := io.MultiWriter(&buf, &cbWriter{cb: onOutput})
		var r io.Reader
		if stdinStr != "" {
			r = bytes.NewReader([]byte(stdinStr))
		}
		err := runSource(context.Background(), src, path, scriptArgs, r, w)
		out := jsObject.New()
		out.Set("output", buf.String())
		if err != nil {
			out.Set("error", err.Error())
		} else {
			out.Set("error", js.Null())
		}
		resolve(out)
	})
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

	return newPromise(func(resolve, reject func(any)) {
		var buf bytes.Buffer
		w := io.MultiWriter(&buf, &cbWriter{cb: onOutput})
		var r io.Reader
		if stdinStr != "" {
			r = bytes.NewReader([]byte(stdinStr))
		}
		err := runCompiled(context.Background(), data, scriptArgs, r, w)
		out := jsObject.New()
		out.Set("output", buf.String())
		if err != nil {
			out.Set("error", err.Error())
		} else {
			out.Set("error", js.Null())
		}
		resolve(out)
	})
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

	rt := newRoutine()
	if onOutput.Type() == js.TypeFunction {
		f := onOutput
		rt.onChunk = func(s string) { f.Invoke(s) }
	}
	rt.launch(src, opts)

	handle := jsObject.New()
	handle.Set("id", rt.id)

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

// cbWriter forwards writes to a JS function (used as a streaming hook).
type cbWriter struct {
	cb js.Value
}

func (c *cbWriter) Write(p []byte) (int, error) {
	if !c.cb.Truthy() {
		return len(p), nil
	}
	defer func() { _ = recover() }()
	c.cb.Invoke(string(p))
	return len(p), nil
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
// API installer + SharedWorker bridge
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
// SharedWorkerGlobalScope.
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

// installSharedWorkerBridge registers `onconnect` so each connecting page tab
// can drive the runtime through a MessagePort. Messages have the shape
//
//	{ id: <number>, op: <string>, args: [...] }
//
// Replies are
//
//	{ id, result }                     for sync results
//	{ id, error: "..." }               for failures
//	{ type:"output", routineId, chunk } streamed routine output
//	{ id, done: true }                 for routine completion notifications
func installSharedWorkerBridge() {
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
	default:
		portError(port, id, fmt.Sprintf("unknown op: %s", op.String()))
	}
}

func portRun(port js.Value, id js.Value, data js.Value, compiled bool) {
	scriptArgs := jsValueStringArray(data.Get("args"))
	stdinStr := ""
	if v := data.Get("stdin"); v.Type() == js.TypeString {
		stdinStr = v.String()
	}
	streamID := data.Get("streamId")
	stream := streamID.Truthy()

	var buf bytes.Buffer
	w := io.Writer(&buf)
	if stream {
		w = io.MultiWriter(&buf, &portStream{port: port, streamID: streamID})
	}
	var r io.Reader
	if stdinStr != "" {
		r = bytes.NewReader([]byte(stdinStr))
	}
	var err error
	if compiled {
		err = runCompiled(context.Background(), toBytes(data.Get("bytecode")), scriptArgs, r, w)
	} else {
		path := ""
		if p := data.Get("path"); p.Type() == js.TypeString {
			path = p.String()
		}
		err = runSource(context.Background(), toBytes(data.Get("source")), path, scriptArgs, r, w)
	}
	out := jsObject.New()
	out.Set("output", buf.String())
	if err != nil {
		out.Set("error", err.Error())
	} else {
		out.Set("error", js.Null())
	}
	portReply(port, id, out)
}

func portSpawn(port js.Value, id js.Value, data js.Value) {
	src := toBytes(data.Get("source"))
	rt := newRoutine()
	rt.onChunk = func(s string) {
		msg := jsObject.New()
		msg.Set("type", "output")
		msg.Set("routineId", rt.id)
		msg.Set("chunk", s)
		port.Call("postMessage", msg)
	}
	rt.launch(src, data)
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

func main() {
	// Always install the API on the current global so that the binary works
	// in main thread, dedicated worker, or shared worker contexts.
	installAPI(js.Global())
	// In a SharedWorker we additionally bridge connecting MessagePorts.
	installSharedWorkerBridge()

	// Keep the runtime alive — without this the Go program would exit and
	// every registered js.Func would become invalid.
	select {}
}

