# FIXME — std/cui architectural review

Systemic issues identified during review, grouped by category.
Severity: **critical** (will cause crashes/corruption under concurrency), **high** (likely to cause subtle bugs), **medium** (performance/maintainability concern), **low** (minor improvement).

---

## 1. Thread Safety

### 1.1 Stale context capture in widget callbacks — critical

Every `set_*_func` wrapper in `cui.go` captures `ctx` (the context from the registration call) in a closure that is later invoked by the tview event loop:

```go
b.SetSelectedFunc(func() {
    callFunc(ctx, cb) // ctx captured at registration time
})
```

`callFunc` extracts the parent VM from this context and calls `ShallowClone()`. When the rumo routine that registered the callback exits or is aborted, the context is cancelled and the VM may be partially torn down. The callback, however, is still alive in the tview event loop and will attempt to clone a dead/aborted VM on the next UI event. This causes use-after-free style corruption and panics.

**Affected:** All 19 `callFunc(ctx, cb, ...)` call sites in `cui.go`.

**Fix direction:** Callbacks should either (a) capture the `*App` and use `QueueUpdate` to bounce execution back to a known-safe goroutine, or (b) store a long-lived, routine-independent context that is explicitly invalidated when the app stops — with the callback checking validity before invoking `callFunc`.

---

### 1.2 Callback errors are silently discarded — critical

`callFunc` returns `(vm.Object, error)`, but every call site inside a tview callback closure ignores both values:

```go
a.QueueUpdate(func() {
    callFunc(ctx, args[0]) // error silently dropped
})
```

If the rumo callback function panics, returns an error, or the VM clone fails, the error vanishes. This makes debugging multi-routine CUI applications nearly impossible and can leave widget state inconsistent.

**Affected:** All 19 `callFunc` call sites in `cui.go`.

**Fix direction:** At minimum, log errors via the VM error collection mechanism (`vm.addError`). Ideally, propagate errors to the App so the user can install a global error handler.

---

### 1.3 Dual lock hierarchies between Box and widget — high

Widgets embed `*Box` (which has its own `l sync.RWMutex`) and also embed their own `sync.RWMutex`. The `Draw()` methods acquire locks in this order:

```
Box.l.RLock()  →  release  →  Widget.Lock()  →  Box.l.RLock() (via GetInnerRect)
```

For example, `TextView.Draw()` calls `t.Box.Draw(screen)` (acquires/releases `Box.l`), then `t.Lock()`, then `t.GetInnerRect()` (acquires `Box.l.RLock()` again). Meanwhile, a mutation method like `SetText()` acquires `t.Lock()` only. If the event loop calls `Draw()` while a routine calls `SetRect()` (which acquires `Box.l.Lock()`), the nested acquisition pattern can produce inconsistent reads — `Draw()` may see a partially-updated rect because `Box.l` and `t.RWMutex` are independent.

**Affected:** `TextView`, `Table`, `List`, `Flex`, `Grid`, `Form`, `Panels`, `DropDown`, `InputField`, `CheckBox`, `ProgressBar` — all widgets with the dual-mutex pattern.

**Fix direction:** Unify the lock hierarchy. Either (a) use a single mutex per widget that also protects Box fields, or (b) always acquire locks in a strict documented order (widget lock before Box lock) and audit every code path.

---

### 1.4 `FocusManager` calls `setFocus` under its own lock — high

`Focus()`, `FocusPrevious()`, `FocusNext()`, and `FocusAt()` all call `f.setFocus(...)` while holding `f.Lock()`. The `setFocus` callback typically resolves to `App.SetFocus()`, which acquires `App.Lock()`. If any code path reaches `FocusManager` while holding the App lock, a lock-ordering deadlock occurs.

```go
func (f *FocusManager) FocusNext() {
    f.Lock()
    defer f.Unlock()
    f.focused++
    f.updateFocusIndex(false)
    f.setFocus(f.elements[f.focused].widget) // calls App.SetFocus → App.Lock
}
```

**Fix direction:** Release `f`'s lock before invoking `setFocus`, or use a channel/queue to decouple focus changes from the FocusManager lock scope.

---

### 1.5 `App.SetFocus` lock juggling creates TOCTOU windows — high

`SetFocus` performs multiple Lock/Unlock cycles within a single logical operation:

```go
func (a *App) SetFocus(w Widget) {
    a.Lock()
    if a.beforeFocus != nil {
        a.Unlock()
        ok := a.beforeFocus(w)  // arbitrary user code runs unlocked
        if !ok { return }
        a.Lock()                // state may have changed
    }
    // ... modifies a.focus ...
    if a.afterFocus != nil {
        a.Unlock()
        a.afterFocus(w)         // arbitrary user code runs unlocked
    } else {
        a.Unlock()
    }
    if w != nil {
        w.Focus(...)            // may recursively call SetFocus
    }
}
```

Between the Unlock and re-Lock, another routine can change `a.focus`, `a.beforeFocus`, or other state. The recursive `w.Focus(...)` call at the end can also trigger re-entrant `SetFocus` calls, causing unexpected state.

