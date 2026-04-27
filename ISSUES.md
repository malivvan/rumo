# Architectural / Systemic Issues

> Findings from a security-, correctness-, and consolidation-focused audit of
> the `github.com/malivvan/rumo` package and its `vm` and `std` subpackages.
> Issues that are already enumerated in `KNOWN_ISSUES.md` are intentionally
> omitted. Severity tags: **CRIT** > **HIGH** > **MED** > **LOW**.

## 1. Security

### 1.1 Default permissions allow everything &nbsp; **HIGH**

`script.go:62` &mdash; `Script.permissions` is a zero-value `vm.Permissions`
struct (every `Deny*` field is `false`). Combined with `KNOWN_ISSUES.md` 2.2
(unbounded `MaxStringLen`/`MaxBytesLen`/`MaxAllocs`), an embedder that does
nothing more than `rumo.NewScript(src).Run()` exposes file I/O, env mutation,
process exec, `os.exit`, and `os.chdir` to untrusted scripts. `vvm.go:18`
even documents this:

```go
// Permissions ... The zero value (all fields false) permits
// every operation, preserving backward-compatible behaviour.
```

`os.exit(1)` from a script will tear the host process down. This is the
same shape of issue as `KNOWN_ISSUES.md` 2.2 but for *capabilities* rather
than *resources*; the same fix shape applies (deny-by-default with an
`Unrestricted()` opt-in helper).

### 1.2 `rand` module wraps `math/rand`, not `crypto/rand` &nbsp; **HIGH**

`std/rand/rand.go:5,30` &mdash; the module imports `math/rand` and exposes
`rand.read(buffer)` which delegates to `math/rand.Read` (deprecated since
Go 1.20). The API surface deliberately mirrors Go's `crypto/rand`
(`Read([]byte) (int, error)` / `Int()` / `intn(n)`), which makes it very
easy for script authors to mistake the function for a CSPRNG and use it to
mint session tokens, IDs, secrets, JWTs, etc. The output is a deterministic
PRNG and is not suitable for any security-sensitive purpose.

- **Fix:** rename to `prand`/`pseudorandom`, or add a parallel module
  `crand` backed by `crypto/rand`, and document loudly that `rand` is
  non-cryptographic.

### 1.3 JSON decoder has no maximum nesting depth &nbsp; **MED**

`std/json/scanner.go:129` &mdash; `pushParseState` appends to
`scan.parseState` unbounded. `decode.go:91 value()` then recurses through
`array()` / `object()` for every level of nesting, so a payload like
`"[[[[..."` of 100 k repetitions exhausts the goroutine stack and
panics. Go's standard `encoding/json` caps nesting at 10 000.

- **Fix:** clamp `len(parseState)` in the scanner's begin-array /
  begin-object handlers.

### 1.4 JSON encoder has no cycle detection &nbsp; **MED**

`std/json/encode.go:128 Encode` recursively walks `*Array`, `*Map`,
`*ImmutableArray`, `*ImmutableMap` without a visited set. A self-referencing
map (`m := {}; m["self"] = m`) infinite-loops the encoder until the
goroutine stack overflows. Note that all the *other* container methods
(`String`, `Equals`, `Copy`) gained cycle detection (see
`vm/objects.go:2037`) &mdash; encoding was simply missed.

### 1.5 Native FFI relies on uintptr-converted slice/string pointers &nbsp; **HIGH** (`-tags native` only)

`vm/native.go:528-549` &mdash; for `NativePtr` and `NativeBytes` arguments,
the binding converts a slice or string head pointer to `uintptr` *before*
the call:

```go
return reflect.ValueOf(uintptr(unsafe.Pointer(&v.Value[0]))), v, nil
```

It then attempts to keep the backing object alive via a `keepAlive` slice
ending with `_ = keepAlive`. That trailing assignment is **not**
`runtime.KeepAlive`; the Go compiler can elide a discarded read. The Go
spec explicitly states that uintptr-converted pointers do not keep the
backing object reachable. Today's non-moving GC may free the object after
the conversion but before the call returns, leaving the C function with a
dangling pointer. Under any future moving GC the object can also relocate.

