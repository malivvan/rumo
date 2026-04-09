# FIXME — std/cli architectural review

Systemic issues identified during review, with special focus on cross-platform compatibility (unix/windows/browser/wasi), performance, security, and correctness.
Severity: **critical** (will cause crashes/compilation failures on target platforms), **high** (likely to cause subtle bugs or silent misbehaviour), **medium** (performance/maintainability concern), **low** (minor improvement).

---

## 1. Cross-Platform Compatibility

### 1.1 Pervasive hard dependency on `os.Args` — critical

`ExecuteC()` falls back to `os.Args[1:]` when no args are set (command.go:1105):

```go
if c.args == nil && filepath.Base(os.Args[0]) != "cli.test" {
    args = os.Args[1:]
}
```

In WASI and browser/js environments, `os.Args` may be empty, synthetic, or not follow filesystem path conventions. The `filepath.Base(os.Args[0])` check is a Go-test-specific hack that assumes native OS path semantics. On WASI, `os.Args[0]` may be an arbitrary string (e.g., a module name), causing the guard to misbehave.

Additionally, the `run` builtin in `cli.go:213` falls through to this same `os.Args` default when called with no arguments, meaning a rumo script that calls `app.run()` with no args in WASI will get unpredictable behaviour.

**Affected:** `command.go:1105`, `cli.go:213` (implicit fallthrough).

**Fix direction:** Never rely on `os.Args` as a silent default. Require explicit args to be passed through the rumo `run()` method. Add a WASI/js build-tag file that provides a safe fallback (empty args) instead of reading `os.Args`.

---

### 1.2 Unconditional `os.Getenv` calls throughout — critical

Environment variable reads are scattered across multiple subsystems with no abstraction layer:

- **Flag env_vars:** `cli.go:508` — `os.Getenv(envVar)` for flag defaults
- **Active help:** `active_help.go:48-50` — `os.Getenv(activeHelpGlobalEnvVar)`
- **Completion config:** `completions.go:1015-1018` — `os.Getenv(configEnvVar(...))`
- **Debug logging:** `completions.go:957` — `os.Getenv("BASH_COMP_DEBUG_FILE")`

In WASI environments, `os.Getenv` may return empty strings for all variables (no environment concept), behave differently (WASI preview1 has limited env support), or not be available at all in sandboxed browser/js builds. The code silently degrades (empty strings) but the behaviour is undocumented and untested.

**Fix direction:** Abstract environment access behind an interface (e.g., `EnvFunc func(string) string`) settable on the root command, defaulting to `os.Getenv` on native platforms. This also enables testing without polluting the process environment.

---

### 1.3 Direct `os.Stdin`/`os.Stdout`/`os.Stderr` usage — critical

Default I/O falls back to OS file descriptors in multiple places:

```go
// command.go:394-409
func (c *Command) OutOrStdout() io.Writer { return c.getOut(os.Stdout) }
func (c *Command) ErrOrStderr() io.Writer { return c.getErr(os.Stderr) }
func (c *Command) InOrStdin() io.Reader  { return c.getIn(os.Stdin) }
```

Additionally:
- `util.go:242` — `fmt.Fprintln(os.Stderr, "Error:", msg)` in `CheckErr`
- `completions.go:968` — `fmt.Fprint(os.Stderr, msg)` in `CompDebug`
- `completions.go:958` — `os.OpenFile(path, ...)` for debug file

In WASI/browser environments, `os.Stdin`/`os.Stdout`/`os.Stderr` may be nil, closed, or mapped to non-standard destinations. The code never checks for nil writers before writing.

**Fix direction:** The `SetOut`/`SetErr`/`SetIn` methods exist but are never used by the rumo binding layer (`cli.go`). The `cliNew` function should wire the VM's output streams to the command's I/O. For WASI/browser, provide platform-specific defaults via build tags.

---

### 1.4 `os.Exit(1)` in `CheckErr` — critical

```go
// util.go:239-244
func CheckErr(msg interface{}) {
    if msg != nil {
        fmt.Fprintln(os.Stderr, "Error:", msg)
        os.Exit(1)  // hard process termination
    }
}
```

`CheckErr` is called from the default help command's `Run` function (command.go:1295-1296, 1305). If help rendering fails, the **entire process terminates** — not just the CLI app, but the entire rumo VM host. In WASI/browser, `os.Exit` may not be supported at all, or may behave as `panic`.

`os.Exit` also bypasses all deferred functions, meaning cleanup code (terminal restore, file handles, channel cleanup) is skipped.

**Affected:** `util.go:242`, called from `command.go:1295, 1296, 1305`.

