# AGENTS.md — rumo

## Overview

rumo is a Go-based scripting language VM (fork of [d5/tengo](https://github.com/d5/tengo)) with goroutine-style concurrency (`start`, `chan`, `abort`). It compiles `.rumo` source to bytecode and executes it in a stack-based VM.

## Architecture

- **Root package (`rumo`)** — Public API: `Script`, `Program`, `Variable`. Entry points for embedding rumo in Go applications. `Script.Compile()` → `Program.Run()` is the core lifecycle.
- **`vm/`** — Compiler, VM, object system, builtins. `parser/` handles lexing/parsing, `token/` defines tokens/opcodes. The `Object` interface (`vm/objects.go`) is the universal value type — all custom types must implement it (or embed `ObjectImpl`).
- **`vm/module/`** — Helper framework for defining stdlib modules. `module.NewBuiltin("name").Func(def, impl)` with auto-wrapping of Go function signatures into `vm.CallableFunc`.
- **`std/`** — Standard library modules (base64, fmt, hex, json, math, rand, text, times, shell, cui). Each is a Go package exporting a `Module` variable.
- **`cmd/`** — CLI binary: REPL, `run`, `build` subcommands.

## Key Patterns

### Adding a stdlib module
1. Create `std/<name>/<name>.go` with `var Module = module.NewBuiltin("<name>").Func(...)` using the builder pattern (see `std/fmt/fmt.go`)
2. The `vm/module` package auto-wraps common Go signatures (e.g., `func(string) string`) — check `vm/module/builtin.go` for supported types
3. Source modules (`.rumo` files) go in `std/*.rumo` — they get embedded into `stdlib.go` by the generator

### Code generation — `stdlib.go`
`stdlib.go` is **auto-generated** — do NOT edit manually. Run `go run ./std` (or `make stdlib`) to regenerate it from `std/` directory contents. It scans subdirs for builtin modules and `.rumo` files for source modules.

### Concurrency model
Routines (`start(fn, args...)`) create a `routineVM` with a shallow-cloned VM (`vm/routinevm.go`). Channels (`chan(size)`) are Go channels wrapped as `Map` objects with `send`/`recv`/`close` methods. The VM context chain handles abort propagation.

### Object system
All values implement `vm.Object` interface. Embed `vm.ObjectImpl` for defaults. Key types: `Int`, `Float`, `String`, `Bool`, `Char`, `Bytes`, `Array`, `Map`, `Error`, `Time`, `Undefined`, `CompiledFunction`, `BuiltinFunction`. Singletons: `TrueValue`, `FalseValue`, `UndefinedValue`.

### Builtin functions
Registered in `vm/builtins.go` via `init()` + `addBuiltinFunction(name, fn)`. All builtins take `(context.Context, ...Object)` and return `(Object, error)`. Access the current VM via `ctx.Value(vm.ContextKey("vm")).(*vm.VM)`.

### Binary format
Compiled programs use format: `[4]MAGIC("VVC\0") [4]SIZE [N]DATA [8]CRC64(ECMA)`. Serialization via `vm/encoding/`.

## Build & Test

```bash
make stdlib     # regenerate stdlib.go (MUST run before build/test if std/ changed)
make test       # runs: stdlib generation → golint → gotestsum with race detector → e2e test
make build      # build CLI for current platform → build/
make release    # cross-compile for all platforms (linux/darwin/windows × amd64/arm64/etc.)
make install    # install build+test tool dependencies (cyclonedx-gomod, golint, gotestsum)
```

- Tests use `github.com/stretchr/testify` (via `vm/require/` wrapper) and standard `testing`
- Tests are in `*_test.go` files alongside source; `cmd/main_test.go` tests CLI behavior
- End-to-end test runs `vm/testdata/cli/test.rumo` via the built CLI
- Version/commit injected via `-ldflags` at build time

## File Import Security
File imports are sandboxed: `importBase` is set to the initial import directory, and all relative imports are checked against it via `isPathWithinBase()` in `vm/compiler.go`. Absolute paths are rejected.