**Fix direction:** Collect the operations to perform while locked, then execute them after a single unlock. Or use a single-writer event queue for focus changes.

---

### 1.6 `App.Stop()` can deadlock on double-call — high

`Stop()` sends `nil` to `screenReplacement` (buffer size 1) while holding the App lock:

```go
func (a *App) Stop() {
    a.Lock()
    defer a.Unlock()
    a.finalizeScreen()
    a.screenReplacement <- nil // blocks if buffer full
}
```

If `Stop()` is called twice (e.g., from an abort handler and a Ctrl-C handler racing), the second call finds `screen == nil` (no-op in `finalizeScreen`), then attempts to send to a channel whose buffer is already full (the replacement goroutine already exited). This blocks forever while holding the App lock, deadlocking the entire application.

**Fix direction:** Make `Stop()` idempotent with a `sync.Once` or a `stopped` flag checked under lock, skipping the channel send on subsequent calls.

---

### 1.7 `List.GetItem()` is unprotected — high

```go
func (l *List) GetItem(index int) *ListItem {
    if index > len(l.items)-1 {
        return nil
    }
    return l.items[index]
}
```

This method reads `l.items` without acquiring any lock, unlike all other List getter methods (`GetItemCount`, `GetCurrentItem`, etc.). Concurrent mutation of the items slice causes a data race.

**Fix direction:** Add `l.RLock()`/`l.RUnlock()`.

---

### 1.8 `Styles` and `TabSize` globals are unsynchronized — medium

`Styles` is a package-level `var Theme` read during widget construction (`NewBox()`, `NewTextView()`, `NewTable()`, etc.) and during `Draw()`. `TabSize` is read during `Write()`. Neither is protected by a mutex. If routines create widgets while another routine modifies these globals, data races occur.

**Affected:** Every `New*()` constructor and `Draw()` method.

**Fix direction:** Either make them `sync.Once`-initialized constants, protect them with a `sync.RWMutex`, or document them as init-only (set before any widget creation).

---

### 1.9 `Table.Select()` / `List.SetCurrentItem()` unlock-callback-relock pattern — medium

Several widget methods release their lock before invoking a user callback, then re-acquire:

```go
func (t *Table) Select(row, column int) {
    t.Lock()
    defer t.Unlock()
    t.selectedRow, t.selectedColumn = row, column
    if t.selectionChanged != nil {
        t.Unlock()
        t.selectionChanged(row, column)
        t.Lock()
    }
}
```

While necessary to avoid deadlocks, this creates windows where the widget's invariants do not hold. If the callback mutates the table (e.g., removes rows), the post-callback state may be inconsistent with assumptions made before the unlock.

**Affected:** `Table.Select()`, `Table.InputHandler()`, `List.SetCurrentItem()`, `List.RemoveItem()`, `List.InsertItem()`, `List.InputHandler()`, `List.MouseHandler()`, `TextView.Highlight()`.

**Fix direction:** Document the re-entrancy contract. Validate state after re-acquiring the lock. Consider queuing mutation operations instead of immediate execution.

---

### 1.10 `TextView.SetMaxLines` and `SetToggleHighlights` are unprotected — medium

```go
func (t *TextView) SetMaxLines(maxLines int) *TextView {
    t.maxLines = maxLines  // no lock
    t.clipBuffer()         // no lock, mutates t.buffer
    return t
}

func (t *TextView) SetToggleHighlights(toggle bool) *TextView {
    t.toggleHighlights = toggle  // no lock
    return t
}
```

These methods mutate struct fields without acquiring the widget's lock, unlike all other setter methods on `TextView`.

**Fix direction:** Add `t.Lock()`/`t.Unlock()` wrappers.

---

### 1.11 `TextView.SetHighlightedFunc` is unprotected — medium

```go
func (t *TextView) SetHighlightedFunc(handler func(...)) *TextView {
    t.highlighted = handler  // no lock
    return t
}
```

**Fix direction:** Add `t.Lock()`/`t.Unlock()`.

---

## 2. No Abort/Context Propagation to App.Run()

### 2.1 `App.Run()` is not context-aware — critical

When a rumo routine calls `app.run()`, the call blocks in the tcell event loop. If the routine is aborted via `abort()`, the VM's context is cancelled, but `App.Run()` has no mechanism to observe this cancellation. The terminal stays in raw mode, the event loop keeps running, and the routine goroutine hangs forever waiting for `Run()` to return.

```go
"run": fn("run", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
    if err := a.Run(); err != nil {  // blocks, ignores ctx
        return module.WrapError(err), nil
    }
    return vm.UndefinedValue, nil
}),
```

**Fix direction:** Either (a) add a `RunWithContext(ctx)` method to `App` that monitors `ctx.Done()` and calls `Stop()`, or (b) spawn a goroutine in the wrapper that watches `ctx.Done()` and calls `a.Stop()`:

```go
go func() {
    <-ctx.Done()
    a.Stop()
}()
```