- **Fix:** pass `unsafe.Pointer` (not `uintptr`) into `reflect.ValueOf` so
  the GC tracks it, and use real `runtime.KeepAlive(v)` after
  `fnValue.Call`.

### 1.6 Native allow-list does not canonicalise paths &nbsp; **LOW** (`-tags native` only)

`vm/native.go:34 AllowNativePath` is exact-string match.
`AllowNativePath("/usr/lib/libfoo.so")` does not approve
`/usr/lib/./libfoo.so`, `libfoo.so` (resolved against `LD_LIBRARY_PATH`),
or a symlink path that resolves to the same inode. This is a correctness
hazard rather than a privilege-escalation hazard, because non-matching
paths fail closed; but it makes the allow-list painful to use correctly.

- **Fix:** canonicalise with `filepath.Abs` + `filepath.EvalSymlinks` on
  both register and check.

### 1.7 Native loader leaks dlopen handle on VM teardown &nbsp; **LOW** (`-tags native` only)

`vm/native.go:309` caches the handle in `o.handle` and only releases it
via the user-callable `close()` member. There is no Finalize / Close hook
called when the VM exits or when the constant pool is GCed, so a
long-running embedder that loads a different native lib per Program leaks
one handle per Program. `Native.Copy()` (line 254) drops the cache, but
that just means a fresh dlopen on the *next* call; it does not close the
prior handle.

### 1.8 `ContextKey` is an exported string-typed key with panic-cast users &nbsp; **MED**

`vm/vm.go:18` declares `type ContextKey string` and stores the running
VM under `ContextKey("vm")`. This is exported, so any caller (in this
repo or a third-party module) can construct the same key value &mdash;
the standard-library guidance is to use a private, unexported key type to
make the value un-forgeable.

The bigger problem is that several builtins do an unchecked cast:

```go
v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
```

(`std/fmt/fmt.go:18,28,57`, `std/os/os.go:294`, `vm/routinevm.go:47,129`).
If any of these are invoked from a non-VM context (e.g. a test harness
calling them with `context.Background()` or a future API that batches
them outside the run loop), the cast panics. Compare the safe pattern in
`std/fmt/fmt.go:96 vmConfig`. There are mixed conventions in the
codebase.

- **Fix:** make the key unexported, expose
  `vm.VMFromContext(ctx) (*VM, bool)` and audit all sites; standardise on
  the safe form.

### 1.9 Format string DoS via `%*d` runtime width &nbsp; **LOW**

`vm/formatter.go:1184,1214` enforces `MaxFormatWidth` against the
*format-string-parsed* width but `intFromArg` (line 1009) supplies a
runtime width through `%*d`/`%.*f`. The only guard there is
`tooLarge` (1 MB) so a `printf("%*d", 1_000_000, 0)` allocates ~1 MB per
call. In a tight loop this is a memory amplifier.

### 1.10 REPL line buffer has no size cap &nbsp; **LOW**

`rumo.go:193 readLine.ReadLine` calls `bufio.Reader.ReadBytes('\n')` with
no maximum. A line of 4 GB exhausts memory before the parser ever runs.
This is the fallback path used when the `readline` build tag is not set
(otherwise `readline.go` is used).

---

## 2. Correctness

### 2.1 `BinaryOp` and `module.Func` helpers ignore the per-VM `Config` &nbsp; **MED**

The string/bytes size guards inside `vm/objects.go:499 (Bytes), 1770,
1776 (String)` consult `DefaultConfig.MaxStringLen` / `.MaxBytesLen`
directly. `vm/module/builtin.go` does the same in **15 separate
helpers** (lines 311, 329, 347, 374, 652, 677, 706, 786, 828, 888, 974,
1059, 1186, 1214, 1239). The `Format` builtin and `std/text` use the
correct `vm.ConfigFromContext(ctx)` helper, but anything reaching Go
through `module.Func(...)` falls back to the package default. Net effect:
a `Script.SetMaxStringLen(1<<10)` is silently bypassed by, e.g.,
`text.repeat`, `os.read_file`, or any of the dozens of wrappers that hit
these code paths.

