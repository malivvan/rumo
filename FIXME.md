# FIXME — root `rumo` package + `cmd` architectural review

Systemic issues in the root `rumo` package (public API: `Script`, `Program`, `Variable`, REPL, compile/run entry points) and the `cmd` CLI binary, with focus on cross-package interactions, embedding API correctness, and how the application exposes itself as a whole.

Issues already documented in sub-package reviews (`vm/FIXME.md`, `std/FIXME.md`) are **not** repeated here. This review focuses on **emergent** problems at the integration boundary — where the root package, the CLI, the VM, and the stdlib modules meet.

Severity: **critical** (will cause crashes, data races, or incorrect behaviour under normal use), **high** (likely to cause subtle bugs or silently break embedding contracts), **medium** (performance/maintainability/usability concern), **low** (minor improvement).

---

## Executive Summary

Three systemic themes dominate:

1. **Eager global initialization** — The `Modules` and `Exports` package-level vars force all stdlib modules to be initialized at Go import time. Every binary that imports `rumo` pays the full cost regardless of which modules are used.

2. **`Program`-level locking blocks the entire execution** — `Program.Run()` and `RunContext()` hold a write lock for the entire script execution lifetime, making the embedding API effectively single-threaded and causing `Get()`/`Set()`/`IsDefined()` to block until the script finishes.

3. **CLI binary lacks signal handling, argument forwarding, and context propagation** — Ctrl-C kills the process without cleanup, scripts cannot receive arguments, and the REPL ignores context cancellation.

---

## 1. Concurrency / Thread Safety

### 1.1 `Program.Run()` / `RunContext()` hold write lock for entire execution — critical

```go
// script.go:329-335
func (p *Program) Run() error {
    p.lock.Lock()
    defer p.lock.Unlock()
    v := vm.NewVM(context.Background(), p.bytecode, p.globals, p.maxAllocs)
    return v.Run()
}
```

Both `Run()` and `RunContext()` acquire `p.lock.Lock()` (write lock) before creating the VM and hold it until execution completes. This means:

- `p.Get()`, `p.Set()`, `p.IsDefined()`, `p.GetAll()` all acquire `p.lock.RLock()` — they block for the **entire duration** of the script.
- For long-running or infinite scripts (event loops, servers), these accessors are effectively deadlocked.
- `RunContext()` spawns a goroutine (script.go:344) that inherits the write lock via the deferred unlock — the lock is held until the goroutine finishes, not until `RunContext` returns.
- `Program.Clone()` also takes a write lock, so cloning a running program blocks.

The doc comment on `Clone()` says _"Cloned copies are safe for concurrent use"_ but this only works because `Run()` serializes everything — it's correctness through exclusion, not through safe sharing.

**Fix direction:** Move `globals` out of `Program` into a per-run context. Let `Run()` create its own copy of the globals slice (like `Clone()` does), allowing `Program` to be read concurrently during execution. The lock should protect the `Program` struct fields during compilation/mutation, not during execution.

---

### 1.2 `Program.Clone()` shares mutable objects in globals — critical

```go
// script.go:370-376
func (p *Program) Clone() *Program {
    clone := &Program{
        globalIndices: p.globalIndices,  // shared map reference
        bytecode:      p.bytecode,       // shared pointer
        globals:       make([]vm.Object, len(p.globals)),
    }
    for idx, g := range p.globals {
        if g != nil {
            clone.globals[idx] = g  // copies the pointer, not the object
        }
    }
    return clone
}
```

`Clone()` creates a new `globals` slice but copies object **pointers**, not the objects themselves. If an embedder sets a global to a `*vm.Map` or `*vm.Array` (via `s.Add("cfg", myMap)`), both the original and clone share the same mutable Go map/slice. Concurrent execution of the original and clone races on the shared mutable object.

