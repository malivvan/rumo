---
title: native
---


## Table of Contents

- [Overview](#overview)
- [Syntax](#syntax)
- [Type System](#type-system)
- [Examples](#examples)
    - [Calling libm](#calling-libm)
    - [Calling into libc](#calling-into-libc)
    - [Passing Bytes](#passing-bytes)
    - [Passing and Returning Pointers](#passing-and-returning-pointers)
- [Introspection and Lifecycle](#introspection-and-lifecycle)
- [Memory and Lifetime Rules](#memory-and-lifetime-rules)
- [Errors](#errors)
- [Platform Support](#platform-support)
- [Limitations](#limitations)

---

## Overview

Rumo can dynamically load shared C libraries at runtime and call exported
symbols directly, with no Go `cgo` involvement. The feature is built on
[`vm/purego`](https://github.com/malivvan/rumo/tree/master/vm/purego) and is
exposed through a dedicated **`native`** statement.

A `native` statement:

1. **Declares** which symbols to bind and the exact C signature of each.
2. **Opens** the library (via `dlopen`) the first time the declared variable
   is accessed — typically immediately after the statement executes.
3. **Resolves** each symbol (via `dlsym`) and wraps it in a rumo-callable
   function that transparently converts arguments and return values.
4. **Binds** the result to a variable as a regular rumo `map`, so each call
   site looks like `lib.symbol(args...)`.

Because symbol resolution happens at **runtime**, the library does not need
to exist on the compile host — only on the machine that runs the compiled
bytecode.

---

## Syntax

```
native <name> = <path> {
    <funcDecl>
    <funcDecl>
    ...
}
```

Where each `<funcDecl>` is one of two equivalent forms:

```
name: func(paramType, paramType, ...) returnType   // long form
name   (paramType, paramType, ...) returnType      // short form
```

The return type may be omitted to indicate a `void` C function:

```
rumo_reset()                                       // no return value
exit(int) void                                     // explicit void
```

`<name>` becomes a regular variable in the enclosing scope (global or local)
and `<path>` is a string literal identifying the shared object. On Linux
this is typically something like `"libm.so.6"` or `"./libmylib.so"`; on
macOS, `"libSystem.dylib"` or `"libmylib.dylib"`.

The short form exists so short, type-only declarations read cleanly:

```rumo
native libc = "libc.so.6" {
    strlen(string) int
    abs(int)       int
    free(ptr)
}
```

---

## Type System

The following keywords map rumo values to C calling-convention slots:

| Rumo keyword | rumo value type(s) accepted       | C ABI type                | Notes |
|:------------:|:---------------------------------:|:-------------------------:|:------|
| `int`        | `int`, `char`, `bool`             | `int64_t` / `long`        | Signed; widened to 64-bit |
| `uint`       | `int`, `char`, `bool`             | `uint64_t` / `unsigned long` | Reinterpreted as unsigned |
| `bool`       | `bool`                            | `_Bool`                   | 0/1 |
| `float`      | `float`, `int`                    | `double`                  | rumo has no 32-bit float literal |
| `string`     | `string`                          | `const char *`            | Copied to a null-terminated arena if not already null-terminated |
| `ptr`        | `int`, `bytes`, `string`, `undefined` | `void *`              | `undefined` → `NULL`; `bytes`/`string` pass a pointer to their backing data |
| `bytes`      | `bytes`                           | `void *`                  | Passes a pointer to the first byte; C must learn length out-of-band |
| `void`       | _(return only)_                   | — / `void`                | Disallowed as a parameter type |

Return types follow the inverse mapping. `bytes` and `ptr` returns are
surfaced as raw integer pointers (`int` values), because purego has no way
to know the length of the returned buffer.

---

## Examples

### Calling libm

```rumo
fmt := import("fmt")

native libm = "libm.so.6" {
    sqrt: func(float) float
    pow:  func(float, float) float
    fabs: func(float) float
}

fmt.println(libm.sqrt(2.0))      // 1.4142135623730951
fmt.println(libm.pow(2.0, 10.0)) // 1024
fmt.println(libm.fabs(-3.14))    // 3.14
```

### Calling into libc

```rumo
native libc = "libc.so.6" {
    strlen(string) int
    abs(int)       int
    getpid()       int
}

fmt.println("pid    =", libc.getpid())
fmt.println("strlen =", libc.strlen("hello world"))
fmt.println("abs    =", libc.abs(-7))
```

### Passing Bytes

`bytes` arguments are handed to C as a raw `void *`. Length is not implicit;
you typically accompany the pointer with an `int` length parameter.

```rumo
native lib = "./libmylib.so" {
    sum_bytes(bytes, int) int
}

buf   := bytes("ABC")
total := lib.sum_bytes(buf, len(buf))    // 65 + 66 + 67 = 198
```

On the C side:

```c
int64_t sum_bytes(const unsigned char *buf, int64_t n) {
    int64_t s = 0;
    for (int64_t i = 0; i < n; i++) s += buf[i];
    return s;
}
```

### Passing and Returning Pointers

Raw pointers round-trip as `int` values, so you can compose C APIs that
return opaque handles:

```rumo
native libc = "libc.so.6" {
    malloc(int)      ptr
    free(ptr)
    memcpy(ptr, ptr, int) ptr
}

p := libc.malloc(64)             // returns a pointer as int
// ... use p with other calls ...
libc.free(p)
```

> **Caution:** rumo's garbage collector will _not_ manage memory returned
> from C. Anything allocated through a native call must be released by a
> matching native call (e.g. `free`) before the library is closed.

---

## Introspection and Lifecycle

Every loaded native map carries two extra keys in addition to the declared
symbols:

| Key          | Type                    | Purpose |
|:-------------|:------------------------|:--------|
| `__path__`   | `string`                | Path the library was opened from |
| `close`      | `builtin-function`      | Unloads the library (`dlclose`) |

`close()` is idempotent; calling it a second time is a no-op. After
closing, do not invoke any previously-bound function on the library — the
symbol pointers become invalid.

```rumo
native libm = "libm.so.6" {
    sqrt(float) float
}

fmt.println("loaded from", libm.__path__)
result := libm.sqrt(2.0)
libm.close()
```

---

## Memory and Lifetime Rules

- **Strings** passed as `string` are copied to purego-managed memory for the
  duration of the call if they are not already null-terminated. Do not hold
  onto the resulting C pointer after the function returns.
- **Bytes** are passed as a pointer to the slice's backing array. Rumo pins
  the slice for the duration of the call, but the C side must not retain
  the pointer after the call returns.
- **Returned C strings** (when a function's return type is `string`) are
  copied into a fresh rumo `string`; the original C memory is _not_ freed
  by rumo.
- **Pointers returned as `ptr` / `bytes`** are raw addresses; ownership is
  whatever the C API documents.

When in doubt, treat native calls like [cgo rules](https://pkg.go.dev/cmd/cgo#hdr-Go_references_to_C):
C code must not keep references to rumo memory after a call returns.

---

## Errors

### Compile-time errors

The compiler validates type names and binding uniqueness:

| Condition                                               | Error message                                                                        |
|---------------------------------------------------------|--------------------------------------------------------------------------------------|
| Unknown parameter type keyword                          | `native: unknown parameter type "..." in function "..." (allowed: int, uint, ...)`   |
| Unknown return type keyword                             | `native: unknown return type "..." in function "..." (allowed: ...)`                 |
| `void` used as a parameter type                         | `native: 'void' is not allowed as a parameter type in function "..."`                |
| Two bindings with the same name                         | `native: duplicate function binding "..."`                                           |
| Empty library path                                      | `native: empty library path`                                                         |
| Variable name already defined in the same block         | `'...' redeclared in this block`                                                     |

### Run-time errors

Loading the library and calling into it can fail at runtime:

| Condition                                          | Error message format                                                    |
|----------------------------------------------------|-------------------------------------------------------------------------|
| `dlopen` failure (missing/invalid file, ABI issue) | `native: failed to open "...": ...`                                     |
| Missing symbol                                     | `native <path>.<name>: ...` (wrapping the underlying `dlsym` error)     |
| Wrong argument count at the call site              | `native <name>: wrong number of arguments: want=N got=M`                |
| Type mismatch (e.g. string for an `int` parameter) | `native <name>: argument <i>: expected <type>, got <type>`              |

---

## Platform Support

`native` is supported on every platform where `vm/purego` ships a
`Dlopen`/`Dlsym`/`Dlclose` implementation:

- **Linux** (amd64, arm64, 386, arm, loong64, ppc64le, riscv64, s390x)
- **macOS / Darwin** (amd64, arm64)
- **FreeBSD / NetBSD**
- **Android**

**Windows is not currently supported.** The Go Playground build tag
(`faketime`) is also unsupported: on that target every `native` statement
fails at runtime with a "Dlopen is not supported in the playground" error.

Struct arguments and returns are **not currently exposed through the
`native` DSL**, even on platforms where purego supports them.

---

## Limitations

- No struct, array, or callback parameter types. Only the scalar types
  listed in [Type System](#type-system) are supported.
- `func`-valued parameters (C function pointers / callbacks) must currently
  be produced by other native calls and passed back as `ptr`; there is no
  way to hand a rumo closure to a C function directly.
- The return length of `bytes` / `ptr` return values is not tracked. If
  the C API returns a buffer whose length is known only at runtime, pair
  it with a companion `int` length call or wrap the API on the C side.
- Symbol resolution happens once, on first use of the binding; redefining
  or reloading the library from rumo requires calling `close()` and then
  executing the `native` block again in a new scope.
- The native variable is always mutable at the rumo level — you can
  reassign it — but re-assignment simply drops the reference; it does not
  automatically unload the library. Call `close()` first.
