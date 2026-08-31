# AGENTS.md

Guide for AI agents (and humans) working on the rumo codebase.

## What rumo is

rumo is a secure, lightweight, embeddable scripting language for Go hosts.
The primary goal is a language that **feels a lot like golang without
compromising its scripting-language character**: closures, first-class
functions, maps/arrays/structs, routines and channels, `defer`, `switch`,
`select`, `goto`, type system with compile-time type checking, and a
batteries-included standard library.

The runtime is a **bytecode-interpreted VM** (no JIT, no cgo for the core):
scripts are compiled to portable bytecode once and then executed by the VM.

## Goals

1. **Secure by default.** Untrusted scripts must be safe to run with the
   default configuration: deny-by-default permissions for privileged
   `std/os` operations (`vm.Permissions`, opt-in via
   `UnrestrictedPermissions()`), safe default resource limits
   (`MaxStringLen` / `MaxBytesLen` / `MaxAllocs` / `MaxFrames` / stack),
   and a deny-by-default allow-list for the `native` feature
   (`vm.AllowNativePath`).
2. **Embeddable and interruptable.** A script acting as an HTTP handler can
   be compiled to bytecode first and then **cloned / instantiated per
   request** (`Program.Clone()`); the running VM can be interrupted
   (`Abort`, `RunContext` + context cancellation, `routine.stop()`) when
   the request is cancelled, for optimal resource/performance management.
   The VM observes context cancellation from the run loop.
3. **Native shared libraries via purego.** The `native` statement loads
   shared libraries at runtime through `github.com/ebitengine/purego`
   (no cgo), gated by the host-controlled path allow-list. Only build with
   `-tags native`; the stub build must keep bytecode portable.
4. **Go-style concurrency.** `start fn(...)` launches routines (goroutine-
   backed child VMs) and `chan(n)` creates channels usable from `select`.
   Channel and routine implementations live in `vm/routinevm.go`,
   `vm/chan.go`, `vm/select.go`.
5. **Configurable standard library.** Hosts can compile with a subset of
   the standard library (pick modules via `rumo.GetModuleMap("fmt",
   "json", ...)`) to fine-tune space requirements, and can restrict a
   script to a subset of the available modules for security/sandboxing
   purposes (module maps are per-`Script`/`Program`).
6. **No platform limitations.** Do not impose constraints like wazero's
   "usable speed on 64-bit only". The VM is pure Go and runs everywhere Go
   runs, including `js/wasm` and 32-bit platforms. The `js/wasm` build is
   an ordinary Go wasm binary: it behaves like `GOMAXPROCS=1` — routines
   are in-process goroutines; do **not** reintroduce webworker-based
   concurrency or worker-specific IPC into the VM core.

## Non-goals

- **Not a Go compiler / not Go-compatible.** The syntax is Go-flavoured but
  the semantics are rumo's own (e.g. `start` instead of `go`, `stop()`
  instead of cancel, string-keyed maps). Do not chase 100% Go spec
  conformance.
- **No JIT / no native code generation for scripts.** Performance work
  belongs in the bytecode VM.
- **No cross-process or cross-worker object migration.** Channels and VMs
  are single-process objects. `js/wasm` runs a normal single-wasm runtime;
  SharedWorkers may host separate VM instances, but the VM core must not
  grow IPC machinery (a previous webworker attempt was removed for this
  reason).
- **The `native` feature is an escape hatch, not a sandbox.** It is
  disabled unless the host opts in per-path; inside an allowed library the
  script has raw FFI power. It is not a goal to virtualise memory.
- **No stable bytecode format guarantees** across releases (see
  `FormatVersion`); bytecode is an optimisation, not a public API.

## Repository layout

- `rumo.go`, `script.go`, `stdlib.go`, `variable.go`, `info.go` — public
  host API (`Script`, `Program`, `Modules`, REPL).
- `vm/` — the interpreter: `parser/` (lexer/parser/AST), `compiler*.go`,
  `bytecode.go` (marshalling/dedup), `vm.go` (run loop), `objects.go` /
  `types.go` / `numeric.go` (value types), `builtins*.go` (builtin table),
  `routinevm.go` + `chan.go` + `select.go` (concurrency), `native*.go`
  (purego FFI, `-tags native`), `module/` (Go-module helpers), `codec/`.
- `std/` — standard library modules. `stdlib.go` at the repo root is
  **generated**: run `go run ./std` (or `make stdlib`) after adding a
  module under `std/<name>/`; `Func("name(params) (ret) description")`
  strings feed the generated `doc/std-<name>.md`.
- `cmd/` — CLI (`main.go`) and the js/wasm runtime (`main_js.go`, build
  tag `js && wasm`, plus `cmd/web/` glue: `index.html`, `rumo.js`,
  `worker.js`).
- `doc/` — user-facing documentation (keep in sync with behaviour changes).
- `vm/testdata/` — `.rumo` fixtures used by tests.

## Build & test

```bash
go test ./...                 # default build
go test -tags native ./...    # with purego FFI
make test                     # gotestsum + race + native tag + e2e CLI run
make stdlib                   # regenerate stdlib.go + doc/std-*.md
make js                       # GOOS=js GOARCH=wasm build + web bundle
GOOS=js GOARCH=wasm go vet ./cmd   # quick wasm-target compile check
```

Always run `go test ./...` and `go test -tags native ./...` after touching
the VM; run the wasm check after touching anything used by `cmd/`.
Concurrency work must pass `go test -race -tags native ./vm/...`.

## Conventions and cautions

- **Builtin indices are a deterministic table**: all builtins register in
  `builtins.go`/`builtins_new.go` `init()`; adding/removing builtins
  changes bytecode indices.
- **Use `vm.VMFromContext(ctx)`** (never forge a context key or panic-cast
  `ctx.Value(...)`); builtin helpers without a VM should degrade gracefully
  (error, `ErrNoVM`, or documented fallback).
- **Respect per-VM limits**: helpers must read limits via
  `vm.ConfigFromContext(ctx)`, not `vm.DefaultConfig`; the run loop
  re-checks string/bytes limits on `OpBinaryOp` results.
- **Keep the VM sandbox intact**: deny-by-default permissions and limits;
  any new privileged stdlib function needs a permission gate.
- **Cycle-safe container operations**: `Equals`/`Copy`/`String` on
  `Array`/`Map`/`StructInstance` must use the visited-set helpers; never
  key visited sets with `uintptr(unsafe.Pointer(...))` — use typed object
  keys.
- **Truncated bytecode must never panic**: the dispatch loop validates
  every instruction (opcode + operand widths) before executing it.
- **Channels are typed `*Chan`** wrapping a local Go channel; script
  syntax `c.send(v)` / `c.recv()` / `c.close()` is the compatibility
  surface.
- **Docs are generated for std modules** — edit the `Func(...)` signature
  strings in the Go source, not `doc/std-*.md` directly.
