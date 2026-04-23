---
title: defer statement
---

## defer

A `defer` statement schedules a function call to be executed when the
surrounding function returns. Multiple deferred calls are executed in
last-in-first-out (LIFO) order.

```golang
f := func() {
    defer cleanup()
    // ... do work ...
    // cleanup() runs after this function returns
}
```

### Execution order

Deferred calls execute in reverse order of the `defer` statements.
The last deferred call runs first:

```golang
result := []
f := func() {
    defer func() { result = append(result, 1) }()
    defer func() { result = append(result, 2) }()
    defer func() { result = append(result, 3) }()
}
f()
fmt.println(result) // [3, 2, 1]
```

### Argument evaluation

Arguments to deferred function calls are evaluated immediately when
the `defer` statement executes, not when the deferred function runs:

```golang
f := func() {
    x := 0
    defer fmt.println(x) // prints 0, not 1
    x = 1
}
```

### Return value

The return value of a function is determined before deferred calls
execute. Deferred calls cannot modify the return value:

```golang
f := func() {
    x := 10
    defer func() { x = 99 }()
    return x // returns 10, not 99
}
```

### Defer in loops

Deferred calls inside a loop accumulate and all execute when the
function returns:

```golang
f := func() {
    for i := 0; i < 5; i++ {
        defer fmt.println(i)
    }
    // prints: 4, 3, 2, 1, 0 (LIFO order)
}
```

### Common patterns

#### Resource cleanup

```golang
f := func() {
    ch := chan()
    defer ch.close()

    // use channel...
    ch.send("hello")
}
```

#### Timing

```golang
times := import("times")

timed := func(name, fn) {
    start := times.now()
    defer func() {
        elapsed := times.since(start)
        fmt.println(name, "took", elapsed)
    }()
    fn()
}
```

### Restrictions

- `defer` can only be used inside a function body.
- The deferred expression must be a function call.

```golang
// OK
defer cleanup()
defer func() { /* ... */ }()
defer obj.method()

// Error: not a function call
defer x
defer 42
```
