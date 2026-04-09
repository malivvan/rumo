# FIXME — std/shell

Architectural and systemic issues in the `shell` package, focused on cross-platform correctness, security, and long-term maintainability.

---

## 1. Cross-Platform Compatibility

### 1.1 Hard-coded `syscall.Stdin` / `syscall.Stdout` — breaks custom I/O and non-Unix platforms

**Files:** `utils.go`, `platform/utils_unix.go`, `platform/utils_windows.go`, `platform/utils_js.go`

`rawModeHandler` always calls `term.MakeRaw(int(syscall.Stdin))` and `term.Restore(int(syscall.Stdin), ...)`, completely ignoring the configurable `Config.Stdin` / `Config.Stdout`. If a user supplies a custom `io.Reader`/`io.Writer` (e.g. a PTY, an SSH channel, or a pipe), raw mode is still applied to the process-global stdin fd. Similarly, `DefaultIsTerminal()` and `GetScreenSize()` check `syscall.Stdin`/`syscall.Stdout` rather than the fds associated with the configured streams. On WASI/js where `syscall.Stdin` may not exist or may be 0 without a real terminal behind it, every one of these calls is either a no-op or an error.

**Fix:** The raw-mode and terminal-detection functions should accept file descriptors (or `*os.File` values) derived from the actual `Config.Stdin`/`Config.Stdout`, not hard-coded `syscall.Stdin`. `Config.init()` should extract the fd from the configured streams when they implement `interface{ Fd() uintptr }`, or disable interactive mode entirely when they don't.

### 1.2 No WASI build tag — `utils_js.go` does not cover WASI

**Files:** `platform/utils_js.go`

The JS/Wasm stub uses `//go:build js`, but Go 1.21+ introduced the `wasip1` GOOS. Code compiled for `GOOS=wasip1` will fall through to the `term_unsupported.go` stub, which returns hard errors from `MakeRaw`, `GetSize`, etc. There is no `platform/utils_wasi.go` at all.

**Fix:** Add a `utils_wasi.go` (build tag `wasip1`) with no-op / sensible-default implementations mirroring `utils_js.go`. Also add `wasip1` to the exclusion list in `term/term_unsupported.go` once a real stub is in place, or add it alongside `js` in the existing stubs.

### 1.3 Missing `DefaultOnSizeChanged` on Windows

**File:** `platform/utils_windows.go`

`DefaultOnSizeChanged` is a no-op on Windows. This means terminal resize events are never detected. The cached `termDimensions` will become stale if the user resizes their console, causing rendering corruption (line-wrapping miscalculated, completions clipped, etc.). Windows Console API does provide `WINDOW_BUFFER_SIZE_EVENT` via `ReadConsoleInput`, or a polling approach via `GetConsoleScreenBufferInfo`.

**Fix:** Implement `DefaultOnSizeChanged` on Windows using periodic polling or console input event monitoring.

### 1.4 ANSI escape reliance without fallback

**Files:** `runebuf.go`, `terminal.go`, `search.go`, `complete.go`

The entire rendering pipeline assumes VT100 escape sequences (`\x1b[...`) work. On older Windows consoles (pre-Win10 1511) or dumb terminals, `ansi.EnableANSI()` can fail, but the code proceeds to emit escapes anyway because `newTerminal` returns an error only if `EnableANSI` fails — and the Config's `isInteractive` flag might still be true if stdin/stdout are a console. Even on modern Windows, `ENABLE_VIRTUAL_TERMINAL_PROCESSING` can fail for redirected handles. There is no graceful degradation path.

**Fix:** If ANSI cannot be enabled, force `isInteractive = false` so the shell falls back to raw line-by-line I/O rather than emitting garbled escape sequences.

### 1.5 `CaptureExitSignal` uses `syscall.SIGTERM` — not portable

**File:** `instance.go`

`CaptureExitSignal` references `syscall.SIGTERM` unconditionally. While this compiles on Windows (the constant exists), the signal is never actually delivered on Windows. More critically, this file directly imports `syscall`, preventing compilation on `js` or `wasip1` targets where `syscall.SIGTERM` may not exist.

**Fix:** Use build-tagged files or guard the signal registration. On non-Unix platforms, `CaptureExitSignal` should be a documented no-op or use the platform's native mechanism.

---

## 2. Concurrency & Safety

### 2.1 Data race on `sizeChangeCallback`

**File:** `platform/utils_unix.go`

`sizeChangeCallback` is a package-level `func()` variable. `DefaultOnSizeChanged` writes it outside any lock, and the SIGWINCH goroutine reads it without synchronization. If `DefaultOnSizeChanged` is called multiple times (e.g. by multiple shell instances), this is a data race.

**Fix:** Protect `sizeChangeCallback` with an `atomic.Pointer[func()]` or a mutex. Better yet, support multiple listeners via a slice under a lock and provide an unsubscribe mechanism.

### 2.2 Only one global SIGWINCH listener is ever registered

**File:** `platform/utils_unix.go`