**Fix direction:** Replace `os.Exit(1)` with `panic` (recoverable) or propagate errors. The help command should use `RunE` instead of `Run` to return errors cleanly.

---

### 1.5 `os.Create` in completion file generators — high

All four completion file generators use `os.Create`:

```go
// bash_completions.go:701, bash_completionsV2.go:470,
// zsh_completions.go:70, fish_completions.go:284,
// powershell_completions.go:320
outFile, err := os.Create(filename)
```

In WASI sandboxed environments, filesystem write access may be restricted or unavailable. These functions are not exposed to rumo scripts (the `completion` subcommand writes to stdout), but they are public Go API that could be called from Go embedders.

**Fix direction:** These are low-risk since the rumo binding only uses the `io.Writer` variants. Document that file-writing variants require filesystem access.

---

### 1.6 Shell completion scripts are platform-specific — high

The `completion` subcommand (completions.go:769-927) generates shell scripts for bash, zsh, fish, and powershell. These are completely non-functional in WASI/browser environments where no shell exists. The command is always registered (unless explicitly disabled), adding noise to help output.

All generated scripts use `eval` to execute the program for completion results, which assumes the binary is available in the shell's PATH and can be executed as a subprocess.

**Fix direction:** Auto-detect the runtime environment. On WASI/js builds, either suppress the completion subcommand entirely (via build tag) or provide a no-op stub. Add a `CompletionOptions.DisableDefaultCmd` default for non-native platforms.

---

### 1.7 `CompDebug` writes to filesystem — medium

```go
// completions.go:957-964
if path := os.Getenv("BASH_COMP_DEBUG_FILE"); path != "" {
    f, err := os.OpenFile(path,
        os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err == nil {
        defer f.Close()
        WriteStringAndCheck(f, msg)
    }
}
```

In WASI sandboxed environments, `os.OpenFile` may fail silently or panic. The error from `OpenFile` is handled, but `WriteStringAndCheck` calls `CheckErr` which calls `os.Exit(1)` on write failure.

**Fix direction:** Guard with a build tag, or replace `CheckErr` with a silent discard on write errors in debug logging.

---

### 1.8 No WASI/js build tags anywhere — high

The package has only two platform-specific files: `command_win.go` (`//go:build windows`) and `command_notwin.go` (`//go:build !windows`). There are **no** `//go:build wasi` or `//go:build js` files. All `os.*` and `filepath.*` calls compile and execute on all non-Windows platforms, including WASI and js/wasm.

**Fix direction:** Add `command_wasi.go` (and/or `command_js.go`) build-tag files that:
- Provide a no-op `preExecHookFn`
- Override default I/O to safe stubs
- Disable completion subcommand by default
- Replace `os.Exit` with `panic` in `CheckErr`

---

## 2. Thread Safety & Concurrency

### 2.1 Global mutable state without synchronisation — critical

Multiple package-level variables are read during command execution and writable via public API:

```go
// util.go
var initializers []func()             // mutated by OnInitialize()
var finalizers []func()               // mutated by OnFinalize()
var EnablePrefixMatching = false       // read in findNext()
var EnableCommandSorting = true        // read in Commands()
var EnableCaseInsensitive = false      // read in commandNameMatches()
var EnableTraverseRunHooks = false     // read in execute()
var MousetrapHelpText = "..."         // read in preExecHook()
var MousetrapDisplayDuration = 5*time.Second
var templateFuncs = template.FuncMap{} // mutated by AddTemplateFunc()
```

Since rumo supports concurrent routines (`start(fn)`), multiple CLI apps or concurrent calls to `OnInitialize`/`AddTemplateFunc` race with command execution. The `flagCompletionFunctions` global is correctly protected by a mutex, but all others are unprotected.

**Fix direction:** Either (a) make these fields on `Command` (per-instance configuration), (b) protect with `sync.RWMutex`, or (c) document as init-only (must be set before any `Execute` call) and enforce with a `sync.Once` pattern.

---

### ~~2.2 `callFunc` / `ShallowClone` inherits VM data-race issues~~ → see `std/FIXME.md` §2

Consolidated into `std/FIXME.md` §2 which is the canonical entry for the `callFunc`/`ShallowClone` data-race issue affecting both `cli` and `cui` packages. The underlying `ShallowClone` bugs are documented in `vm/FIXME.md` §1–§3.

---

### 2.3 `flagCompletionFunctions` global map leaks memory — high

```go
// completions.go:38
var flagCompletionFunctions = map[*pflag.Flag]CompletionFunc{}
```

