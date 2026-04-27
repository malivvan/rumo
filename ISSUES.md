# Architectural / Systemic Issues

Audit of `github.com/malivvan/rumo`, `…/vm`, and `…/std`. Findings are
ranked by likelihood of causing user-visible damage and grouped by
theme. File:line references use the snapshot at audit time.

Severity legend: **CRIT** = exploitable / data loss, **HIGH** = silent
incorrect behaviour, **MED** = future-pain & footgun, **LOW** =
ergonomics.

---

## 1. Cross-platform compatibility

The README claims rumo "must run in all the different environments
(unix/windows/browser/wasi)". The current code base does not. Several
hard blockers prevent a `js/wasm` or `wasip1` build from working at
runtime, even where it compiles.

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

### 1.3 Host-specific values baked into stdlib constants &nbsp; **HIGH**

`std/os/os.go:38-40` exposes:

```go
Const("path_separator string", string(os.PathSeparator))
Const("path_list_separator string", string(os.PathListSeparator))
Const("dev_null string", os.DevNull)
```

These are evaluated at compile time of the *Go binary*. A bytecode
file produced with `dev_null = "/dev/null"` and shipped to a Windows
host (`NUL`) will misbehave silently. The same applies to all
`os.O_*`, `os.Mode*` constants — Go's portable values do not match the
target OS's actual constants once the bytecode crosses platforms.

- **Fix:** resolve `os.*` constants at script execution time, not at
  Go compile time of the host binary, so the running platform's value
  is used. This is a property of "language must run on every
  platform" - the *script bytecode* should be portable, not the host.

### 1.4 `times.date` leaks the host's local timezone &nbsp; **HIGH**

`std/times/times.go:386` constructs a date with
`time.Now().Location()`. The script writer expects a deterministic
calendar; running the same script on a server in UTC vs. a developer
laptop in CET produces different `time` values for the same inputs.

- **Fix:** default to `time.UTC`; require an explicit
  `times.date_in(zone, …)` for local construction.

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

### 1.8 `std/times.sleep` cancellation semantics &nbsp; **HIGH**

`std/times/times.go:97-117`:

```go
if time.Duration(i1) <= time.Second {
    time.Sleep(time.Duration(i1))   // un-interruptible
    return
}
done := make(chan struct{})
go func() {
    time.Sleep(time.Duration(i1))   // continues to run after ctx.Done
    select { case <-ctx.Done(): case done <- struct{}{}: }
}()
```

- A `sleep(950)` (sub-second) cannot be aborted by `vm.Abort()` or
  `ctx.Cancel()`.
- A `sleep(1h)` that *is* cancelled leaves the goroutine alive for an
  hour. With 1000 cancelled scripts that's 1000 leaked goroutines.
- **Fix:** unify on `time.NewTimer` + `select { ctx.Done() }`; never
  block the calling goroutine on a non-cancellable sleep.

---

## 2. Security

### 2.1 `OpImmutable` does not actually freeze data &nbsp; **CRIT** ✅

`vm/vm.go:620-643`:

```go
case parser.OpImmutable:
    value := v.stack[v.sp-1]
    switch value := value.(type) {
    case *Array:
        var immutableArray Object = &ImmutableArray{Value: value.Value}
        ...
    case *Map:
        var immutableMap Object = &ImmutableMap{Value: value.Value}
```

The underlying slice / map is *shared* with the original mutable
container. After `imm := immutable(arr); arr[0] = 99`, `imm[0]` is
also 99. Compare with `builtinImmutableArray` /
`builtinImmutableMap` (`vm/builtins_new.go:204-270`) which correctly
copy.

This is not a bug in immutable semantics only — it is a security
hazard, because exported (i.e. immutable) module attributes
(`BuiltinModule.AsImmutableMap`) are returned to scripts under the
assumption they cannot be tampered with.

- **Fix:** clone the slice / map in `OpImmutable`; deep-clone if
  nested mutable containers can leak.

### 2.2 `Ptr` is constructible from any integer &nbsp; **CRIT (with native)** ✅

