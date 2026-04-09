# Architectural Review: `github.com/malivvan/rumo/std`

## Executive Summary

The standard library modules are functionally complete and the `vm/module` builder pattern provides a clean API for registering functions with auto-wrapping. However, the JSON decoder uses **panics for control flow** with no recovery, the `cli` and `cui` modules inherit **all VM data-race issues** through `ShallowClone`, the code generator has an **injection vulnerability** via backticks, and several deprecated Go APIs are still in use. There are also a handful of copy-paste bugs in argument error messages and missing edge-case guards.

---

## 🔴 Critical

### 1. JSON decoder panics on malformed internal state — no recovery anywhere

```go
// json/decode.go:84
func (d *decodeState) value() (vm.Object, error) {
    switch d.opcode {
    default:
        panic(phasePanicMsg) // crashes the process
    ...
}
```

There are **seven** `panic(phasePanicMsg)` calls across `value()`, `array()`, `object()`, and `literal()` in `json/decode.go`. There is **zero** `recover()` anywhere in the json package. Although `checkValid` pre-validates the input, a bug in the scanner state machine or an unexpected opcode sequence will crash the entire process — not return an error.

**Impact:** Unrecoverable process crash from malformed JSON edge cases.
**Fix:** Add a deferred `recover()` in the top-level `Decode()` function that converts panics into `error` returns.

---

### 2. `cli` and `cui` `callFunc` inherits all VM data-race issues via `ShallowClone`

> **Canonical entry** — this covers both `cli/cli.go` and `cui/cui.go`. Previously duplicated in `cli/FIXME.md` §2.2 (now consolidated here). The underlying `ShallowClone` bugs are in `vm/FIXME.md` §1–§3. Related: `cui/FIXME.md` §1.1 (stale context in callbacks).

```go
// cli/cli.go:31
func callFunc(ctx context.Context, fn vm.Object, args ...vm.Object) (vm.Object, error) {
    if cfn, ok := fn.(*vm.CompiledFunction); ok {
        if vmVal := ctx.Value(vm.ContextKey("vm")); vmVal != nil {
            parentVM := vmVal.(*vm.VM)
            clone := parentVM.ShallowClone()   // shares globals, constants
            return clone.RunCompiled(cfn, args...)
        }
        ...
    }
    ...
}
```

Both `cli/cli.go:28` and `cui/cui.go:14` contain an identical `callFunc` that calls `parentVM.ShallowClone()`. This inherits all of the data-race hazards documented in `vm/FIXME.md` #1–#3 (shared `globals` slice, mutable `constants`, context storing parent VM pointer). Cobra callbacks may execute on different goroutines, amplifying the race window.

Additionally, the `callFunc` function is **duplicated verbatim** across two packages — any fix applied to one must be manually propagated to the other.

**Impact:** Data races, silent corruption, incorrect abort propagation in CLI/CUI callbacks.
**Fix:** (a) Extract `callFunc` into `vm/module` or `vm` as a shared helper. (b) Apply the fixes from `vm/FIXME.md` #1–#3 to `ShallowClone` itself.

---

## 🟠 High

### 3. `rand.Seed` is deprecated (Go 1.20+)

```go
// rand/rand.go:18
Func("seed(seed int)", rand.Seed).
```

`rand.Seed` was deprecated in Go 1.20 — the global source is now automatically seeded. The module still exposes it both at the top level and inside `randRand` instances. On Go 1.20+ it is a no-op (with `GODEBUG=randseednop=0` required to restore old behaviour, as the test does).

**Impact:** Misleading API — users call `seed()` expecting it to work. Test relies on `GODEBUG` override.
**Fix:** Remove `seed` from the module-level API. For `randRand` instances (which use `rand.New(rand.NewSource(...))`) seeding is still meaningful and should be kept.

---

### 4. `strings.Title` is deprecated (Go 1.18+)

```go
// text/text.go:41
Func("title(s string) (ret string)", strings.Title).
```

`strings.Title` has been deprecated since Go 1.18 because it does not handle Unicode correctly (e.g., Dutch "ij" digraph). The Go team recommends `golang.org/x/text/cases`.

**Impact:** Incorrect title-casing for non-ASCII input; deprecation warnings.
**Fix:** Replace with `cases.Title(language.Und).String()` from `golang.org/x/text/cases` and `golang.org/x/text/language`.

---

### 5. Code generator backtick injection in `.rumo` source embedding

```go
// std/main.go:85
out.WriteString("\t\"" + modName + "\": module.NewSource(`" + modSrc + "`),\n")
```