This global map uses `*pflag.Flag` pointers as keys. When CLI apps are created and destroyed (e.g., in a test suite, or in a long-running rumo program that creates multiple CLI instances), old flag pointers are never removed. The map grows unboundedly.

**Fix direction:** Move completion function registration to per-command storage, or add a cleanup mechanism. Consider using a `sync.Map` with weak-reference semantics, or clear the map when a root command is garbage collected.

---

### 2.4 `run` closure captures `cmd` and `ctx` from `cliNew` — high

```go
// cli.go:177-238
"run": &vm.BuiltinFunction{
    Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
        // captures: cmd, hideVersion, hideHelp, hideHelpCommand from cliNew scope
        cmd.SetArgs(runArgs[1:])
        cmd.Execute()
    },
},
```

The `run` function closure captures the `cmd` pointer and config booleans from the `cliNew` call. If `run` is called multiple times (e.g., in a test loop), cobra's internal state from the previous execution leaks into the next (flag parse state, `commandCalledAs`, etc.). Cobra is not designed for re-execution without `ResetCommands()`.

**Fix direction:** Either (a) document that `run` is single-use, (b) call `cmd.ResetFlags()` and reinitialise state before each execution, or (c) rebuild the command tree on each `run` call.

---

## 3. Correctness

### 3.1 `checkCommandGroups` panics instead of returning errors — high

```go
// command.go:1209
if sub.GroupID != "" && !c.ContainsGroup(sub.GroupID) {
    panic(fmt.Sprintf("group id '%s' is not defined ...", sub.GroupID, sub.CommandPath()))
}
```

A misconfigured rumo script (e.g., using `category: "tools"` on a subcommand without defining the group) will **panic and crash the entire process**, not return a rumo error. This is inherited cobra behaviour meant for compile-time Go programs, but is inappropriate for a dynamic scripting language.

Similarly, `MarkFlagsRequiredTogether`, `MarkFlagsOneRequired`, and `MarkFlagsMutuallyExclusive` (flag_groups.go:33-77) all `panic` on missing flags.

**Affected:** `command.go:1209`, `flag_groups.go:38, 56, 70`.

**Fix direction:** Convert panics to error returns. In the rumo binding layer, wrap `cmd.Execute()` in a `recover` to catch any escaped panics from cobra internals.

---

### 3.2 `template.Must` panics on invalid template — high

```go
// util.go:189
template.Must(t.Parse(text))
```

If a user provides a malformed Go template via `SetUsageTemplate`, `SetHelpTemplate`, or `SetVersionTemplate`, the parse error panics at template execution time (not at registration time). While these methods are not currently exposed to rumo scripts, they are part of the public Go API.

**Fix direction:** Pre-validate templates at registration time (`SetUsageTemplate` etc.) and return errors. Alternatively, recover panics in template execution.

---

### 3.3 `run()` ignores rumo context — abort doesn't stop execution — high

```go
// cli.go:177
"run": &vm.BuiltinFunction{
    Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
        // context parameter is ignored (underscore)
        if err := cmd.Execute(); err != nil { ... }
    },
},
```

The `run` function accepts but ignores the context. If the rumo script's context is cancelled (via `abort()`), cobra execution continues to completion. Long-running commands or commands waiting for input will hang indefinitely.

**Fix direction:** Use `cmd.ExecuteContext(ctx)` instead of `cmd.Execute()`. Additionally, spawn a goroutine that watches `ctx.Done()` and aborts execution (though cobra has limited support for mid-execution cancellation).

---

### 3.4 `getShorthand` silently discards multi-character aliases — medium

```go
// cli.go:132-138
func getShorthand(aliases []string) string {
    for _, a := range aliases {
        if len(a) == 1 {
            return a
        }
    }
    return ""
}
```

pflag supports only a single shorthand character per flag. If a rumo script provides `aliases: ["v", "verbose"]`, only `"v"` is used as the shorthand; `"verbose"` is silently ignored. There is no mechanism to register long-form aliases for flags.

**Fix direction:** Document the limitation. Emit a warning or error if multi-character aliases are provided for flags (as opposed to commands, where aliases work correctly).

---

### 3.5 `env_vars` modifies `DefValue`, polluting help text — medium

```go
// cli.go:510-513
if f := flags.Lookup(name); f != nil {
    _ = f.Value.Set(val)
    f.DefValue = val  // changes displayed default in --help
}
```

When a flag's value is set from an environment variable, the code also overwrites `DefValue`. This means `--help` output shows the env-var value as the "default", which is misleading — the actual default was whatever was specified in the flag definition. Users who don't have the env var set will see different help text than users who do.