- `vm/vvm.go:282-313` `ToPtr(*Int) → unsafe.Pointer(uintptr(o.Value))`.
- `vm/builtins_new.go` `builtinPtr` accepts an `Int`.
- `vm/encoding.go:399` deserializes a `Ptr` from the wire as
  `unsafe.Pointer(uintptr(u))`.

Any script — and any malicious bytecode file — can synthesize an
arbitrary pointer value. With `-tags native`, `Native.Call` happily
hands that pointer to a foreign function (`vm/native.go:464-484`).
Result: arbitrary memory read/write, RCE.

Even *without* `native`, embedders that expose Go callbacks taking
`*vm.Ptr` will dereference attacker-controlled addresses.

- **Fix:** make `Ptr` a sealed type only obtainable from FFI return
  values; remove `Int → Ptr` coercion; never marshal `Ptr` (it cannot
  survive an exec across runs anyway).

### 2.3 Bytecode integrity uses CRC64, not a cryptographic hash &nbsp; **HIGH**

`script.go:288-294` validates with `crc64/ECMA`. CRC is a
non-cryptographic checksum: a passive attacker who can modify the
bytecode file can also recompute the CRC. The four-byte magic +
two-byte version + CRC is therefore *integrity-against-corruption
only*, not authentication.

If embedders treat compiled bytecode as a trust boundary (e.g. ship
signed packages), this misleads them.

- **Fix:** swap CRC64 for SHA-256 (or BLAKE2b) and document it as
  "tamper-detection, not authentication". Add an optional Ed25519
  signature block.

### 2.4 Unmarshal lacks size caps & sanity checks &nbsp; **HIGH**

- `codec/encoding.go:118-150` `UnmarshalSlice` reads a varint and
  immediately `make([]T, s)`.
- `codec/encoding.go:47-58` `UnmarshalString` allocates `string(b[n:n+s])`
  with no upper bound aside from `len(b)`.
- `vm/encoding.go:77` `MakeObject` *panics* on an unknown type code,
  so a corrupted byte produces a host-process crash instead of an
  error.
- A bytecode file with `len = 1<<31 - 1` immediately allocates 16 GB
  of `[]Object`.

- **Fix:** validate every length against `Config.MaxBytesLen` /
  `MaxStringLen` (or a separate `MaxConstantBytes` cap) *before*
  allocating; convert panics in `MakeObject` to errors that bubble
  through `Unmarshal`.

### 2.5 File-import path containment can be bypassed via symlinks &nbsp; **HIGH**

`vm/compiler.go:578-592` resolves `modulePath = filepath.Abs(filepath.Join(c.importDir, name))`
and checks containment with `filepath.Rel`. This is a *lexical*
check; if `importDir/foo.rumo` is a symlink that points at
`/etc/passwd`, the read at line 594 succeeds and the contents are
embedded into bytecode constants.

- **Fix:** call `filepath.EvalSymlinks` on both `c.importBase` and
  `modulePath` before comparing; deny resolution failures.

### 2.6 `std/os` exposes process & environment without sandboxing &nbsp; **HIGH**

- `os.exec`, `os.start_process`: arbitrary process launch.
- `os.exit`: terminates the *host* program. There is no
  intercept hook.
- `os.setenv`, `os.unsetenv`, `os.clearenv`: mutates process-wide
  state, leaking effects to concurrently running VMs and the host.
- `os.chdir`: same — and, worse, breaks relative-path-based lookups
  in *other* in-flight scripts.
- `os.read_file`, `os.write_file`-via-`file.write`: no allow-list.

For "small, fast and *secure*" to hold, embedders need to gate these.
There is no `Config.AllowOS bool`, no permission interface, no
analogue to Lua's `loadstring` / Deno's `--allow-*`.

- **Fix:** add a per-VM permissions struct (read/write/exec/env) and
  consult it at every privileged builtin entry; default to *deny*.

### 2.7 `expand_env` length accounting under-counts &nbsp; **MED**

`std/os/os.go:228-249` only counts the *value bytes substituted*
into the result, not the literal text of the template. A 100MB
template that references `$X` once with an empty `X` slips through a
1MB `MaxStringLen`.

- **Fix:** check `len(s)` against the cap *after* `os.Expand`
  returns, before constructing the `String`.

