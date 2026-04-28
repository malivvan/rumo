# select

A `select` statement waits on multiple channel operations and runs the body
of the first one that becomes ready.  It mirrors Go's `select`, but uses
rumo's `chan.send`/`chan.recv` method syntax instead of the `<-` operator.

```go
select {
case ch.recv():            // receive, value discarded
    // ...
case v := ch.recv():       // receive with assignment
    // ...
case v, ok := ch.recv():   // receive with ok flag
    // ok == false ⇒ channel was closed and v is undefined
case ch.send(value):       // send
    // ...
default:                   // optional: fires when no other case is ready
    // ...
}
```

## Semantics

- All channel and value expressions inside the cases are evaluated **once**,
  in source order, **before** the select chooses a case.
- If a `default` clause is present, the select is non-blocking: when no
  other case is immediately ready, the default body runs.
- If no `default` is present, the select blocks until at least one case is
  ready.  When several cases are ready, one is chosen pseudo-randomly
  (Go's `reflect.Select` semantics).
- A `break` statement inside a case body exits the surrounding select.
- `continue` and `fallthrough` are not allowed inside `select`.
- Per-case bindings (`v`, `ok`) are scoped to that case body only.
- Only locally-created channels (those returned by `chan(...)`) may
  participate in a select.  Channels backed by remote transports raise a
  runtime error.

## Examples

Choose between two channels:

```go
a := chan(1)
b := chan(1)
b.send("hi")
select {
case v := a.recv():
    fmt.println("a:", v)
case v := b.recv():
    fmt.println("b:", v)   // prints: b: hi
}
```

Non-blocking receive:

```go
c := chan(0)
select {
case v := c.recv():
    fmt.println("got", v)
default:
    fmt.println("no data")  // prints when c has nothing
}
```

Detect a closed channel:

```go
c := chan(0)
c.close()
select {
case v, ok := c.recv():
    if !ok {
        fmt.println("closed")  // prints
    } else {
        fmt.println("got", v)
    }
}
```

See also: [chan](chan.md).