**Fix direction:** Set the value via `f.Value.Set(val)` but do **not** modify `f.DefValue`. The help text should show the compiled-in default, with a note about the env var.

---

### 3.6 `buildCommand` silently ignores unknown config keys — medium

```go
// cli.go:346-443
func buildCommand(ctx context.Context, cfg map[string]vm.Object, persistent bool) *Command {
    if name := cfgStr(cfg, "name"); name != "" { cmd.Use = name }
    // ... checks known keys, ignores everything else
}
```

A typo like `"actoin"` instead of `"action"` produces no error, no warning — the callback is simply never registered. This makes rumo CLI scripts difficult to debug.

**Fix direction:** After consuming all known keys, check for remaining unknown keys and either log a warning or return an error.

---

### 3.7 `wrapContext` returns `ImmutableMap` — not extensible — low

```go
// cli.go:523
func wrapContext(cmd *Command, args []string) *vm.ImmutableMap {
```

The context object passed to action/before/after callbacks is immutable. Rumo scripts cannot attach custom data to it in a `before` hook for consumption in the `action`. This limits the composability pattern common in CLI frameworks (e.g., middleware passing data through context).

**Fix direction:** Use a mutable `*vm.Map` instead of `*vm.ImmutableMap`, or provide a `set(key, value)` / `get(key)` pair that stores data on the command's annotations.

---

## 4. Performance

### 4.1 Templates recompiled on every invocation — medium

```go
// util.go:183-193
func tmpl(text string) *tmplFunc {
    return &tmplFunc{
        tmpl: text,
        fn: func(w io.Writer, data interface{}) error {
            t := template.New("top")            // new template every call
            t.Funcs(templateFuncs)
            template.Must(t.Parse(text))        // re-parse every call
            return t.Execute(w, data)
        },
    }
}
```

Every invocation of the template function re-creates and re-parses the template. For commands that display help frequently (e.g., a CLI tool run in a loop), this is wasteful. The `templateFuncs` map is also snapshot at execution time rather than registration time, which is inconsistent.

**Fix direction:** Parse the template once at registration time (in `tmpl()`) and reuse the parsed `*template.Template` in the closure. Capture `templateFuncs` at parse time.

---

### 4.2 Levenshtein distance allocates a 2D matrix per comparison — low

```go
// util.go:200-227
func ld(s, t string, ignoreCase bool) int {
    d := make([][]int, len(s)+1)
    for i := range d {
        d[i] = make([]int, len(t)+1)
    }
    ...
}
```

`ld` allocates `O(n*m)` memory for every Levenshtein comparison. `SuggestionsFor` calls it for every available subcommand. With many subcommands, this creates non-trivial GC pressure on every typo.

**Fix direction:** Use a single-row DP approach (`O(min(n,m))` space), or cache/pool the allocation.

---

### 4.3 `Gt` and `Eq` use `reflect` — low

```go
// util.go:118-161
func Gt(a interface{}, b interface{}) bool {
    av := reflect.ValueOf(a)
    ...
}
```

These functions are registered as template functions and called during help/usage rendering. Using `reflect` for simple comparisons adds overhead. They are also marked as deprecated in comments but still registered in `templateFuncs`.

**Fix direction:** Remove from `templateFuncs` if unused by the default templates (they are used by the template-based path but not by the `defaultUsageFunc`/`defaultHelpFunc` Go-code path). Or replace with type-switch implementations.

---

## 5. Security

### 5.1 `eval` in generated shell completion scripts — medium

All four shell completion generators produce scripts that use `eval` to invoke the program:

```bash
# bash
out=$(eval "${requestComp}" 2>/dev/null)

# zsh
out=$(eval ${requestComp} 2>/dev/null)
```

If the program name or arguments contain shell metacharacters (e.g., a program named `foo;rm -rf /`), the `eval` will execute arbitrary commands. While this is inherited from upstream cobra and is standard practice for shell completions, it's worth noting for security-sensitive deployments.

**Fix direction:** Document the risk. For high-security environments, recommend disabling the completion subcommand via `CompletionOptions.DisableDefaultCmd`.

---

### 5.2 `env_vars` can leak sensitive environment data — medium

```go
// cli.go:507-515
for _, envVar := range cfgStrArray(flagCfg, "env_vars") {
    if val := os.Getenv(envVar); val != "" {
        _ = f.Value.Set(val)
```

Rumo scripts can read arbitrary environment variables via the `env_vars` flag feature. In sandboxed environments, this could leak secrets (API keys, tokens) that happen to be in the environment. There is no allow-list or filtering.