### 2.8 Format-string DoS &nbsp; **MED**

`builtinFormat` (`vm/builtins.go:385`) and `vm.Format` accept
arbitrary verbs including width specifiers. `format("%[1]20000000d", 1)`
produces a 20MB string instantly, ignoring `MaxAllocs` (the limit is
checked once per `OpBinaryOp`, not inside the formatter).

- **Fix:** intercept width / precision specifiers and reject anything
  above a configurable bound.

### 2.9 `std/text.repeat` & `re_replace` size checks are post-hoc &nbsp; **MED**

`text.go:608` checks `len(s1)*i2 > MaxStringLen` *before* the call —
good — but `i2` is taken from a script-supplied `int`. With
`s1 = "x"` and `i2 = 1<<63`, the multiplication wraps in Go's two's
complement and might *pass* the check. Use unsigned multiplication
with overflow detection.

### 2.10 `std/rand` is `math/rand`, not crypto-safe &nbsp; **MED**

`std/rand/rand.go` exports `rand.int`, `rand.read`, etc. Most users
will reach for `rand.read(bytes)` to make a token; that's a non-CSPRNG
output, predictable on demand.

- **Fix:** rename to `mrand`, add a `crand` module backed by
  `crypto/rand`. Mark every function with whether it is suitable for
  security use.

### 2.11 `Native.Call` accepts arbitrary library paths from script &nbsp; **HIGH**

`vm/native.go:260` does `purego.Dlopen(o.Path, …)` where `o.Path`
came directly from the source `native foo = "/some/path" { … }`. A
malicious script (or compiled bytecode) loads any `.so`/`.dll` on
disk. No allow-list, no signature check.

- **Fix:** require the embedder to register named native bindings
  ahead of time; reject `Dlopen` of paths not present in that
  registry. Reserve `-tags native` for trusted, embedder-driven
  configurations.

### 2.12 Bytecode does not isolate `__module_name__` namespace &nbsp; **MED**

`vm/bytecode.go:351-356` `inferModuleName` reads the
`__module_name__` key from any `*ImmutableMap`. A user can construct
`{"__module_name__": "os"}` in script source and `RemoveDuplicates`
will treat it as the canonical `os` module, replacing valid module
constants on subsequent `fixDecodedObject` passes.

- **Fix:** carry the module name out-of-band (e.g. a separate
  `*Module` constant type), or namespace it under a non-string key
  unobtainable from script.

---

## 3. Concurrency & data races

### 3.1 `String.runeStr` lazy population is racy &nbsp; **HIGH**

`vm/objects.go:1583-1604`:

```go
if o.runeStr == nil {
    o.runeStr = []rune(o.Value)
}
```

`String` instances are shared (constants pool, immutable maps,
returned values). Two goroutines calling `s[0]` concurrently both see
nil, race on the assignment, and one's `[]rune` may be discarded
mid-use.

- **Fix:** populate `runeStr` once at construction (cheap for short
  strings) or guard with `sync.Once` / atomic pointer.

### 3.2 `Bytes` has no mutex &nbsp; **HIGH**

`Bytes.Iterate` (`vm/iterator.go:67`) captures the raw `[]byte`
slice; concurrent `IndexSet` (only via runtime selectors) or
mutations from native FFI cause data races. `BytesIterator` does not
snapshot.

- **Fix:** add `sync.RWMutex` like `Array`/`Map`, or document
  immutable semantics for `Bytes` and remove all mutating paths.

### 3.3 `Map`/`Array` partial locking &nbsp; **MED**

The mutex is private (`mu sync.RWMutex`), but several call sites
*read or write `Value` directly without the lock*:

- `vm/encoding.go:184-188, 273-282` — marshal walks `Value` of
  `Array`/`Map`/`ImmutableArray`/`ImmutableMap` without locking.
- `vm/objects.go` `Array.IndexGet` locks but `Array.BinaryOp Add` only
  locks `o`/`rhs` for the snapshot duration; if the caller concurrently
  appends, you get a torn snapshot.