- **Fix:** thread the VM config through the helper signatures, or have
  every wrapper re-read the limit from the context (as `Format` does).

### 2.2 `ImmutableArray.copyWithMemo` returns `*Array`; `ImmutableMap.copyWithMemo` returns `*Map` &nbsp; **MED**

`vm/objects.go:1106` and `vm/objects.go:1237` &mdash; both immutable
container types deep-copy themselves into the *mutable* variant:

```go
c := &Array{Value: make([]Object, len(o.Value))}    // ImmutableArray
c := &Map{Value: make(map[string]Object, len(o.Value))} // ImmutableMap
```

This breaks immutability: `copy(immutable_array)` quietly returns a
mutable array. Any code path that relies on
"immutable-in-≡-immutable-out" (e.g. caching module exports, returning
constants to scripts) becomes a vector for unintended mutation if the
caller stores the result and later mutates it.

- **Fix:** return same-kind containers, deep-copying elements as today.

### 2.3 JSON encoder silently drops unknown types &nbsp; **MED**

`std/json/encode.go:289`:

```go
default:
    // unknown type: ignore
}
```

The encoder has cases for `Int`, `Int8`, `Int16` but **not `Int32`**,
nor `Error`, `RangeObject`, `StructInstance`, or `UserType`. Encoding
`[1, int32(2), 3]` produces `[1,,3]` (invalid JSON); encoding
`{"err": error("oops")}` produces `{"err":}`. Should return an error,
and at minimum should add cases for Int32 and StructInstance (since the
language has user-visible struct types).

### 2.4 JSON decoder loses precision on integers &nbsp; **MED**

`std/json/decode.go:223` decodes every literal as `*vm.Float64`, even
inputs without a decimal point. JSON integers above 2⁵³ silently lose
precision, so `json.encode(int(9007199254740993))` round-trips to
`9007199254740992`. Aside from precision, the call is also
`strconv.ParseFloat(s, 10)` &mdash; the second argument is `bitSize`
(only 32 or 64 are documented; 10 is treated as 64 by today's stdlib but
is explicitly outside the documented domain).

### 2.5 `fmt.printf("...")` prints a quoted string &nbsp; **MED**

`std/fmt/fmt.go:43`:

```go
if numArgs == 1 {
    fmt.Fprint(v.Out, format)   // *vm.String → Quote
    return nil, nil
}
```

`Object.String()` on `*vm.String` returns `strconv.Quote(o.Value)`, so
`fmt.printf("hello")` writes `"hello"` (with quotes). Should pass
`format.Value`. Same shape of issue is avoided by `fmtPrintln` which
goes through `getPrintArgs` &rarr; `vm.ToString`.

### 2.6 VM cancellation does not reach `CallFunc` clones &nbsp; **MED**

`vm/callfunc.go:19` does:

```go
clone := parentVM.ShallowClone()
return clone.RunCompiled(cfn, args...)
```

It never registers the clone with the parent via `parentVM.addChild`. So
when the parent's `Abort()` walks `childCtl.vmMap` (vm.go:283), the
clone is invisible &mdash; aborting the parent does not abort the
callback. `UserType.callFunc` (the type-checked function wrapper at
`vm/types.go:242`) goes through `CallFunc`, so any user-typed callback
is uncancellable mid-flight.

`routinevm.go:82` is the working pattern (it does `addChild`); apply the
same to `CallFunc`.

### 2.7 VM run loop never inspects `ctx.Done()` &nbsp; **MED**