**Fix direction:** For sandboxed/WASI environments, provide a mechanism to disable or restrict env var access. Consider an `AllowEnvVars` option on the root command.

---

### 5.3 `os.OpenFile` with fixed permissions in `CompDebug` — low

```go
// completions.go:959
f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
```

The debug log file is created with world-readable permissions (0644). In multi-user environments, completion debug logs (which may contain command-line arguments including secrets) are readable by other users.

**Fix direction:** Use 0600 permissions.

---

## 6. Maintenance / Code Quality

### 6.1 Dead/deprecated code still included — low

| Item | Location |
|------|----------|
| `Gt` and `Eq` functions | util.go:118-161 — marked as deprecated in comments, still registered |
| `appendIfNotPresent` | util.go:170-175 — marked as deprecated, still registered in `templateFuncs` |
| `BashCompletionFunction` field | command.go:102 — legacy bash completion, superseded by V2 |
| `GenBashCompletion` (V1) | bash_completions.go:682 — entire V1 bash completion system retained alongside V2 |
| `MarkZshCompPositionalArgumentFile/Words` | zsh_completions.go:55-68 — deprecated no-ops |
| `ExactValidArgs` | args.go:129-131 — deprecated, delegates to `MatchAll` |

**Fix direction:** Remove deprecated symbols in next major version. For now, mark with `// Deprecated:` doc comments for Go tooling.

---

### 6.2 `Module` declared in `util.go`, initialised in `cli.go` — low

```go
// util.go:34
var Module *module.BuiltinModule

// cli.go:14
func init() {
    Module = module.NewBuiltin().Func(...)
}
```

The module variable is declared in one file and initialised in another via `init()`. This split makes the code harder to follow and creates a dependency on Go's file-ordered `init()` execution.

**Fix direction:** Declare and initialise `Module` in the same file.

---

### 6.3 Error return values silently discarded — low

Multiple sites discard error returns with `_`:

```go
_ = cmd.MarkFlagRequired(name)    // cli.go:499
_ = f.Value.Set(val)               // cli.go:511
_ = flags.MarkHidden(name)         // cli.go:503
```

If these operations fail (e.g., flag not found due to a race condition or typo), the failure is invisible.

**Fix direction:** Log or return errors, especially for `MarkFlagRequired` which indicates a configuration problem.

---

## Summary

| #    | Issue | Severity | Category |
|------|-------|----------|----------|
| 1.1  | Hard dependency on `os.Args` | critical | compatibility |
| 1.2  | Unconditional `os.Getenv` calls | critical | compatibility |
| 1.3  | Direct `os.Stdin/Stdout/Stderr` usage | critical | compatibility |
| 1.4  | `os.Exit(1)` in `CheckErr` | critical | compatibility |
| 2.1  | Global mutable state without sync | critical | thread safety |
| ~~2.2~~ | ~~`callFunc`/`ShallowClone` data races~~ — see `std/FIXME.md` §2 | — | — |
| 3.1  | `checkCommandGroups` / flag group panics | high | correctness |
| 1.8  | No WASI/js build tags | high | compatibility |
| 2.3  | `flagCompletionFunctions` memory leak | high | thread safety |
| 2.4  | `run` closure captures stale cobra state | high | thread safety |
| 3.2  | `template.Must` panics on bad template | high | correctness |
| 3.3  | `run()` ignores rumo context / abort | high | correctness |
| 1.5  | `os.Create` in completion file generators | high | compatibility |
| 1.6  | Shell completions meaningless on WASI/js | high | compatibility |
| 3.4  | `getShorthand` discards multi-char aliases | medium | correctness |
| 3.5  | `env_vars` pollutes help text via `DefValue` | medium | correctness |
| 3.6  | Unknown config keys silently ignored | medium | correctness |
| 4.1  | Templates recompiled on every invocation | medium | performance |
| 5.1  | `eval` in shell completion scripts | medium | security |
| 5.2  | `env_vars` can leak sensitive environment data | medium | security |
| 1.7  | `CompDebug` writes to filesystem | medium | compatibility |
| 5.3  | Debug log file world-readable (0644) | low | security |
| 3.7  | `wrapContext` returns immutable map | low | correctness |
| 4.2  | Levenshtein allocates 2D matrix per call | low | performance |
| 4.3  | `Gt`/`Eq` use `reflect` | low | performance |
| 6.1  | Dead/deprecated code retained | low | maintenance |
| 6.2  | Module split across files | low | maintenance |
| 6.3  | Error returns silently discarded | low | maintenance |