- `vm/bytecode.go:282-318` mutates `Array.Value`/`Map.Value` during
  `fixDecodedObject` without lock — fine at deserialize time but
  dangerous as a precedent.

- **Fix:** either expose the mutex through a sealed `Lock()/Unlock()`
  method on the `Object` interface or eliminate `mu` entirely and
  require callers to copy-on-write.

### 3.4 `vmChildCtl.cancelFns` grows without bound &nbsp; **MED**

`vm/vm.go:283-316`. `addChild` appends to `cancelFns`; `delChild`
calls each `cancel()` but does not remove it from the slice. A long
running parent VM that spawns N short-lived non-compiled callees
ends up with len(cancelFns) = N — pure leak, plus eventual O(N) walk
on every `Abort()`.

- **Fix:** track via a map keyed by an opaque token returned from
  `addChild`, deleted in `delChild`.

### 3.5 `BuiltinModule.AsImmutableMap` shares attribute references &nbsp; **MED**

`vm/objects.go:389-396` does `attrs[k] = v.Copy()` which is a
*shallow* copy for compound types (recall `Array.Copy` does deep, but
`ObjectPtr.Copy` returns the same pointer). Two concurrent VMs that
each "import" the same module receive `BuiltinFunction` and
`ImmutableMap` instances that share the same backing function
closures. Most are stateless, but the `os` module's per-process
helpers and `times` constants are not.

- **Fix:** treat builtin module attributes as singletons (the
  current `Copy()` is intended to deep-clone) and freeze them at
  module load.

### 3.6 `Program.Marshal` reads `bytecode` under RLock but compiler may write &nbsp; **MED**

`script.go:324`. The lock guards `globals`/`globalIndices`/`maxAllocs`,
but `bytecode.MainFunction.Instructions`, `bytecode.Constants` are
mutated by `RemoveDuplicates` and `updateConstIndexes`. Concurrent
`Compile`/`Run`/`Marshal` is not safe.

- **Fix:** make `Bytecode` immutable after `Compiler.Bytecode()`
  returns, or document that `Compile` must not race with `Marshal`.

### 3.7 Routine return value double-locking pattern &nbsp; **LOW**

`vm/routinevm.go:107-109`:

```go
gvm.mu.Lock(); gvm.VM = nil; gvm.mu.Unlock()
```

Right after that, `getRet` reads `gvm.ret` under the lock. Fine, but
`gvm.ret` is also assigned in `defer`. The write happens-before the
`close(gvm.doneChan)`, the read happens-after, so it is correct *only*
because of the channel close memory barrier. A future refactor that
removes the channel will introduce a race silently.

- **Fix:** make `ret` an `atomic.Pointer[ret]` to make the ordering
  explicit.

---

## 4. Performance & resource use

### 4.1 `range(start, stop, step)` is eagerly materialized &nbsp; **HIGH**

`vm/builtins.go:367-383` builds an `[]Object` of length
`(stop-start)/step` upfront. `range(0, 1_000_000_000)` allocates 8 GB
of pointers before `for x in range(...)` ever runs.

- **Fix:** introduce a lazy `RangeIterator` so `for x in range(...)`
  produces values on demand. Keep `range` as the language-level
  construct and reserve `to_array(range(...))` for materialisation.

### 4.2 `String.IndexGet` & `String.Iterate` decode the full string &nbsp; **MED**

`vm/objects.go:1582-1604` populates `[]rune` for the whole string on
first index access. `s[0]` on a 100 MB string allocates a 100 MB
rune slice (4× expansion) just to read one code point.

- **Fix:** for `IndexGet` use `utf8.DecodeRuneInString` with byte
  scanning; reserve full decode for iterator construction.

### 4.3 `isolateClosureFreeRec` runs on every `go fn()` &nbsp; **MED**

`vm/routinevm.go:226-262`. Every routine spawn deep-walks the
closure graph. Hot fan-out patterns (e.g., a fixed worker pool that
re-launches goroutines per job) pay this cost N times even though
the closures are static.

- **Fix:** memoize the isolated copy on the source `*CompiledFunction`
  the first time it is spawned; or do the isolation lazily on first
  `OpSetFree` from a child.

