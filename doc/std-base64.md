---
title: Standard Library - base64
---

## Import

```golang
base64 := import("base64")
```

## Functions

- `encode(b bytes) => string`: returns the base64 encoding of src
- `decode(s string) => bytes`: returns the bytes represented by the base64 string s
- `raw_encode(b bytes) => string`: returns the unpadded base64 encoding of src
- `raw_decode(s string) => bytes`: returns the bytes represented by the unpadded base64 string s
- `url_encode(b bytes) => string`: returns the base64 encoding of src using URL and Filename safe alphabet
- `url_decode(s string) => bytes`: returns the bytes represented by the base64 URL and Filename safe string s
- `raw_url_encode(b bytes) => string`: returns the unpadded base64 encoding of src using URL and Filename safe alphabet
- `raw_url_decode(s string) => bytes`: returns the bytes represented by the unpadded base64 URL and Filename safe string s