Additionally, `globalIndices` map and `bytecode` pointer are shared by reference. `bytecode.Constants` contains `*CompiledFunction` objects whose `Free` fields are mutable (as documented in `vm/FIXME.md` #2), so concurrent execution of cloned programs can race on shared constant closures.

**Impact:** Data races in the primary intended use case (`Compile` → `Clone` × N → `Run` concurrently).
**Fix direction:** Deep-copy globals (call `.Copy()` on each object), and clone the `bytecode.Constants` slice with cloned `CompiledFunction.Free` cells.

---

### 1.3 `Program.Equals()` doesn't lock `other` — high

```go
// script.go:452-479
func (p *Program) Equals(other *Program) bool {
    p.lock.RLock()
    defer p.lock.RUnlock()
    // reads other.globalIndices, other.globals, other.bytecode — no lock
```

`Equals` acquires a read lock on `p` but accesses `other` fields without any synchronisation. If another goroutine is running or modifying `other`, this is a data race.

**Fix direction:** Acquire `other.lock.RLock()` as well (with consistent ordering to avoid deadlock).

---

### 1.4 `Program.Unmarshal` is not atomic on failure — medium

```go
// script.go:264-284
n, p.globalIndices, err = codec.UnmarshalMap(...)  // partially written
n, p.globals, err = codec.UnmarshalSlice(...)       // partially written
n, p.maxAllocs, err = codec.UnmarshalInt64(...)     // partially written
p.bytecode = &vm.Bytecode{}
err = p.bytecode.Unmarshal(body[n:], Modules)       // may fail
```

Fields are overwritten sequentially under the lock. If deserialization fails partway (e.g., corrupt bytecode section), the `Program` is left in a partially-initialized state with some fields from the new data and some from the old (or zero). Callers who catch the error cannot safely reuse the `Program` object.

**Fix direction:** Unmarshal into local variables first, then assign all at once after validation succeeds. Or document that `Program` is unusable after a failed `Unmarshal`.

---

## 2. Cross-Package Integration

### 2.1 Eager init of `Modules` and `Exports` forces all stdlib into every binary — critical

```go
// rumo.go:41-43
var Modules = GetModuleMap(AllModuleNames()...)
var Exports = GetExportMap(AllModuleNames()...)
```

These package-level variables are initialized at import time via `init()`. `GetModuleMap` calls `mod.Objects()` on every `BuiltinModule`, which triggers `init()` in every stdlib package (`fmt`, `json`, etc.). This means:

- **Binary size:** Every binary that imports `rumo` (even for a one-line script using only `math`) links and all their transitive dependencies.

**Fix direction:** Make `Modules` and `Exports` lazy — either compute on first access (`sync.Once`) or remove the global entirely and require callers to explicitly construct the module map. Consider using Go build tags to exclude heavy modules on constrained platforms.

**Performance impact (see also §5.2 below):**
- **Startup time:** All module `init()` functions run at program start, including those for unused modules.
- **Memory:** All module objects (function tables, constant maps) are allocated at program start.

---

### 2.2 `NewVM` defaults `In`, `Out`, `Args` to OS globals — never overridden — high

```go
// vm/vm.go:84-87
v := &VM{
    In:  os.Stdin,
    Out: os.Stdout,
    Args: os.Args,
}
```

The VM's I/O streams default to `os.Stdin` and `os.Stdout`. The root package's `compileSrc()` → `NewScript()` → `Compile()` → `Program.Run()` → `NewVM()` chain **never overrides** these defaults. Even `RunREPL`, which accepts custom `in`/`out` parameters, creates the per-line VM via `vm.NewVM(ctx, bytecode, globals, -1)` without setting `In`/`Out` on the resulting VM (rumo.go:264). The custom streams are only used for readline I/O, not for script-level I/O (e.g., `fmt.print` retrieves the VM's `Out` via context).

`Args` defaults to `os.Args`, which leaks the host binary's argument list into every rumo script. A script calling `args()` or a `cli` module that falls back to `os.Args` will see the host's arguments, not the script's.

**Fix direction:** (a) Add `Script.SetIn()`/`Script.SetOut()`/`Script.SetArgs()` methods that propagate to the VM. (b) In `RunREPL`, set `machine.In` and `machine.Out` to the rl streams. (c) In `cmd/main.go`, set `Args` to the script-relevant subset.

**Security note (see also §6.1 below):** The default `vm.Args = os.Args` means scripts can inspect the host binary's full argument list, including flags like `--config=/path/to/secrets.yaml` or environment-derived arguments. In sandboxed or multi-tenant environments, this leaks host configuration to untrusted scripts. Default `vm.Args` to an empty slice (or just `[]string{scriptName}`). Populate explicitly from the CLI with only script-relevant arguments.

> **Related:** §3.2 (CLI cannot forward arguments to scripts).

---

### 2.3 `Program.Unmarshal` hardcoded to global `Modules` — high

```go
// script.go:279
err = p.bytecode.Unmarshal(body[n:], Modules)
```

`Unmarshal` always passes the global `Modules` variable (which, per §2.1, contains all stdlib modules). An embedder who uses custom modules (via `script.SetImports()`) has no way to inject them into deserialization. `Unmarshal` is a method on `Program`, but there's no way to set the module map before calling it. Deserialized programs that reference custom builtin modules will fail with `"user function not decodable"` errors.

**Fix direction:** Add an `UnmarshalWithModules(b []byte, modules *vm.ModuleMap)` method, or accept modules as a parameter. Fall back to global `Modules` if nil.

**Security note (see also §6.2 below):** When a compiled `.out` file is loaded via `RunCompiled`, it inherits the global `Modules` map — which includes `shell` (arbitrary command execution). The compiled format does not record which modules the original script was compiled with. An attacker could compile a benign-looking script, then run it in an environment where `shell` is available, gaining capabilities not present during compilation. Record the required module names in the compiled format. At load time, validate that only the recorded modules are provided.

---

### 2.4 Source modules imported in REPL run in isolated VMs — medium

```go
// rumo.go:187-194
} else if _, ok := SourceModules[name]; ok {
    s := NewScript([]byte(fmt.Sprintf(`__result__ := import("%s")`, name)))
    s.SetImports(Modules)
    p, err := s.RunContext(ctx)
    if err == nil {
        sym := symbolTable.Define(name)
        globals[sym.Index] = p.Get("__result__").Object()
    }
}
```

For each source module imported globally in the REPL, a separate `Script` is created and run in a fresh VM. The resulting export object is captured but:

- The module's internal state (closures, variables) lives in the now-discarded VM. If the exported value is a closure that references module-level variables, those variables are orphaned — modifying them from the REPL won't work.
- Any errors during source module import are silently swallowed (the `if err == nil` check discards the error with no feedback to the user).
- Each module import creates and destroys a full VM, which is wasteful for REPL startup with many modules.

**Fix direction:** Either (a) compile and execute source modules within the REPL's own VM/compiler pipeline, or (b) at minimum, log/print errors when module import fails.

---

### 2.5 `compileSrc` unconditionally enables file imports — medium

```go
// rumo.go:277
s.EnableFileImport(true)
```

When invoked via the CLI (`runFile` → `CompileAndRun` → `compileSrc`), file imports are always enabled. While the import sandbox (`importBase` / `isPathWithinBase()`) restricts traversal outside the import root, this means any `.rumo` file within the root directory tree is importable. For untrusted script execution, this may be more permissive than desired.

The embedding API (`Script`) defaults to `enableFileImport: false`, which is secure-by-default. But `compileSrc` is also used by `CompileOnly` (the `build` command), meaning compiled `.out` files were created with file imports enabled. When deserialized and run, the compiled modules are already baked in, but the implicit permission model is inconsistent.

**Fix direction:** Document the security model. Consider making file import enablement configurable in the CLI (e.g., `--no-file-import` flag). For sandboxed environments, add an option to disable at the `Program` level too.

---

## 3. CLI Correctness & Robustness

### 3.1 No signal handling — Ctrl-C kills without cleanup — critical

```go
// cmd/main.go:16
func main() {
    fmt.Println(rumo.AllModuleNames())
    os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
```

The CLI binary passes `context.Background()` to `run()`. There is no SIGINT/SIGTERM handler that cancels the context. When a rumo script is running:

- **Any script:** Child routines (`start()`) are not aborted, goroutines leak into the ether.

The `os.Exit(run(...))` call bypasses all deferred functions, compounding the problem.

**Affected:** `cmd/main.go:16`, every script with long-running operations.

**Fix direction:** Install a signal handler that cancels a context:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
os.Exit(run(ctx, ...))
```

Replace `os.Exit` with a return from `main` (or defer the exit) so cleanup runs. This propagates through `CompileAndRun` → `RunContext` → `v.Abort()` → child routine abort chain.

---

### 3.2 CLI cannot forward arguments to scripts — high

```go
// cmd/main.go:33-37
case "run":
    if len(args) != 2 {
        _, _ = fmt.Fprintln(errOut, "usage: rumo run <input_file>")
        return 1
    }
    return runFile(ctx, args[1], errOut)
```

The `run` subcommand accepts exactly one argument (the file). There is no mechanism to pass additional arguments to the script. The default case (`args[0]` as file) has the same limitation.

Combined with §2.2 (`vm.Args` defaults to `os.Args`), a rumo script sees the **host binary's** arguments (e.g., `["rumo", "run", "script.rumo"]`), not script-specific arguments. There is also no `--` separator to distinguish rumo CLI flags from script arguments.

**Fix direction:** Accept `rumo run <file> [-- script_args...]` or `rumo <file> [script_args...]`. Pass the script arguments to the VM via `vm.Args` or a dedicated mechanism.

---

### 3.3 Debug output left in `main()` — high

```go
// cmd/main.go:15
fmt.Println(rumo.AllModuleNames())
```

Every invocation of the `rumo` CLI prints the list of all module names to stdout before doing anything else. This is debug output that was left in the production binary. It pollutes the output of every command, including `rumo run`, `rumo build`, and the REPL, and can break piped output workflows.

**Fix direction:** Remove the line.

---

### 3.4 `CompileAndRun` error message says "compiling" for runtime errors — medium

```go
// cmd/main.go:78-80
if err := rumo.CompileAndRun(ctx, data, inputFile); err != nil {
    _, _ = fmt.Fprintf(errOut, "Error compiling %s: %s\n", inputFile, err.Error())
    return 1
}
```

`CompileAndRun` can fail during either compilation or execution, but the error message always says "Error compiling". A user debugging a runtime panic will see "Error compiling script.rumo: Runtime Panic: ..." which is confusing.

**Fix direction:** Either split the error message based on the error phase, or use a neutral message like `"Error: %s: %s"`.

---

### 3.5 Build output path inconsistent between CLI and public API — medium

```go
// cmd/main.go:87
outputFile = filepath.Base(inputFile) + ".out"  // "script.rumo.out"

// rumo.go:54
outputFile = basename(inputFile) + ".out"        // "script.out"
```

`buildFile` in the CLI uses `filepath.Base` (keeps the `.rumo` extension), while `CompileOnly` uses `basename` (strips the extension). Since `buildFile` always provides the output path, the `CompileOnly` fallback is never reached from the CLI. But a Go embedder calling `CompileOnly("source.rumo", "")` gets `"source.out"`, while the CLI produces `"source.rumo.out"`. The behaviour is inconsistent.

**Fix direction:** Unify on one approach. `basename` (stripping the extension) is more intuitive: `script.rumo` → `script.out`.

---

### 3.6 REPL ignores context cancellation — medium

```go
// rumo.go:236-270
for {
    // ... never checks ctx.Done()
    line, readErr := rl.ReadLine()
```

The REPL loop never checks `ctx.Done()`. If the context is cancelled (via timeout, signal, or programmatic cancellation), the REPL continues running until `ReadLine()` happens to fail. For non-interactive input (piped), the loop terminates on EOF, but for interactive terminals the context cancellation is silently ignored.

**Fix direction:** Check `ctx.Err()` at the top of each loop iteration:

```go
for {
    if ctx.Err() != nil {
        return
    }
    // ...
}
```

---

## 4. Binary Format / Compatibility

### 4.1 No version field in the binary format — high

```go
// script.go:21
const Magic = "VVC\x00"
// format: [4]MAGIC [4]SIZE [N]DATA [8]CRC64(ECMA)
```

The compiled binary format has a 4-byte magic number but no version field. If the bytecode format changes (new opcodes, different encoding scheme, different constant types), old compiled files will either:

- Pass the CRC check but produce garbage when decoded (wrong field offsets).
- Fail with cryptic unmarshal errors ("expected *ObjectPtr", "unsupported type", etc.).

There is no way to distinguish "incompatible version" from "corrupt file".

**Fix direction:** Reserve one of the magic bytes as a format version (e.g., `"VVC\x01"` for version 1), or add a version field after the magic. Check the version during `Unmarshal` and return a clear error like `"incompatible bytecode version: got 2, want 1"`.

---

### 4.2 Deserialized `BuiltinFunction` objects are non-functional — high

```go
// vm/encoding.go:361-366
case _builtinFunction:
    n, o.(*BuiltinFunction).Name, err = codec.UnmarshalString(n, b)
    // Name is restored, but Value (the actual Go function) is NOT
```

`MarshalObject` serializes only the `Name` of a `BuiltinFunction`. `UnmarshalObject` restores the `Name` but not the `Value` (the Go function pointer). `fixDecodedObject` in `bytecode.go` handles module-level `ImmutableMap` restoration (via `modules.GetBuiltinModule`), but individual `BuiltinFunction` objects in constants are left with `Value == nil`.

If a compiled program stores a reference to a builtin in a constant (e.g., `f := append; export {f: f}`), the deserialized program will panic with a nil-pointer dereference when calling the function.

**Fix direction:** In `fixDecodedObject`, resolve `BuiltinFunction` by name against the known builtin function registry (`GetAllBuiltinFunctions()`). Return an error if the name is not found.

---

## 5. Performance

### 5.1 REPL constants accumulate without bounds — medium

```go
// rumo.go:235,269
var constants []vm.Object
for {
    // ...
    c := vm.NewCompiler(srcFile, symbolTable, constants, Modules, nil)
    // ...
    constants = bytecode.Constants  // grows every iteration
}
```

The REPL maintains a `constants` slice that is passed to each new compiler and grows with every evaluated line. There is no deduplication (unlike `Bytecode.RemoveDuplicates()` which is called during `Script.Compile()`), no compaction, and no upper bound.

In a long REPL session (especially one that evaluates many string/numeric literals), constants grow monotonically, consuming memory that is never reclaimed. Each iteration also scans the entire constants slice during compilation.

**Fix direction:** Periodically call `RemoveDuplicates()` on the accumulated bytecode, or implement a constants table with deduplication at insertion time.

---

### ~~5.2 Eager module initialization — binary size and startup cost~~ → merged into §2.1

Performance and binary-size concerns have been consolidated into §2.1 (Eager init of `Modules` and `Exports`).

---

## 6. Security

### ~~6.1 `vm.Args` leaks host arguments to scripts~~ → merged into §2.2

Security concern has been consolidated into §2.2 (VM defaults `In`/`Out`/`Args` to OS globals).

---

### ~~6.2 Compiled bytecode accepts any module map at load time~~ → merged into §2.3

Security concern has been consolidated into §2.3 (`Unmarshal` hardcoded to global `Modules`).

---

## 7. Maintenance / Code Quality

| # | Issue | Severity |
|---|-------|----------|
| ~~7.1~~ | ~~Debug `fmt.Println` in `cmd/main.go:15`~~ — duplicate of §3.3. | — |
| 7.2 | **`AllModuleNames()` returns non-deterministic order** — iterates over Go maps, so module names are in random order each run. Affects REPL help, CLI output, and test stability. Sort the result. | low |
| 7.3 | **`BuiltinModules` and `SourceModules` comment says "source type"** — `stdlib.go:19` comment for `BuiltinModules` says "source type standard library modules" — should say "builtin type". Copy-paste from the `SourceModules` comment. | low |
| 7.4 | **`variable.go` doc comments say "returns 0"** for non-numeric methods — e.g., `Array()` and `Map()` doc comments say "returns 0 if not convertible" — they return `nil`, not `0`. | low |
| 7.5 | **`prepCompile` panics on symbol index mismatch** — `script.go:209-212` uses `panic` for what should be an error return. This is an internal consistency check, but a panic here crashes the embedding application with no recovery. | low |
| 7.6 | **`TestRunCompiledFileExecutesOnlyOnce` is commented out** — `cmd/main_test.go:28-53`. Suggests compiled execution has an unresolved issue. | low |
| 7.7 | **`TestModulesRun` is commented out** — `stdlib_test.go:18-66`. References a removed `os` module. Dead test code. | low |

---

## Summary

| # | Issue | Severity | Category |
|---|-------|----------|----------|
| 1.1 | `Program.Run()`/`RunContext()` hold lock during execution | critical | thread safety |
| 1.2 | `Program.Clone()` shares mutable globals and constants | critical | thread safety |
| 2.1 | Eager init forces all stdlib into every binary (+ perf) | critical | integration |
| 3.1 | No signal handling — terminal left in raw mode | critical | CLI robustness |
| 8.1 | WASI/JS compilation impossible via root import chain | critical | compatibility |
| 1.3 | `Program.Equals()` doesn't lock `other` | high | thread safety |
| 2.2 | VM defaults `In`/`Out`/`Args` to OS globals (+ security) | high | integration |
| 2.3 | `Unmarshal` hardcoded to global `Modules` (+ security) | high | integration |
| 3.2 | CLI cannot forward arguments to scripts | high | CLI robustness |
| 3.3 | Debug output left in `main()` | high | CLI robustness |
| 4.1 | No version field in binary format | high | compatibility |
| 4.2 | Deserialized builtins are non-functional | high | compatibility |
| 8.2 | End-to-end context/abort propagation gap | high | integration |
| 1.4 | `Unmarshal` not atomic on failure | medium | thread safety |
| 2.4 | REPL source module imports run in isolated VMs | medium | integration |
| 2.5 | `compileSrc` unconditionally enables file imports | medium | integration |
| 3.4 | "Error compiling" message for runtime errors | medium | CLI robustness |
| 3.5 | Build output path inconsistent | medium | CLI robustness |
| 3.6 | REPL ignores context cancellation | medium | CLI robustness |
| 5.1 | REPL constants accumulate without bounds | medium | performance |
| 7.2–7.7 | Maintenance items (see table above) | low | maintenance |

## Prioritised Recommendations

1. **Install signal handler in `cmd/main.go`** — one-liner via `signal.NotifyContext`, highest user-facing impact. Remove the debug `fmt.Println`.
2. **Fix `Program` locking strategy** — move globals copy into `Run()`/`RunContext()` so the lock is only held briefly for setup, not during execution. This also fixes the `Clone()` sharing issue.
3. **Make `Modules`/`Exports` lazy or explicit** — break the eager init chain. Largest binary-size and portability win. Also fixes §8.1 (WASI/JS).
4. **Wire context/abort through all interactive modules** — fixes §8.2 end-to-end.
5. **Add bytecode format version** — simple change, prevents cryptic errors on format evolution.
6. **Fix deserialized `BuiltinFunction` resolution** — resolve by name against the builtin registry in `fixDecodedObject`.
7. **Propagate VM I/O from the root package** — add `Script.SetIn()`/`SetOut()`/`SetArgs()` and wire through to `NewVM`.
8. **Implement script argument forwarding in CLI** — accept `rumo run <file> [-- args...]` and set `vm.Args`.
9. **Fix `Equals()` locking** — acquire `other.lock.RLock()`.
10. **Check context in REPL loop** — add `ctx.Err()` check.
11. **Fix error message for runtime failures** — distinguish compile vs. run errors in `cmd/main.go`.