`vm/vm.go:388 run()` only looks at `v.aborting`. A pure-bytecode hot
loop (e.g. `for { x = x + 1 }`) keeps running until something else
calls `Abort`. `Program.RunContext` does call `Abort` on `ctx.Done`, so
the top-level case is covered, but anything else that drives a VM
directly (the REPL's per-line VM in `rumo.go:294`, and the `CallFunc`
clones above) is not. A periodic `if v.ctx.Err() != nil { v.Abort() }`
inside the dispatch loop &mdash; or every N opcodes &mdash; would close
the gap consistently.

### 2.8 `OpClosure` error message uses a nil receiver &nbsp; **LOW**

`vm/vm.go:1185-1188`:

```go
fn, ok := v.constants[constIndex].(*CompiledFunction)
if !ok {
    v.err = fmt.Errorf("not function: %s", fn.TypeName()) // fn is *(nil) here
```

Works only because `(*CompiledFunction).TypeName` is a method on a
pointer that doesn't dereference. The error message is therefore
permanently `"not function: compiled-function"` regardless of the
constant's actual type. Use `v.constants[constIndex].TypeName()`.

### 2.9 Tail-call detection reads past the instruction stream &nbsp; **LOW**

`vm/vm.go:931-934` does:

```go
nextOp := v.curInsts[v.ip+1]
if nextOp == parser.OpReturn ||
    (nextOp == parser.OpPop &&
        parser.OpReturn == v.curInsts[v.ip+2]) {
```

with no bounds check. The compiler always appends `OpReturn`, but a
maliciously-crafted bytecode file deserialised via `Program.Unmarshal`
can omit it, producing an `index out of range` panic that propagates as
a runtime panic instead of a clean error.

### 2.10 `Time.String()` includes the monotonic-clock suffix &nbsp; **LOW**

`vm/objects.go:1903` uses Go's default `time.Time.String()`, which can
include `m=±N.NNNs` for any time captured by `time.Now()`. The
suffix is non-deterministic (changes between runs), surprising in script
output, and breaks naive equality-by-format. Use
`Format(time.RFC3339Nano)` or strip via `.Round(0)` before formatting.

### 2.11 `BuiltinFunction.Equals` matches by name only &nbsp; **MED**

`vm/objects.go:440` &mdash; two `*BuiltinFunction` instances with the
same `Name` compare equal even if their `Value` differs. This is
internally consistent with the marshal-by-name design (a deserialised
builtin is replaced by the same-named entry in `builtinFuncs`), but it
also means a host-injected builtin named `len` is "equal" to the stdlib
`len`. Combined with the dedup logic in `vm/bytecode.go`, a host that
registers a per-script `BuiltinFunction{Name:"len"}` may find it silently
collapsed to the global `len`. Document the global-name space and
forbid duplicate registration.

### 2.12 REPL writes runtime errors to stdout &nbsp; **LOW**

`rumo.go:300, 301`:

```go
if err := machine.Run(); err != nil {
    _, _ = fmt.Fprintln(stdout, err.Error())   // ← should be stderr
```

`RunREPL` accepts a `stderr` writer parameter &mdash; runtime errors
should go there.

### 2.13 REPL `addPrints` synthesises AST nodes without source positions &nbsp; **LOW**

`rumo.go:308 addPrints` builds `parser.CallExpr`/`parser.Ident` nodes
with no `Pos` set. Errors emitted from these nodes carry `parser.NoPos`
and produce useless source locations in REPL diagnostics.

### 2.14 `ToBool` always returns `ok=true`, dropping the default-arg branch &nbsp; **LOW**

`vm/vvm.go:362-366`:

```go
func ToBool(o Object) (v bool, ok bool) {
    ok = true
    v = !o.IsFalsy()
    return
}
```

Callers like `builtinBool` rely on the second return value to fall back
to the optional default arg. Because `ok` is unconditionally true,
`bool(undefined, "default")` ignores the default and returns
`FalseValue`. Either remove the conversion-default branch from
`builtinBool` or special-case `*Undefined` here.

### 2.15 `Modules()` rebuilds and re-deep-copies every call &nbsp; **LOW**

