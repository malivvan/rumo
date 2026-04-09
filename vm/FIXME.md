# Architectural Review: `github.com/malivvan/rumo/vm`

## Executive Summary

The `vm` package is largely solid inherited architecture from tengo, but the concurrency layer (`routinevm.go`) bolted on top introduces several **data-race hazards, resource-leak vectors, and panic-safety gaps**. The core VM also carries a few pre-existing issues that become *amplified* under concurrency.

---

## 🔴 Critical — Concurrency (`routinevm.go`)

### 1. Shared `globals` slice: unsynchronised concurrent read/write (DATA RACE)

```go
// ShallowClone — vm.go:126
vClone := &VM{
    globals: v.globals, // shared slice, no synchronisation
}
```

`ShallowClone()` copies the pointer to the same `[]Object` globals slice. The parent VM and every child `routineVM` read **and write** globals (`OpSetGlobal`, `OpGetGlobal`) without any mutex or atomic guard. This is a **textbook data race** detectable by `-race` under any non-trivial concurrent workload.

**Impact:** Silent corruption, non-deterministic crashes.
**Fix:** Either (a) deep-copy globals per routine (isolation model — simplest), (b) wrap globals in a `sync.RWMutex`-guarded accessor, or (c) make globals copy-on-write.

---

### 2. Shared `constants` slice: mutable objects inside (DATA RACE)

```go
vClone := &VM{
    constants: v.constants, // shared
}
```

Constants are shared and assumed immutable, but the `constants` slice can hold `*CompiledFunction` whose `.Free` field contains `*ObjectPtr` — an *indirection cell* that is mutated at runtime via `OpSetFree`. If two VMs execute the same closure, they race on the pointed-to `Object`.

**Impact:** Subtle corruption of closed-over variables across routines.
**Fix:** Clone the `Free` slice (and the cells it points to) when a `CompiledFunction` is used in `OpClosure` inside a cloned VM, or deep-clone constants on `ShallowClone`.

---

### 3. `ShallowClone` context stores **parent** VM pointer, not the clone

```go
// vm.go:137
vClone.ctx, vClone.cancel = context.WithCancel(
    context.WithValue(v.ctx, ContextKey("vm"), v))
//                                                ^ parent, not vClone
```

Any builtin called inside the cloned VM that does `ctx.Value(ContextKey("vm"))` will receive **the parent VM**, not the child. For `builtinAbort` this means calling `abort()` inside a child routine aborts the *parent* instead of the child. For `builtinStart`, it registers new children on the wrong VM, breaking the child-tracking tree.

**Impact:** Incorrect abort propagation, orphaned goroutines.
**Fix:** Change to `context.WithValue(v.ctx, ContextKey("vm"), vClone)`.

---

### 4. `routineVM.abort()` races with goroutine completion (nil VM)

```go
func (gvm *routineVM) abort(ctx context.Context, args ...Object) (Object, error) {
    if gvm.VM != nil { // read
        gvm.Abort()
    }
    ...
}
```

Meanwhile the goroutine does:

```go
gvm.VM = nil // write, in the deferred cleanup
```

There is no synchronisation between the two, so `gvm.Abort()` can be called after `gvm.VM` was nilled, causing a nil-pointer dereference.

**Impact:** Panic / crash.
**Fix:** Guard `gvm.VM` access with a mutex, or use `atomic.Pointer[VM]`.

---

### 5. `wait()` channel receive is one-shot but `waitChan` is read from multiple call sites

`gvm.waitChan` has capacity 1. The first call to `wait()` drains it. A second concurrent call to `wait()` (or `result()`, which calls `wait(-1)`) from the parent script will **block forever** because the `ret` was already consumed but `done` flag was set non-atomically *after* the receive:

```go
case gvm.ret = <-gvm.waitChan: // consumes the single value
    atomic.StoreInt64(&gvm.done, 1)
```

If two goroutines call `wait()` simultaneously and both see `done == 0`, they both enter `select` — only one succeeds, the other blocks indefinitely.

