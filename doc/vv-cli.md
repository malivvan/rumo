---
title: cli
---

VV is designed as an embedding script language for Go, but it also ships with a small CLI in `cmd/` for running source files, running compiled bytecode, building compiled bytecode files, and starting a REPL.

## Building the CLI

This repository currently exposes the CLI from `./cmd`, so the most reliable way to install or build a `vv` binary is:

```bash
go build -o vv ./cmd
# or
make build
```

If you want it on your `PATH`:

```bash
go build -o /usr/local/bin/vv ./cmd
```

Or, you can download the precompiled binaries from
[the latest release](https://github.com/malivvan/vv/releases/latest).

## Commands

```bash
vv <file>
vv run <file>
vv build <file> [output_file]
vv -o <output_file> <file>
vv version
vv help
```

- `vv <file>` runs either a `.vv` source file or a compiled VV bytecode file.
- `vv run <file>` does the same explicitly.
- `vv build <file> [output_file]` compiles a source file into VV bytecode.
- `vv -o <output_file> <file>` is a shorthand for choosing the build output file.
- `vv version` prints the embedded version string.
- Running `vv` with no arguments starts the REPL.

## Compiling and Executing VV Code

You can directly execute VV source code:

```bash
vv myapp.vv
# equivalent to
vv run myapp.vv
```

You can also compile the code into a VV bytecode file and execute it later:

```bash
vv build myapp.vv myapp.out
vv myapp.out
```

The `-o` shorthand is also supported:

```bash
vv -o myapp.out myapp.vv
vv myapp.out
```

## Resolving Relative Import Paths

When file imports are enabled by the embedding application, relative imports are
resolved from the importing file. This version also rejects imports that escape
that initial import root.

## VV REPL

You can run the VV [REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop)
by running `vv` with no arguments.

```bash
vv
```