`stdlib.go:46 Modules()` and `Exports()` are documented as recomputing
fresh maps on every call to allow late module registration. Each call
walks `BuiltinModules` and `SourceModules` and (via
`BuiltinModule.AsImmutableMap`) deep-copies each Object. The REPL calls
`Modules()` *per executed REPL line* (rumo.go:287) &mdash; for a normal
session this means N copies of the entire stdlib live until the next GC.
Cache the result with a "did anything change since last call" guard, or
expose a snapshotting helper.

### 2.16 JSON encoded map ordering is non-deterministic &nbsp; **LOW**

`std/json/encode.go:160-195` iterates `*vm.Map` / `*vm.ImmutableMap` via
range over the underlying Go map, so the same input produces different
byte-for-byte outputs across runs. Surprising for hashing, signing,
golden-file tests, content-addressable caches. Sort keys on output.

### 2.17 Compiler's TCO warning trips on incidental `OpCall` patterns &nbsp; **LOW**

`vm/compiler.go:1626 scopeHasTailCallPattern` checks any `OpCall` in
the scope, not just self-recursive ones. A function with a top-level
`defer X()` and an unrelated `f()` that happens to be the last
expression triggers the "tail-call optimisation suppressed" warning even
when `f != self`. Document or scope-narrow.

### 2.18 `compileEmbed` rejects absolute paths but accepts `..` glob escapes &nbsp; **LOW**

`vm/compiler.go:967` rejects absolute patterns, but
`filepath.Join(c.importDir, pattern)` happily resolves `../../etc/*` to
files outside the import root. The matched paths are then read via
`fsReadFile` &mdash; if `importFS` is set this is contained, but in the
default `os.ReadFile` mode the script can include arbitrary host files
in its bytecode at compile time. Apply the same `isPathWithinBase` check
that file imports use.

---

## 3. Performance & scalability

### 3.1 Reading `DefaultConfig` on every primitive op

See 2.1. Every string concatenation and bytes append re-reads
`DefaultConfig.MaxStringLen` from the package global; a per-VM config
would avoid the indirection (and incidentally would also fix the
correctness gap).

### 3.2 No regex cache for `text.re_*` shortcuts

`std/text/text.go:101, 133, 242, 297` recompile the user-supplied
pattern on every call. RE2 compilation cost is non-trivial for complex
patterns. Either cache by pattern string, or document that callers
should use `text.re_compile` for repeated matches (a `Regexp` object is
already returned).

### 3.3 `gatherBuiltinIndices` walks bytecode twice per Marshal

`vm/bytecode.go:45,191` &mdash; `gatherBuiltinIndices` walks every
compiled function's instructions, then `MarshalObject` walks them again
to write them out. Acceptable for typical sizes, but consolidating into
one pass is straightforward and would halve marshal CPU on large
programs.

### 3.4 `unsafe.Pointer`-based cycle detection is GC-fragile

`vm/objects.go:204, 249, 279, 1102, 1126, 1233, 1271, 1610, 1614,
1623, 1626, 1631` and `vm/types.go:537, 568, 600, 612` &mdash; cycle
detection uses `uintptr(unsafe.Pointer(o))` as a map key. This is
correct under Go's current non-moving GC. If Go ever introduces a
moving GC (and the language spec already allows it), the same object
can change uintptr identity mid-walk, breaking the visited set. Use
`map[*Array]struct{}` etc. keyed on the typed pointer instead &mdash;
no `unsafe` import is needed and the resulting code is shorter.

### 3.5 `RemoveDuplicates` hashes every `Bytes` constant unconditionally

`vm/bytecode.go:389` SHA-256s every single `Bytes` constant during
deduplication. For embed-heavy programs (thousands of files of binary
data), this is the dominant compile cost. A first-pass length+head
fingerprint, or a bypass when there's only one Bytes constant, would
save a lot of CPU at compile time.

### 3.6 `delChild`/`addChild` token map grows monotonically