Source modules are embedded in Go raw string literals (backtick-delimited). If any `.rumo` file contains a backtick character, the generated `stdlib.go` will have a **syntax error** and fail to compile.

**Impact:** Build breakage — a single backtick in any source module breaks `make stdlib`.
**Fix:** Use `strconv.Quote(modSrc)` instead of backtick wrapping, or implement backtick escaping via string concatenation (`` "`" + "`" + ... ``).

---

### 6. JSON `Encode` silently drops unknown `Object` types

```go
// json/encode.go:253
default:
    // unknown type: ignore
}
return b, nil
```

When encoding an array or map that contains a `CompiledFunction`, `BuiltinFunction`, `Error`, or custom `Object` implementation, the encoder produces **no output** for that element. This can generate malformed JSON (e.g., `[,1]` from `[func, 1]`).

**Impact:** Silently malformed JSON output.
**Fix:** Either (a) return an error for unrepresentable types, or (b) encode them as `null`.

---

### 7. `fmt` print functions panic on missing VM in context

```go
// fmt/fmt.go:18
func fmtPrint(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
    v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)  // nil-pointer if key absent
    ...
}
```

`fmtPrint`, `fmtPrintf`, and `fmtPrintln` all perform an unchecked type assertion on the context value. If the VM context key is missing (e.g., when called from a non-compiled callable or test harness), this produces a nil-pointer panic instead of a clean error.

**Impact:** Unrecoverable panic.
**Fix:** Check for nil before the type assertion and return a descriptive error.

---

## 🟡 Medium

### 8. `text.repeat` panics on negative count

```go
// text/text.go:597
if len(s1)*i2 > vm.MaxStringLen {
    return nil, vm.ErrStringLimit
}
return &vm.String{Value: strings.Repeat(s1, i2)}, nil
```

`i2` is converted via `vm.ToInt` which can return negative values. `strings.Repeat` panics when given a negative count. Additionally, `len(s1)*i2` can overflow on 32-bit platforms (both operands are `int`), bypassing the limit check.

**Impact:** Panic on negative count; potential overflow bypass.
**Fix:** Check `i2 < 0` before calling `strings.Repeat`. Use `int64` multiplication for the overflow check.

---

### 9. `text.format_float` panics on empty format string

```go
// text/text.go:734
ret = &vm.String{Value: strconv.FormatFloat(f1.Value, s2[0], i3, i4)}
```

`s2` comes from `vm.ToString` which can return an empty string. Accessing `s2[0]` on an empty string panics with index out of range.

**Impact:** Panic on empty format byte argument.
**Fix:** Validate `len(s2) > 0` before accessing `s2[0]`.

---

### 10. `text.substr` operates on byte indices, not rune indices

```go
// text/text.go:412
strlen := len(s1)    // byte length, not rune count
...
ret = &vm.String{Value: s1[i2:i3]}  // byte slicing
```

For multi-byte UTF-8 strings, `substr("日本語", 0, 2)` returns the first 2 *bytes* (which is an incomplete character), not the first 2 characters. This contradicts the expected string semantics.

**Impact:** Incorrect results for non-ASCII strings.
**Fix:** Document the byte-index behaviour explicitly, or switch to rune-based indexing via `[]rune(s1)`.

---

### 11. JSON decoder returns all numbers as `Float`

```go
// json/decode.go:213
default: // number
    ...
    n, _ := strconv.ParseFloat(string(item), 10)
    return &vm.Float{Value: n}, nil
```

All JSON numeric literals — including integers like `42` — are decoded as `*vm.Float`. This means `json.decode(json.encode(42))` produces `42.0` (a float), losing type fidelity. Downstream code that does `obj.(*vm.Int)` type assertions will fail.

**Impact:** Type mismatch after encode/decode round-trip for integers.
**Fix:** Detect whether the literal contains a decimal point or exponent; if not, parse as `int64` and return `*vm.Int`.

---

### 12. `wrapError` duplication in `vm/module`

```go
// vm/module/module.go:7 (unexported)
func wrapError(err error) vm.Object { ... }

// vm/module/error.go:6 (exported)
func WrapError(err error) vm.Object { ... }
```

These two functions have identical logic. The unexported `wrapError` is used by the auto-wrapping functions in `builtin.go`, while the exported `WrapError` is used by external module implementations. This duplication adds maintenance overhead and confusion.

**Fix:** Remove `wrapError` from `module.go`; update all internal references in `builtin.go` to use `WrapError`.

---

### 13. Copy-paste argument error-message bugs

Three error messages reference the wrong argument name or type name:

