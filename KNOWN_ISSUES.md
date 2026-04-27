> The following are known issues that will be addressed in a future release as they require human attention to resolve:

## 1. Cross-platform compatibility

### 1.1 `std/os` unconditionally imports `os/exec` & `syscall` &nbsp; **CRIT (browser/wasi)**

- `std/os/os.go:8-9` imports `os/exec` and `syscall` at package level.
- `os/exec` in `js/wasm` builds returns "exec: not implemented" on every
  call; in `wasip1` it is functionally absent.
- `syscall.Signal` is unsupported on `js/wasm`.
- `os.StartProcess`, `os.FindProcess`, `os.Hostname`, `os.Chown`,
  `os.Lchown`, `os.Setuid`/`Setgid`-related fields, named-pipe and
  socket file modes are no-ops or compile errors on the JS runtime.
- `Modules()` (`rumo.go:44`) eagerly registers *every* builtin module,
  including `os`, so anyone using the default module map cannot strip
  unsupported modules out at build time.
- **Fix:** introduce build tags (`//go:build !js && !wasip1`) to gate
  the heavy parts of `std/os`; provide a `nostdos` tag to omit the
  module entirely; expose a `WithoutOS()` helper alongside `Modules()`.

### 1.2 Native FFI fundamentally cannot work on WASM &nbsp; **HIGH**

- `vm/native.go:260` calls `purego.Dlopen` (BSD/Linux/Mac/Windows only).
  No `wasm` backend.
- `vm/native.go` and `vm/compiler_native.go` are guarded by
  `//go:build native`, but the *parser* recognises the keyword
  unconditionally (`vm/parser/native_stub.go:29` returns a compile
  error). Good.
- However the bytecode *type tag* for `*Native` is allocated
  unconditionally in `vm/bytecode.go:228`, so a bytecode file produced
  by a `-tags native` build cannot be loaded by a non-native runtime
  (the type is stubbed in `native_stub.go` and `Call` returns an error,
  but `RemoveDuplicates` still references the concrete type — fine —
  yet a deserialized `Native` constant is just a bare `ObjectImpl` with
  no funcs/path and will crash if called).
- **Fix:** version the bytecode header with a "feature flags" word
  listing the toolchain capabilities required to load it, and refuse
  the file when the runtime lacks them.

### 1.5 REPL & input handling unusable in browser / WASI &nbsp; **MED**

- `rumo.go:201-203` calls `term.IsTerminal(int(fin.Fd()))` —
  `*os.File.Fd()` returns 0 / panics in some non-FD-backed runtimes.
- `readline` cannot share the JS event loop. Embedding rumo into a
  browser via Go's `js/wasm` will require an alternative driver. There
  is no abstraction layer (`Reader`/`Writer` only) to support that.
- **Fix:** factor the REPL out of the core package, or add a
  `RunREPLLoop` that takes pre-read lines from a callback.

### 1.6 Goroutines used for routines, channels, sleep &nbsp; **MED**

`vm/routinevm.go` and `std/times/times.go:103` spawn raw goroutines.
On `js/wasm` the runtime is cooperative — every blocking syscall has
to yield to the event loop. This means:

- `times.sleep(950ms)` (≤ 1 s branch) calls `time.Sleep` directly on
  the calling goroutine, blocking the JS thread.
- `chan.recv()` blocks the entire wasm module if the producing
  goroutine never runs.
- Tight loops in scripts will starve the JS scheduler completely.
- **Fix:** in `js/wasm` builds, periodically yield via
  `runtime.Gosched()` from the VM run loop; document that browser
  embeds must spawn the VM in a Worker.

### 1.7 Build matrix omits the platforms the README promises &nbsp; **MED**

`Makefile:64-73` builds only linux/darwin/windows × {386, amd64,
arm, arm64}. There is no `js/wasm`, no `wasip1`, no CI signal for
either. The audit could not verify the WASM claims because *they were
never built*.

- **Fix:** add `js/wasm` and `wasip1/wasm` to `release` and a CI job
  that at least compiles them.

## 2. Performance & resource use


### 2.1 `Map.IndexGet`/`IndexSet` always lock &nbsp; **MED**

`vm/objects.go:1409, 1425`. Single-threaded use is the common case;
mutex traffic dominates micro-benchmarks. Even more so on `js/wasm`
where mutex contention is emulated.

- **Fix:** add an "owned" fast-path bit (set when the map is
  constant-pool / fresh) that skips the lock. Or split into
  `Map` (locked) vs. `LocalMap` (unsynchronised).


### 2.2 Default limits effectively unbounded &nbsp; **HIGH**

`MaxStringLen = 2_147_483_647`, `MaxBytesLen = 2_147_483_647`,
`MaxAllocs = -1`. Embedders running untrusted scripts get *zero*
out-of-the-box protection.

- **Fix:** ship safe defaults (e.g. 16 MB / 16 MB / 10 M allocs) and
  document `vm.Unlimited()` for users who knowingly opt in.