**Impact:** Deadlock in concurrent `wait`/`result` calls.
**Fix:** Use a `sync.Once` or `close()`-based broadcast pattern (e.g. `chan struct{}`) instead of a single-value channel.

---

### 6. Channel `close()` panics are unrecoverable

```go
func (oc objchan) close(ctx context.Context, args ...Object) (Object, error) {
    close(oc) // double-close panics
    return nil, nil
}
```

A double-close or send-on-closed-channel will `panic` the goroutine. The `builtinStart` defer does recover panics for `CompiledFunction` routines, but the `callers == nil` check will itself panic with `"callers not saved"`. For non-compiled callables the panic is completely unrecovered.

**Impact:** Unhandled panics crash the entire process.
**Fix:** Wrap `close(oc)` in a recover, or track closed state with an `atomic.Bool`.

---

### 7. Non-compiled callables pass parent `ctx`, not child ctx

```go
// routinevm.go:97
val, err = fn.Call(ctx, args[1:]...)
```

`ctx` here is the **parent VM's context** (captured from `builtinStart`'s parameter). When the parent is aborted, the child's `ctx.Done()` fires correctly (it's a child context). But the child has **no independent cancel** — calling `gvm.abort()` when `gvm.VM == nil` (non-compiled case) does nothing because there is no `cancel()` for the non-compiled path. The child cannot be aborted.

**Impact:** Non-compiled routines are not abortable.
**Fix:** Create a dedicated `context.WithCancel` for the non-compiled path and wire `gvm.abort()` to call that cancel.

---

### 8. `addChild` with `nil` VM (non-compiled case) increments WaitGroup but is not tracked

```go
func (v *VM) addChild(cvm *VM) error {
    ...
    v.childCtl.Add(1)
    if cvm != nil {         // <-- skipped for non-compiled
        v.childCtl.vmMap[cvm] = struct{}{}
    }
    ...
}
```

This means non-compiled routines are waited on (WaitGroup), but can never be aborted via the `Abort()` loop over `vmMap`. They become **unkillable** from the parent.

---

## 🟠 High — Core VM (pre-existing, amplified by concurrency)

### 9. `Map` and `Array` mutations are not thread-safe

`Map.IndexSet`, `Array.IndexSet`, `builtinAppend`, `builtinDelete`, `builtinSplice` all mutate the underlying Go map/slice directly. When a map or array is reachable from multiple routines (via shared globals or closures), concurrent mutations cause data races.

**Fix:** Either document that sharing mutable objects between routines is undefined behaviour and enforce it at runtime, or introduce a synchronised wrapper for cross-routine objects.

---

### 10. `Int` division/modulo by zero is not guarded

```go
case token.Quo:
    r := o.Value / rhs.Value  // panics on rhs == 0
case token.Rem:
    r := o.Value % rhs.Value  // panics on rhs == 0
```

The `BinaryOp` for `Int` does not check for zero divisor. The resulting panic is caught by the top-level `recover()`, but generates a confusing `runtime panic` message rather than a clean runtime error.