**a)** `funcASSRSs` (builtin.go:762) — second argument error says `"first"`:
```go
return nil, vm.ErrInvalidArgumentType{
    Name:     "first",       // should be "second"
    Expected: "string(compatible)",
    Found:    args[1].TypeName(),
}
```

**b)** `funcASSRI` (builtin.go:839) — second argument error uses `args[0]` type name:
```go
return nil, vm.ErrInvalidArgumentType{
    Name:     "second",
    Expected: "string(compatible)",
    Found:    args[0].TypeName(),  // should be args[1]
}
```

**c)** `timesBefore` (times/times.go:640) — second argument error uses `args[0]` type name:
```go
err = vm.ErrInvalidArgumentType{
    Name:     "second",
    Expected: "time(compatible)",
    Found:    args[0].TypeName(),  // should be args[1]
}
```

**Impact:** Misleading error messages when the wrong type is passed as the second argument.
**Fix:** Correct the `Name` and `Found` fields in all three locations.

---

### 14. `times.date` non-deterministically uses local timezone

```go
// times/times.go:386
ret = &vm.Time{
    Value: time.Date(i1, time.Month(i2), i3, i4, i5, i6, i7, time.Now().Location()),
}
```

The timezone is `time.Now().Location()` — which depends on the system's local timezone at call time. Scripts produce different results on different machines, and the behaviour is not documented.

**Impact:** Non-deterministic, platform-dependent results.
**Fix:** Either (a) default to UTC and accept an optional location argument, or (b) document the current behaviour prominently.

---

### 15. `ParseExport` and module registration panic on invalid input

```go
// vm/module/export.go:55
panic(fmt.Errorf("unexpected export format: %s", s))

// vm/module/builtin.go:182
panic(fmt.Errorf("unsupported function type: %T", impl))
```

`ParseExport`, `Func()`, `Const()`, and `NewSource` all use `panic` instead of returning errors. While this is intentional for catching init-time programming errors, a typo in a function definition string will crash the entire application at startup with a panic stack trace rather than a clear message.

**Impact:** Poor developer experience; makes it harder to debug typos in module definitions.
**Fix:** Consider wrapping the panics in a more descriptive init-time error format, or provide a validating constructor that returns errors for use in tests.

---

## 🟢 Low / Maintenance

| # | Issue |
|---|-------|
| 16 | **Dead code in `builtin.go`:** `funcARI64` (line 210) and `funcAI64R` (line 241) are defined but never called — the `Func()` switch wraps `func() int64` and `func(int64)` through other adapters (`funcARI` with cast, `funcAIR` with cast). |
| 17 | **Code generator bugs in `std/main.go`:** (a) `cdRoot` ignores `os.Chdir` errors. (b) `len(os.Args) < 1` is always false (Go guarantees `os.Args[0]` exists). (c) `os.ReadDir` error silently returns with no output. |
| 18 | **`text.re_match` recompiles the regex on every call** — unlike `re_compile` which returns a reusable `Regexp` object. Worth documenting the performance implication. |
| 19 | **JSON map key ordering is non-deterministic** — `Encode` iterates Go maps directly, producing different byte output on each call. This can cause test flakiness for map-containing assertions. |
| 20 | **`text.pad_left` / `pad_right` integer arithmetic can produce incorrect results** when `padStrLen > 1` — `((i2 - padStrLen) / padStrLen) + 1` rounds down, so the repeated pad string may be shorter than needed before trimming. |
| 21 | **`shell` and `cli` module test coverage is low** — `shell.go` has no tests for the module entry points (only the underlying readline library is tested); `cli.go` has no tests for `buildCommand`, `addFlags`, or `wrapContext`. |

---

## Summary of Recommendations (prioritised)

1. **Add `recover()` to `json.Decode()`** — prevent unrecoverable panics from malformed scanner states.
2. **Extract `callFunc` into a shared location** and fix `ShallowClone` context per `vm/FIXME.md` #1–#3.
3. **Replace `rand.Seed` and `strings.Title`** with non-deprecated alternatives.
4. **Fix backtick injection** in the code generator by using `strconv.Quote` for source embedding.
5. **Return error or `null` for unknown types in `json.Encode`** instead of silent omission.
6. **Guard `fmt` print functions** against missing VM context key.
7. **Fix the three copy-paste error-message bugs** in `builtin.go` and `times.go`.
8. **Add bounds checks** for `text.repeat` (negative count), `text.format_float` (empty format).
9. **Decide on byte vs rune semantics for `text.substr`** and document the choice.
10. **Remove `wrapError` duplication** in `vm/module`.