### 4.4 `Map.IndexGet`/`IndexSet` always lock &nbsp; **MED**

`vm/objects.go:1409, 1425`. Single-threaded use is the common case;
mutex traffic dominates micro-benchmarks. Even more so on `js/wasm`
where mutex contention is emulated.

- **Fix:** add an "owned" fast-path bit (set when the map is
  constant-pool / fresh) that skips the lock. Or split into
  `Map` (locked) vs. `LocalMap` (unsynchronised).

### 4.5 `Bytecode.RemoveDuplicates` skips `Bytes` and `Map` entirely &nbsp; **LOW**

`vm/bytecode.go:217-227`. `embed("file.txt")` uses a `Bytes` constant.
A program that embeds the same file from multiple modules duplicates
it. Same for embedded maps.

- **Fix:** use `crypto/sha256` of the bytes/map JSON as a dedup key
  (or BLAKE2 for speed); the cost is paid once at compile time.

### 4.6 `DefaultConfig` is a *pointer* to mutable globals &nbsp; **HIGH**

`vm/vvm.go:38-45` declares `DefaultConfig = &Config{ … }`. Tests
mutate it (`std/text/text_test.go:177-179`,
`vm/vm_test.go:726`) — that's a global goroutine race vs. any
parallel test or production VM.

Worse, **runtime call sites in `std/*` and in `vm/objects.go:420,
1509, 1515` reference `DefaultConfig.MaxStringLen` directly, *not*
the per-VM `cfg`.** A user who passes a strict `Config` to `NewVM`
gets it ignored by the very paths that should respect it.

- **Fix:** make `DefaultConfig` a value (`Config{...}`) returned by a
  function `Default()`, and route every limit check through
  `vm.Config` of the running VM (look it up via context if needed).

### 4.7 Default limits effectively unbounded &nbsp; **HIGH**

`MaxStringLen = 2_147_483_647`, `MaxBytesLen = 2_147_483_647`,
`MaxAllocs = -1`. Embedders running untrusted scripts get *zero*
out-of-the-box protection.

- **Fix:** ship safe defaults (e.g. 16 MB / 16 MB / 10 M allocs) and
  document `vm.Unlimited()` for users who knowingly opt in.

### 4.8 `Int.Copy`, `Float64.Copy`, … allocate every call &nbsp; **LOW**

Every binary op that returns a value type allocates a new struct.
The "interner" trick used for `TrueValue`/`FalseValue`/`UndefinedValue`
is not extended to small ints. A tight `for i := 0; i < N; i++ {}`
allocates N `*Int` objects.

- **Fix:** intern `[-128, 127]` (or a configurable window) of `*Int`,
  similar to CPython.

### 4.9 `for x in s { … }` over a string repeatedly checks `runeStr` &nbsp; **LOW**

`vm/objects.go:1596-1604`. The check is inside the iterator
construction, not on every iteration, but a script that does
`for x in s { for y in s { … } }` re-decodes nothing — fine — but
discarding the inner `[]rune` is wasted because the outer is the
same. Cache lifetime is per-`*String`, which is correct, just
underused given that constants are reused.

---

## 5. Correctness traps

### 5.1 `is_int32` aliases `is_char`; `is_uint32` aliases `is_uint` &nbsp; **HIGH**

`vm/builtins.go:47-52`:

```go
addBuiltinFunction("is_int32", builtinIsChar)
addBuiltinFunction("is_int64", builtinIsInt)
addBuiltinFunction("is_uint32", builtinIsUint)
```

`is_int32(42)` returns false because `42` is `*Int` (int64), while
`is_int32('A')` returns true because the underlying type is `*Char`.
Anyone porting Go code will be misled.

- **Fix:** drop the aliases or add genuine `Int32`/`Uint32` types
  that aren't disguised characters.

### 5.2 Tail-call optimisation suppressed by *any* defer &nbsp; **MED**

`vm/vm.go:904`:

```go
if callee == v.curFrame.fn && len(v.curFrame.defers) == 0 { // recursion
```

A function with even one `defer` cannot tail-recurse. That's a
correctness *fix* (defers must run in order), but it silently turns
naturally tail-recursive code into stack-blowing recursion. There is
no diagnostic.