`vm/vm.go:309 nextToken` is a uint64 counter; entries are deleted on
delChild, but a long-lived VM that hosts a high-rate spawn workload
walks the integer space up to 2⁶⁴. Practically infinite, but the
linear-id design rules out reuse, which matters if you ever want to
debug a token to a specific spawn after the fact.

---

## 4. Compatibility / future-proofing

### 4.1 `go.mod` requires `go 1.26.2` &nbsp; **HIGH**

`go.mod:3 go 1.26.2`. Go 1.26 is unreleased as of this audit (April 2026
per the project clock; 1.25 is the latest stable). Any consumer on
released Go fails with `module requires go 1.26.2`. Worse, the codebase
already relies on a 1.26 standard-library feature: `errors.AsType` at
`vm/vm.go:365`:

```go
if e, ok := errors.AsType[ErrPanic](err); ok {
```

`errors.AsType` is the (proposed/in-flight) generics-friendly companion
to `errors.As`. If 1.26 ships without it, the code won't build. Either
relax the toolchain requirement and switch to `errors.As(err, &e)`, or
stage the bump for after 1.26's actual release.

### 4.2 Single hard-coded `FormatVersion` with strict equality &nbsp; **LOW**

`script.go:47` has `FormatVersion uint16 = 5` and the loader
(`script.go:353`) refuses anything other than the *exact* current value.
Combined with the bytecode evolution noted in `KNOWN_ISSUES.md` 1.2,
this means each new VM version invalidates every previously-compiled
artifact. A backward-compatibility ladder (read older versions through a
small migration step) would let users keep their compiled programs
across patch releases.

### 4.3 Type-code allocation is hand-synchronised &nbsp; **LOW**

`vm/encoding.go` uses single-byte type codes (`_undefined=1` …
`_rangeObject=105`) defined as untyped consts, paired with `_typeMap`
and `TypeOfObject`. Adding a new type requires editing three places;
mistakes are silent (`TypeOfObject` returns 0, `MakeObject` returns
nil &rarr; surfaces only at decode time as a generic error). The
`_float=6` / `_float64=6` aliasing has already paid this tax. A
`go:generate` step from a single source-of-truth registry would remove
the drift risk.

### 4.4 `ModuleMap` is not safe for concurrent use &nbsp; **LOW**

`vm/modules.go` &mdash; raw `map[string]Importable` with no
synchronisation. Concurrent reads while another goroutine mutates the
map will panic. In practice modules are configured at startup, but the
API does not enforce that, and `Bytecode.Unmarshal` *reads* the map
while the embedder may still be calling `AddBuiltinModule`. Either
freeze the map at first use, or guard with an `sync.RWMutex`.

### 4.5 `BuiltinModule.Func` panics on unknown signatures &nbsp; **LOW**

`vm/module/builtin.go:200` panics with `unsupported function type` for
any Go signature that is not in the hand-coded type switch. Adding a
stdlib function with a novel signature requires adding a new
`funcXxx` helper, otherwise the package fails at init time. Either
return an error from `NewBuiltin().Func(...)` and let the caller
decide, or fall back to a reflection-based generic adapter for rare
signatures.

---

## 5. Consolidation opportunities

### 5.1 Duplicate `wrapError` implementations

`vm/module/error.go:6 WrapError` (exported) and
`vm/module/module.go:7 wrapError` (unexported) are byte-for-byte
identical. Pick one; have the other delegate.

### 5.2 Container cycle-detection scaffolding repeated five times

Each of `Array`, `ImmutableArray`, `Map`, `ImmutableMap`,
`StructInstance` re-implements `equalsWithVisited`, `copyWithMemo`,
`stringWithVisited`. The bodies differ only in the type of the visited
key and the type assertion. Generic helpers would remove ~300 lines and
guarantee semantic consistency &mdash; in particular, JSON encoding
(see 1.4) could just *use* the existing visited-set helper instead of
re-deriving cycle detection.

### 5.3 Duplicate `Run` / `RunContext` snapshot/exec/writeback in `script.go`

