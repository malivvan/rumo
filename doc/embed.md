---
title: embed
---


## Table of Contents

- [Overview](#overview)
- [Enabling Embed](#enabling-embed)
- [Embedding a Single File as String](#embedding-a-single-file-as-string)
- [Embedding a Single File as Bytes](#embedding-a-single-file-as-bytes)
- [Embedding Multiple Files as a Map of Strings](#embedding-multiple-files-as-a-map-of-strings)
- [Embedding Multiple Files as a Map of Bytes](#embedding-multiple-files-as-a-map-of-bytes)
- [Glob Patterns](#glob-patterns)
- [Multiple Patterns](#multiple-patterns)
- [Map Keys](#map-keys)
- [Errors](#errors)

---

## Overview

Rumo supports compile-time file embedding inspired by Go's `//go:embed`
directive. By placing a special `//embed` comment directly above a `:=`
variable declaration, the compiler reads the referenced file(s) from disk at
**compile time** and bakes their contents into the compiled bytecode as
constants. No file I/O happens at runtime.

The format of the directive is:

```
//embed <pattern> [pattern …]
```

where each `pattern` is a file path or a [glob pattern](#glob-patterns)
relative to the directory of the source file being compiled.

The **type** of the embedded data is determined by the placeholder expression
on the right-hand side of the `:=` declaration:

| Placeholder | Result type            | Use case                            |
|-------------|------------------------|-------------------------------------|
| `""`        | `string`               | Single file contents as a string    |
| `bytes("")` | `bytes`                | Single file contents as a byte slice|
| `{}`        | `map` of `string`      | Multiple files, values are strings  |
| `bytes({})` | `map` of `bytes`       | Multiple files, values are bytes    |

---

## Enabling Embed

Embed directives are resolved relative to the **source file directory**, which
must be configured on the `Script` before compilation. Call `SetImportDir` with
the directory that contains the source file (or with any base directory of your
choice):

```go
s := rumo.NewScript(src)
if err := s.SetImportDir("/path/to/scripts"); err != nil {
    log.Fatal(err)
}
compiled, err := s.Compile()
```

If `SetImportDir` is not called (or called with an empty string) and an
`//embed` directive is present in the script, compilation returns an error:

```
embed: file embed is not available (no source directory)
```

When using the `rumo` CLI tool (`rumo run script.rumo`), the import directory
is automatically set to the directory that contains the script file, so embed
directives work out of the box.

---

## Embedding a Single File as String

Place `//embed <file>` immediately above a `:=` assignment whose right-hand
side is an empty string literal `""`. The variable will hold the full UTF-8
text content of the file.

```rumo
//embed templates/header.html
header := ""

fmt.println(header)
```

The variable `header` is a **string** constant at runtime.

---

## Embedding a Single File as Bytes

Use `bytes("")` as the placeholder to receive the raw file content as a
`bytes` value instead of a string.

```rumo
//embed assets/logo.png
logoData := bytes("")

fmt.println(len(logoData), "bytes read")
```

The variable `logoData` is a **bytes** constant at runtime.

---

## Embedding Multiple Files as a Map of Strings

Use an empty map literal `{}` as the placeholder to embed all files matching
one or more patterns. The result is a **map** where each key is the file's
path relative to the source directory and each value is the file's text
content as a string.

```rumo
//embed docs/*.md
docs := {}

for name, content in docs {
    fmt.println(name, "->", len(content), "chars")
}
```

---

## Embedding Multiple Files as a Map of Bytes

Use `bytes({})` as the placeholder to get the same map but with raw `bytes`
values instead of strings.

```rumo
//embed assets/*.png
images := bytes({})

for name, data in images {
    fmt.println(name, "->", len(data), "bytes")
}
```

---

## Glob Patterns

Each pattern after `//embed` is expanded at compile time using standard
[filepath.Glob](https://pkg.go.dev/path/filepath#Glob) rules. Glob patterns
are relative to the source file's directory.

| Pattern        | Matches                                              |
|----------------|------------------------------------------------------|
| `hello.txt`    | Exactly the file `hello.txt`                         |
| `*.md`         | All `.md` files in the source directory              |
| `docs/*.html`  | All `.html` files inside the `docs/` subdirectory   |
| `config.json`  | Exactly the file `config.json`                       |

> **Note:** Recursive glob (`**`) is not supported — `filepath.Glob` only
> matches files in the single directory specified by the pattern. To embed
> files from multiple subdirectories, list each pattern explicitly.

Absolute paths are never allowed and produce a compile error.

---

## Multiple Patterns

A single `//embed` directive can list several patterns separated by
whitespace. All matched files are merged into the result.

Single-file embed with a precise path (no glob needed):

```rumo
//embed README.md
readme := ""
```

Multi-file embed combining a glob and an exact file:

```rumo
//embed src/*.rumo lib/helpers.rumo
sources := {}
```

The resulting map contains entries for every file matched by **any** of the
listed patterns.

---

## Map Keys

When embedding multiple files, the key for each entry in the result map is the
path of the matched file **relative to the source file's directory**, with
forward slashes (`/`) as the separator regardless of the host operating system.

For example, if the source file lives in `/project/scripts/` and the pattern
`data/*.json` matches `/project/scripts/data/config.json`, the map key will
be `data/config.json`.

```rumo
//embed data/*.json
configs := {}

cfgText := configs["data/config.json"]
```

---

## Errors

The following conditions cause a **compile-time** error:

| Condition | Error message |
|-----------|---------------|
| No import/source directory configured | `embed: file embed is not available (no source directory)` |
| No patterns listed in the directive | `embed: no patterns specified` |
| A pattern matches no files | `embed: no files matched pattern "..."` |
| An invalid glob pattern | `embed: invalid glob pattern "...": ...` |
| An absolute path used as a pattern | `embed: absolute paths are not allowed: ...` |
| A glob matches >1 file but the placeholder is a scalar (`""` / `bytes("")`) | `embed: single-file embed matched N files (expected 1)` |
| More than one variable on the left-hand side | `embed: exactly one variable must be on the left-hand side` |
| `//embed` used above a non-`:=` statement | *(directive is silently ignored — only `:=` declarations are affected)* |

Because all errors are caught at compile time, a successfully compiled script
is guaranteed to have its embedded data available with no possibility of
missing-file errors at runtime.