- **Fix:** when compiling a function that contains `defer`, emit a
  warning if the recursion would have been tail-call optimisable;
  document the trade-off.

### 5.3 `OpBinaryOp Less`/`LessEq` swaps operands at compile time &nbsp; **MED**

`vm/compiler.go:133-150` rewrites `a < b` to `b > a`, generating only
the `Greater`/`GreaterEq` opcodes. This is fine for symmetric numeric
types, but breaks user-defined `Object`s where `BinaryOp(<, rhs)`
is implemented but `BinaryOp(>, lhs)` is not. With `*Char` `<` `*Int`
implemented in `Char.BinaryOp` (`vm/objects.go:546-565`) and the
mirror not implemented in `Int.BinaryOp`, the swapped form fails.

- **Fix:** never swap; emit dedicated `OpLess`/`OpLessEq` opcodes.
  Cost is one byte per opcode in the VM dispatch table.

### 5.4 `Int8`/`Int16`/`Uint*` shift width is not checked &nbsp; **MED**

`vm/numeric.go` reuses `signedIntBinaryOp`/`unsignedIntBinaryOp`
which operate on `int64`/`uint64`. `uint8(1) << 9 = 0` silently
because the result is wrapped in `wrap(uint64) → uint8`. Negative
shift counts pass through `uint64(rv)` and produce huge values.

- **Fix:** in the wrapper, validate `0 ≤ rv < 8/16/32/64` and return
  `ErrInvalidOperator` otherwise — match Go's runtime panic.

### 5.5 `Float*.IsFalsy` returns true only for NaN &nbsp; **LOW**

Go convention says `0.0` is falsy; rumo's `Float32`/`Float64.IsFalsy`
treats `NaN` as falsy and `0.0` as *truthy*. The opposite of `Int`
where `0` is falsy. Document or reconcile.

### 5.6 `Char` zero value is falsy &nbsp; **LOW**

`vm/objects.go:578` `Char.IsFalsy() = (Value == 0)`. `'\x00'` is a
valid character; `if c { … }` skips the body for NUL even though
the script writer means "I have a character".

### 5.7 `Float32 == Float32` uses bit-identity not numerical equality &nbsp; **LOW**

`vm/objects.go:796-803`. `NaN == NaN` should be false per IEEE 754 —
let me double-check. Reading the code:

```go
func (o *Float32) Equals(x Object) bool {
    t, ok := x.(*Float32)
    if !ok { return false }
    return o.Value == t.Value
}
```

That's IEEE compare (NaN != NaN). Good. But `Map` keys / dedup uses
`map[float32]int` (`vm/bytecode.go:146`) which *does* compare by bit
pattern. Two NaNs with different mantissas dedup separately; same
NaN written twice dedups. Inconsistent.

### 5.8 Divisor `0` for `Float` produces `Inf`/`NaN`, not error &nbsp; **LOW**

`Float64.BinaryOp Quo` with `rhs.Value == 0` returns `Inf`. `Int.Quo`
returns `ErrDivisionByZero`. Mixed-type operations behave
inconsistently.

### 5.9 `Equals` is O(∞) for cyclic structures &nbsp; **MED**

`Array.Equals`, `Map.Equals` recurse. Two cyclic arrays compare
forever (well, stack overflow). Same with `Copy`, `String()`,
`ToInterface`.

- **Fix:** track visited pointers in a per-call set.

### 5.10 `Builtin` index baked into bytecode &nbsp; **MED**

`vm/builtins.go:13-79` populates `builtinFuncs` via `init()`. The
*order* of `addBuiltinFunction` calls determines the integer index
emitted by the compiler in `OpGetBuiltin`. Adding a new builtin in
the middle of the list silently shifts all later indices, so old
bytecode files now resolve `len` to `format`. There is no version
gate.

- **Fix:** look up builtins by *name* in the bytecode (`OpGetBuiltinByName`
  resolved at unmarshal time), or freeze the index table in
  `FormatVersion`.

### 5.11 `Time` deserialisation drops timezone info &nbsp; **LOW**