**Impact:** Poor error diagnostics; in the concurrency path the panic recovery is more fragile (see #6).
**Fix:** Check for zero and return a clear `ErrDivisionByZero`.

---

### 11. `Abort()` TOCTOU between `atomic.Load` and `Lock`

```go
func (v *VM) Abort() {
    if atomic.LoadInt64(&v.aborting) != 0 { // check
        return
    }
    v.childCtl.Lock()
    atomic.StoreInt64(&v.aborting, 1)        // set
    ...
```

Two concurrent calls to `Abort()` can both pass the check, both acquire the lock sequentially, and both call `v.cancel()` — double-cancelling a context is safe in Go, but the recursive `cvm.Abort()` calls will be duplicated. Move the atomic check-and-set inside the lock, or use `atomic.CompareAndSwap`.

---

### 12. `checkGrowStack` sets `v.err` but does not return from `run()`

```go
func (v *VM) checkGrowStack(added int) {
    ...
    if should >= StackSize {
        v.err = ErrStackOverflow
        return  // returns from checkGrowStack, NOT from run()
    }
    ...
}
```

After `checkGrowStack` sets `v.err`, execution continues in `run()` with the old (too-small) stack, potentially causing an out-of-bounds access before the next iteration checks `v.aborting`. The same issue exists for the spread case (`OpCall`).

**Fix:** Either check `v.err` after every `checkGrowStack` call in `run()`, or have `checkGrowStack` trigger `v.aborting` to exit the loop.

---

## 🟡 Medium

### 13. `builtinRange` has no upper-bound limit

```go
func buildRange(start, stop, step int64) *Array {
    for i := start; i < stop; i += step {
        array.Value = append(array.Value, &Int{Value: i})
    }
}
```

`range(0, 9999999999)` will allocate billions of `*Int` objects with no allocation-limit check (the alloc counter only applies inside the `run()` loop, not in builtins). This is a trivial DoS vector.

---

### 14. `time.After` leak in `wait()`

```go
case <-time.After(time.Duration(seconds) * time.Second):
```

`time.After` creates a timer that is not garbage collected until it fires. If `wait()` returns early (via the channel case), the timer leaks until it expires — up to 100 years for the default `seconds < 0` path (though that path uses the `100 years` constant so it hits the channel case or blocks forever).

For practical timeouts this is a minor leak; for short repeated polls it adds up.

**Fix:** Use `time.NewTimer` + `timer.Stop()`.

---

### 15. `Importable.Import` returns `interface{}` instead of `any` with constrained types

The `Import` method returns a bare `interface{}` that is then type-switched over `[]byte` and `Object`. This makes the contract implicit and easily breakable. Consider a union interface or explicit separate methods.

---

### 16. Builtin function registration via `init()` is order-dependent

`builtins.go` and `routinevm.go` each have their own `init()` that calls `addBuiltinFunction`. Go's `init()` ordering within a package is defined by source file name, but this is fragile. If a file is renamed, the index mapping changes, breaking serialised bytecode that references builtins by index.

**Fix:** Use explicit registration with fixed indices, or a name-based lookup at deserialisation time (which `fixDecodedObject` partially does for modules but not for builtins).

---

### ~~17. Builtin function deserialization is incomplete~~ → moved to `FIXME.md` §4.2

Consolidated into the root-package review (`FIXME.md` §4.2 — "Deserialized `BuiltinFunction` objects are non-functional") which covers the full serialization/deserialization lifecycle including `fixDecodedObject`.

---

## 🟢 Low / Maintenance

| # | Issue |
|---|-------|
| 18 | `ImmutableArray.BinaryOp(Add)` mutates the receiver's underlying slice via `append(o.Value, rhs.Value...)` — the "immutable" contract is violated when the slice has spare capacity. |
| 19 | `String.IndexGet` lazily initialises `runeStr` on the receiver pointer. Under concurrency (shared String in globals), this is a data race on the `runeStr` field. |
| 20 | `ObjectImpl.TypeName()` and `String()` panic instead of returning a sentinel — any forgotten override crashes the VM. |
| 21 | Typo in `indexAssign` error message: `"invaid index value type"`. |
| 22 | `MaxStringLen`/`MaxBytesLen` are package-level `var`, meaning any embedder can mutate them concurrently — should be `atomic` or set once before use. |

---

## Summary of Recommendations (prioritised)

1. **Fix `ShallowClone` context to store `vClone` instead of `v`** — one-line fix, highest ROI.
2. **Synchronise or isolate `globals` and mutable shared objects** — required for correctness under concurrency.
3. **Guard `routineVM.VM` access** against nil-deref after goroutine completion.
4. **Replace single-value `waitChan` with a broadcast mechanism** to prevent deadlock.
5. **Add a cancellable context for non-compiled routines** so they can be aborted.
6. **Protect `objchan.close` / `send` against double-close / send-on-closed panics.**
7. **Check `v.err` after `checkGrowStack`** in the run loop to prevent OOB access.
8. **Guard integer division by zero** in `BinaryOp`.
9. **Pin builtin indices** for stable serialisation.
10. Consider adding a concurrency-safety section to `AGENTS.md` documenting which objects are safe to share across routines.