`script.go:428 Run` and `script.go:458 RunContext` duplicate the
RLock-snapshot &rarr; build VM &rarr; run &rarr; Lock-writeback dance.
The differences are only in how they wait (`v.Run()` vs goroutine +
ctx.Done select). Extract a shared helper that takes the wait closure.

### 5.4 Triple frame-build paths in the VM

Compiled-function frame setup is duplicated in `OpCall`
(`vm/vm.go:949+`), the deferred-call dispatch in `runNextDefer`
(`vm/vm.go:1438+`), and `RunCompiled` (`vm/vm.go:210+`). All three set
`v.curFrame.fn`, `v.curFrame.basePointer`, `v.curFrame.defers`,
`v.curFrame.inDefer`, `v.curInsts`, `v.ip`, `v.framesIndex`,
`v.sp`. Subtle drift is already visible: `OpCall` checks
`v.framesIndex >= v.config.MaxFrames` *after* setting `v.curFrame.ip`;
`runNextDefer` checks it *before*. A `pushFrame(callee, basePointer)`
helper would centralise the invariant.

### 5.5 Predicate-builtins repeat the same 9-line body

`vm/builtins.go` and `vm/builtins_new.go` together contain ~20 functions
of the form:

```go
func builtinIsXxx(_ context.Context, args ...Object) (Object, error) {
    if len(args) != 1 { return nil, ErrWrongNumArguments }
    if _, ok := args[0].(*Xxx); ok { return TrueValue, nil }
    return FalseValue, nil
}
```

A single generic registrar would eliminate the boilerplate. While
collapsing them, fix the inconsistency noted in
`vm/builtins_new.go:335-338` (where `is_uint32` got its own function but
`is_int64` aliases `is_int`).

### 5.6 Per-numeric-type method boilerplate in `vm/numeric.go`

Each of `Byte`, `Int8`, `Uint8`, `Int16`, `Uint16`, `Int32`, `Uint`,
`Uint64` declares `String`, `TypeName`, `Copy`, `IsFalsy`, `Equals`,
`BinaryOp` with the same shape. A generic
`type numeric[T integer] struct { ... }` (or a code-generation step
similar to 4.3) would cut ~150 lines and make adding `Int24` etc.
trivial. The shared `signedIntBinaryOp` / `unsignedIntBinaryOp` already
exist for the math part; only the wrapper plumbing is duplicated.

### 5.7 `ContextKey("vm")` cast pattern

`std/fmt/fmt.go`, `std/os/os.go`, `vm/routinevm.go`, `vm/callfunc.go`,
`vm/builtins.go`, `vm/vvm.go` all do their own variant of
`ctx.Value(ContextKey("vm")).(*VM)` &mdash; some panic on miss, some
return zero values, some return nil-checks. Centralise:

```go
func VMFromContext(ctx context.Context) (*VM, bool)
```

… and audit the call sites. Combined with 1.8, this also fixes the
exported-key concern.

### 5.8 Permission-gate boilerplate in `std/os`

Almost every function in `std/os/os.go` opens with the same five-line
pattern:

```go
if permsFromCtx(ctx).DenyXxx { return nil, vm.ErrNotPermitted }
if len(args) != N { return nil, vm.ErrWrongNumArguments }
```

A `gateFn(perm Permission, fn CallableFunc) CallableFunc` wrapper plus
helpers for arg-count would centralise the policy and make permissions
declarative rather than imperative &mdash; which also makes it easier
to verify by inspection that *every* privileged operation is gated.

### 5.9 `WrapError` returns `vm.TrueValue` for nil

`vm/module/error.go:7-10` and the unexported sibling both return
`vm.TrueValue` when `err == nil`. This is consumed by stdlib helpers
that wrap zero-or-one-error Go functions, and existing scripts test
`if file.close() == true`. It's a sensible convention but inverts
language intuition (an "error or success" Go function returning
`TrueValue` for "no error"). Document it next to `Permissions` and the
other surprising-but-deliberate behaviours, or shift to returning
`UndefinedValue` on nil.