`vm/encoding.go:425` always rebinds to `time.UTC`. A
`time.Now().In(berlin)` round-tripped through bytecode comes back
in UTC. Probably intentional, but undocumented.

### 5.12 Compiler reads files even when the script is "compile only" &nbsp; **MED**

`vm/compiler.go:594, 885, 906` (`os.ReadFile`) runs at compile time
for `import "./mod"` and `embed "*.txt"`. There is no abstraction
(e.g. `fs.FS`), so embedders cannot virtualise the import root,
making it impossible to compile rumo bytecode in a sandboxed
environment (or in browser/wasi where `os.ReadFile` is restricted).

- **Fix:** plumb an `fs.FS` (or interface{Open(string)(io.Reader,…)})
  through `Compiler` and the `Script` API; use `os.DirFS(importDir)`
  by default.

---

## 6. API & maintenance smells

### 6.1 `BuiltinFunction.Equals` always returns `false` &nbsp; **LOW**

`vm/objects.go:364-366`. Two references to the same builtin cannot
compare equal, breaking deduplication and user-script identity
checks (`if my_fn == other_fn { … }`).

### 6.2 `BuiltinFunction.Copy` drops the `Name` &nbsp; **LOW**

`vm/objects.go:358-360`:

```go
func (o *BuiltinFunction) Copy() Object {
    return &BuiltinFunction{Value: o.Value}
}
```

`Name` is essential for marshaling (`MarshalObject` `_builtinFunction`)
and for the type-name string. After a `Copy()`, marshaling produces
a builtin with name "" that can't be unmarshalled.

### 6.3 `unsafe.Pointer` arithmetic in `Program.Equals` &nbsp; **LOW**

`script.go:534-538`:

```go
if uintptr(unsafe.Pointer(first)) > uintptr(unsafe.Pointer(second)) {
    first, second = second, first
}
```

Pointer ordering for deadlock avoidance is not portable across
GC moves or non-amd64 architectures (technically the runtime allows
moving stack-allocated objects). Use a per-Program unique ID
(`atomic.Int64` counter) and compare those.

### 6.4 `Variable.Object()` returns the live VM object &nbsp; **MED**

`variable.go:130`:

```go
func (v *Variable) Object() vm.Object { return v.value }
```

Comment says "returned Object is a copy of an actual Object used in
the script" but the implementation returns the same pointer. An
embedder that mutates the returned `*Map` mutates the script's
internal state behind its back.

- **Fix:** call `v.value.Copy()` (deep) here, or change the doc.

### 6.5 `Modules()` / `Exports()` use `sync.Once` but recompute on test runs that swap modules &nbsp; **LOW**

The lazy init in `rumo.go:44-58` is correct for a single binary, but
embedders that add custom modules after first use will silently miss
them because the cache is computed once.

- **Fix:** document that registering modules must precede the first
  `Modules()` call, or keep the map mutable (and accept the locking
  cost).

### 6.6 `init()`-driven builtin registration spans 3 files &nbsp; **LOW**

`builtins.go`, `routinevm.go` (`go`/`chan`/`cancel`),
`builtins_new.go` are all written into `builtinFuncs` via
package-level `init()`. Toolchains that compile the package without
all three files (build tag combos) end up with mismatched indices —
see 5.10.

---

## 7. Quick wins (next 1-week PRs)

1. Replace CRC64 with SHA-256; bump `FormatVersion`.
2. Fix `OpImmutable` to copy.
3. Lower default `MaxStringLen`/`MaxBytesLen`/`MaxAllocs`.
4. Route every `DefaultConfig.Max*` reference through `*VM.config`.
5. Make `range()` lazy.
6. Add cycle-detection to `Equals`/`Copy`/`String`/`ToInterface`.
7. Snapshot `Bytes` in `Iterate`; make `String.runeStr` thread-safe.
8. Plumb `fs.FS` into `Compiler`.
9. Add `js/wasm` & `wasip1/wasm` to the build matrix; add `//go:build`
   guards on `std/os` privileges.
10. Introduce a `Permissions` struct on `vm.Config` and gate `os.exit`,
    `os.exec`, `os.start_process`, `os.setenv`, `native`, file-open
    behind it.