`sizeChange` is a `sync.Once`. The first shell instance that calls `DefaultOnSizeChanged` registers the goroutine. Subsequent instances replace `sizeChangeCallback` (data race aside), so only the last one gets resize notifications. If the first instance is closed, the goroutine continues running and calling a potentially stale callback. Multiple shell instances cannot coexist correctly.

**Fix:** Redesign as a subscription-based listener registry, or move the SIGWINCH handler into the `terminal` struct so each instance manages its own lifecycle.

### 2.3 Goroutine leak in `ioLoop` on Close

**File:** `terminal.go`

`ioLoop` does a blocking `buf.ReadRune()` on stdin. When `Close()` is called, `stopChan` is closed, but `ioLoop` may be blocked in `ReadRune` waiting for the next keystroke. The comment in the code acknowledges this: _"it will consume one more user keystroke before it exits"_. In long-running applications where the shell is opened/closed many times, this leaks goroutines (one per Close) that are permanently stuck in `ReadRune`.

**Fix:** This is a fundamental Go limitation with blocking reads. Possible mitigations: (a) use `os.File.SetReadDeadline` where available (Unix), (b) read from a cancellable wrapper, or (c) document the limitation clearly and ensure at most one `ioLoop` goroutine exists at a time.

### 2.4 Goroutine leak in `CaptureExitSignal`

**File:** `instance.go`

`CaptureExitSignal` spawns a goroutine and calls `signal.Notify`, but never calls `signal.Stop` or closes the channel. Each call leaks a goroutine and a signal registration. If called multiple times (accidentally or across instances), signal handlers accumulate.

**Fix:** Store the signal channel on the `Instance`, and clean it up in `Close()` via `signal.Stop(ch)` + close.

---

## 3. Security

### 3.1 History file created with world-readable permissions

**File:** `history.go`

`os.OpenFile(cfg.HistoryFile, ..., 0666)` creates history files with mode `0666` (minus umask). On multi-user systems this may expose command history (which can contain passwords, tokens, etc.) to other users. The temporary rewrite file uses the same mode.

**Fix:** Use `0600` for history files.

### 3.2 History file rewrite is not atomic on all platforms

**File:** `history.go`

`rewriteLocked` creates a `.tmp` file, writes to it, then calls `os.Rename`. On Windows, `os.Rename` fails if the destination already exists. Additionally, on a crash between `Rename` and the subsequent `fd.Close()`, the file descriptor state may be inconsistent.

**Fix:** On Windows, remove the destination before rename or use `os.Rename` alternatives. Ensure the temp file is fsynced before rename for crash safety.

### 3.3 `debugPrint` writes to a predictable file path

**File:** `utils.go`

`debugPrint` unconditionally opens `debug.tmp` in the current working directory with mode `0644`. Even though it's only called from `debugList`, the function is exported-by-convention (lowercase, but in the same package). If debug logging is accidentally enabled in production, it creates a world-readable file in the CWD. This is a symlink attack vector on shared systems.

**Fix:** Remove `debugPrint` and `debugList`, or gate them behind a build tag (e.g. `//go:build shelldebug`). If kept, use `os.CreateTemp` with `0600` permissions.

### 3.4 Unbounded history / password leakage

**File:** `history.go`

`ReadPassword` delegates to `ReadLineWithConfig` with a separate config, but the caller's `DisableAutoSaveHistory` setting is not inherited — the password config sets `HistoryLimit: -1` which disables history. However, if a user reads sensitive input via `ReadLine()` (without the password API), it is auto-saved to the history file by default. There is no mechanism to mark individual lines as sensitive.

**Fix:** Document this behavior clearly. Consider adding a `Sensitive` flag or a `ReadSensitiveLine` convenience method that auto-disables history for that read.

---

## 4. Correctness

### 4.1 `ColorFilter` panics on malformed ANSI sequences

**File:** `runes/runes.go`

`ColorFilter` accesses `r[pos+1]` without a bounds check after finding `\033`. If a string ends with a bare `\033`, this panics with an index-out-of-range error. The same function also only handles `\033[...m` sequences; other ANSI sequences (OSC, DCS, etc.) are left in the output, causing display width miscalculations.

**Fix:** Add bounds checking: `if pos+1 < len(r) && r[pos+1] == '['`. Consider handling other escape sequence types or at least skipping them.

### 4.2 `EqualRuneFold` only handles ASCII

**File:** `runes/runes.go`

Case-folding is limited to ASCII `A-Z` / `a-z`. History search with `HistorySearchFold` will fail for non-ASCII (e.g. `ü` vs `Ü`, Cyrillic, etc.).

**Fix:** Use `unicode.ToLower` / `unicode.ToUpper` or `unicode.SimpleFold` for proper Unicode case folding.

### 4.3 `IsWordBreak` is ASCII-only

**File:** `runes/runes.go`

`IsWordBreak` treats everything outside `[a-zA-Z0-9]` as a word break. This means all Unicode letters (accented Latin, CJK, Cyrillic, etc.) are treated as word breaks, making word-movement commands (`Alt-f`, `Alt-b`, `Ctrl-W`) essentially useless for non-English text.

