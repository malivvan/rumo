# switch

Rumo's `switch` statement is Go-style: each case auto-breaks (no implicit
fallthrough), and an explicit `fallthrough` keyword can be used to continue
into the next case body.

## Syntax

```
switch [init;] [tag] {
case expr1, expr2, ...:
    // statements
case expr3:
    // statements
    fallthrough
default:
    // statements
}
```

* The optional `init` is a simple statement (typically a short variable
  declaration) scoped to the switch.
* The optional `tag` expression is compared with `==` against each case
  expression. If `tag` is omitted, each case expression is evaluated as a
  boolean (Go-style `switch true`).
* A `case` may list multiple expressions separated by commas; the case matches
  if any one of them matches.
* `default` runs when no case matches. It may appear in any position; only one
  is allowed per `switch`.
* `break` inside a switch exits the switch (not any enclosing loop).
* Each case body has its own lexical scope.
* `fallthrough` must be the last statement in a non-final case; it transfers
  control unconditionally to the start of the next case body without
  re-evaluating its case expressions.

## Examples

Basic tag switch:

```rumo
switch day {
case "sat", "sun":
    out = "weekend"
default:
    out = "weekday"
}
```

Tagless switch (boolean cases):

```rumo
switch {
case x < 0:
    out = "negative"
case x == 0:
    out = "zero"
default:
    out = "positive"
}
```

With init statement:

```rumo
switch n := len(items); {
case n == 0:
    out = "empty"
case n < 10:
    out = "small"
default:
    out = "big"
}
```

Explicit fallthrough:

```rumo
switch n {
case 1:
    out += "a"
    fallthrough
case 2:
    out += "b"   // executed when n == 1 or n == 2
case 3:
    out += "c"
}
```

