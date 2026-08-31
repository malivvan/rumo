---
title: Standard Library - crand
---

## Import

```golang
crand := import("crand")
```

## Functions

- `read(b bytes) => error`: fills b with cryptographically secure random bytes
- `int() => int`: returns a cryptographically secure random int in [0, 2^63)
- `intn(n int) => int`: returns a cryptographically secure random int in [0, n)