**Fix:** Use `unicode.IsLetter` and `unicode.IsDigit` instead of hard-coded ASCII ranges.

### 4.4 Width calculation doesn't match all terminal behavior

**File:** `runes/runes.go`

The `Width` function uses `unicode.Han`, `unicode.Hangul`, etc. range tables for double-width detection alongside `x/text/width`. The range-table approach and the `x/text/width` approach can disagree (some symbols are ambiguous-width). Also, emoji (which are typically double-width in modern terminals) are not explicitly handled and may be measured as single-width.

**Fix:** Consider using `x/text/width` exclusively and/or adding explicit emoji width handling. Test against a matrix of terminal emulators.

### 4.5 Swapped doc comments on `FuncMakeRaw` / `FuncGetSize`

**File:** `instance.go`

The doc comment for `FuncMakeRaw` says _"FuncGetSize is a function that returns the width and height..."_, and the doc comment for `FuncGetSize` says _"FuncMakeRaw is a function that puts the terminal into raw mode..."_. The comments are swapped.

**Fix:** Swap the doc comments to match their respective fields.

### 4.6 Completion select mode `j`/`k` conflict with normal input

**File:** `complete.go`

In `HandleCompleteSelect`, the literal characters `j` and `k` are used for page navigation (prev/next page). This means if a user types `j` or `k` while in completion select mode, it navigates pages instead of exiting select mode and inserting the character. This is unintuitive and undiscoverable — especially since the guidance text shows `(j: prev page) (k: next page)` but users expect typing to exit select mode.

**Fix:** Use control characters or modifier keys for pagination instead of literal letter keys. Alternatively, use the same `J`/`K` (uppercase) convention to at least reduce conflict with common lowercase input.

---

## 5. Performance

### 5.1 Excessive allocations in the rendering hot path

**Files:** `runebuf.go`, `runes/runes.go`

Every call to `Refresh` triggers `getSplitByLine` → `SplitByLine`, which allocates a new `[][]rune` and calls `append(prompt, rs...)` copying the entire prompt + buffer. `ColorFilter` allocates a new slice on every call. `output()` creates a new `bytes.Buffer` on every call. For long lines or high-frequency refreshes (e.g. Painter callbacks, concurrent writes), this generates significant GC pressure.

**Fix:** Pool or reuse buffers. Cache `ColorFilter` results for the prompt (it doesn't change between refreshes). Pre-allocate `SplitByLine` results.

### 5.2 `opHistory` uses `container/list` (linked list)

**File:** `history.go`

History is stored in a `*list.List` (doubly-linked list), which has poor cache locality and high per-element allocation overhead. For the typical history sizes (500 entries), a slice-based ring buffer would be more efficient for both iteration (search) and memory.

**Fix:** Replace `*list.List` with a slice-based ring buffer (the `ringbuf` package already exists in this project).

### 5.3 Full-buffer rewrite on every history save

**File:** `history.go`

`Compact()` is called on every `Update()`, and it removes elements from the front of a linked list. `rewriteLocked()` iterates the entire history and rewrites the file. This happens during `historyUpdatePath` if the file has grown beyond the limit. For large history files, this is a noticeable pause.

**Fix:** Use an append-only strategy with periodic compaction (e.g. on close or when the file is 2x the limit), rather than compacting eagerly.

---

## 6. API / Design Issues

### 6.1 `term/` package is a vendored fork of `golang.org/x/term` with modifications

**File:** `term/term.go` (1042 lines)

The `term/` sub-package is a fork of `golang.org/x/term` with the `Terminal` type (VT100 terminal emulator) included. This `Terminal` type is ~900 lines of code that appears to be **entirely unused** by the shell package — the shell has its own `terminal` type in `terminal.go`. The forked `term` package is only used for `MakeRaw`, `Restore`, `GetSize`, and `IsTerminal`. This is a large maintenance burden for four functions.

**Fix:** Either (a) depend on `golang.org/x/term` directly for the four functions actually used and delete the vendored `Terminal` type, or (b) extract only the needed platform functions into minimal files. This removes ~900 lines of dead code and the associated test/maintenance burden.

### 6.2 No `context.Context` integration in blocking reads

**Files:** `operation.go`, `terminal.go`

`ReadLine()` and its variants block until input is received, with no way to cancel via `context.Context`. The rumo VM's concurrency model uses contexts for abort propagation, but `shell.readline()` in a rumo script cannot be cancelled by `abort()` — it will block the goroutine indefinitely. The `deadline` channel parameter on internal methods is a partial solution, but it's never exposed to the public API.

**Fix:** Add `ReadLineContext(ctx context.Context)` that wires the context's `Done()` channel to the internal deadline mechanism. Update the rumo bindings in `shell.go` to pass the VM context through.

### 6.3 Shell instance bindings ignore context

**File:** `shell.go`

The rumo binding functions (e.g. `readline`, `read_password`) receive a `context.Context` but never pass it to the underlying shell operations. This means `abort()` in a rumo script won't interrupt a blocking `readline()` call.

**Fix:** Thread the context through to the shell instance's read methods (requires fixing 6.2 first).

