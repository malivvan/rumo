---
title: Standard Library - rand
---

## Import

```golang
rand := import("rand")
```

## Functions

- `int() => int`: non-cryptographic random int in [0, 2^63); use crand for secrets
- `float() => float`: non-cryptographic random float in [0.0, 1.0)
- `intn(n int) => int`: non-cryptographic random int in [0, n); use crand for secrets
- `exp_float() => float`: non-cryptographic exponentially distributed float
- `norm_float() => float`: non-cryptographic normally distributed float
- `perm(n int) => []int`: non-cryptographic random permutation of [0, n)
- `read(b bytes) => error`
- `rand(seed int) => *Rand`