---

## 3. Performance

### 3.1 Per-widget closure allocation — medium

Every `wrap*` function allocates a `map[string]vm.Object` with closure-based `BuiltinFunction` objects for every method of every widget. A moderately complex UI (50+ widgets) creates hundreds of closures and map entries.

**Affected:** All 18 `wrap*` functions in `cui.go`.

**Fix direction:** Consider a single `IndexGet`-based dispatch on a custom `Object` type (implementing `vm.Object.IndexGet`) that performs method dispatch without pre-allocating all method closures. Methods would be resolved lazily on first access.

---

### 3.2 `wrapTreeNode` re-wraps on every callback invocation — medium

`TreeView.SetSelectedFunc` and `SetChangedFunc` call `wrapTreeNode(node)` inside the callback, creating a new `ImmutableMap` + closures on every selection/change event:

```go
tv.SetSelectedFunc(func(node *TreeNode) {
    callFunc(ctx, cb, wrapTreeNode(node))  // new wrapper every call
})
```

For a tree with frequent navigation, this creates significant GC pressure.

**Fix direction:** Cache wrappers per `*TreeNode` identity (e.g., in a `sync.Map` or a field on the node).

---

### 3.3 `App.QueueUpdate` channel backpressure — medium

`App.updates` is a buffered channel of size 100. If rumo routines enqueue updates faster than the event loop processes them, the 101st `QueueUpdate` call blocks the calling routine silently. There is no timeout, backpressure signal, or error.

**Fix direction:** Document the limit. Consider a dynamically-sized queue, or return an error/drop updates when the queue is full.

---

## 4. Security

### 4.1 No terminal escape sequence sanitization — medium

Text set via `set_text`, `add_item`, `set_cell`, etc. is rendered directly to the terminal. Malicious or untrusted input containing raw ANSI escape sequences can:
- Rewrite arbitrary screen regions
- Change terminal title
- Inject OSC sequences (clipboard access on some terminals)
- Cause denial-of-service via excessive output

While tview's color tag system processes `[color]` tags, raw `\x1b[...` sequences pass through to the terminal unchanged.

**Fix direction:** Strip or escape raw ANSI sequences from all user-supplied text at the binding layer in `cui.go`, before passing to widget methods.

---

### 4.2 `widgetPtr.Copy()` returns self (shared mutable state) — low

```go
func (o *widgetPtr) Copy() vm.Object { return o }
```

`ImmutableMap` values can be copied in rumo scripts, but the underlying widget pointer is shared, not cloned. This is likely intentional (widgets are inherently singleton), but it violates the expectation that `copy()` produces an independent value. A routine that `copy()`s a widget map and mutates it (e.g., adds keys) will affect all holders.

**Fix direction:** Document that widgets are reference types and `copy()` does not deep-clone. Alternatively, have `Copy()` return a new `ImmutableMap` with the same `__widget` pointer (shallow wrapper copy).

---

## 5. Compatibility

### 5.1 `reflect` import in `form.go` — low

`form.go` imports `reflect`, which increases binary size and can cause issues with certain build constraints (e.g., TinyGo). Audit whether the reflect usage is necessary or can be replaced with type assertions.

---

### 5.2 `UnifyEnterKeys` global in `bind.go` — low

```go
var UnifyEnterKeys = true
```

This is a package-level mutable global that changes key interpretation behavior. It's not protected by any synchronization and could cause inconsistent behavior if changed at runtime while events are being processed.

**Fix direction:** Make it a configuration option on `BindConfig` rather than a global, or make it a `const`.

---

## Summary

| #    | Issue | Severity | Category |
|------|-------|----------|----------|
| 1.1  | Stale context in callbacks | critical | thread safety |
| 1.2  | Callback errors discarded | critical | thread safety |
| 2.1  | App.Run() ignores context cancellation | critical | abort propagation |
| 1.3  | Dual lock hierarchies | high | thread safety |
| 1.4  | FocusManager calls setFocus under lock | high | thread safety |
| 1.5  | SetFocus lock juggling / TOCTOU | high | thread safety |
| 1.6  | App.Stop() double-call deadlock | high | thread safety |
| 1.7  | List.GetItem() missing lock | high | thread safety |
| 1.8  | Unsynchronized globals (Styles, TabSize) | medium | thread safety |
| 1.9  | Unlock-callback-relock pattern | medium | thread safety |
| 1.10 | Unprotected TextView setters | medium | thread safety |
| 1.11 | Unprotected SetHighlightedFunc | medium | thread safety |
| 3.1  | Per-widget closure allocation | medium | performance |
| 3.2  | wrapTreeNode re-wraps per callback | medium | performance |
| 3.3  | QueueUpdate channel backpressure | medium | performance |
| 4.1  | No ANSI escape sanitization | medium | security |
| 4.2  | widgetPtr.Copy() returns self | low | security |
| 5.1  | reflect import in form.go | low | compatibility |
| 5.2  | UnifyEnterKeys mutable global | low | compatibility |

