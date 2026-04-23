---
title: Standard Library - json
---

## Import

```golang
json := import("json")
```

## Functions

- `decode(data bytes|string) => any`: decodes JSON-encoded data and returns the resulting value
- `encode(v any) => bytes`: returns the JSON encoding of v
- `indent(data bytes|string, prefix string, indent string) => bytes`: returns an indented form of the JSON-encoded data
- `html_escape(data bytes|string) => bytes`: returns the JSON-encoded data with &, <, and > characters escaped to \u0026, \u003c, and \u003e
