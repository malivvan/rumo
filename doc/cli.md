---
title: cli
---

rumo is designed as an embedding script language for Go, but it also ships with a small CLI in `cmd/` for running source files, running compiled bytecode, building compiled bytecode files, and starting a REPL.

## Building the CLI

This repository currently exposes the CLI from `./cmd`, so the most reliable way to install or build a `rumo` binary is:

```bash
go build -o rumo ./cmd
# or
make build
```

If you want it on your `PATH`:

```bash
go build -o /usr/local/bin/rumo ./cmd
```

Or, you can download the precompiled binaries from
[the latest release](https://github.com/malivvan/rumo/releases/latest).

## Commands

```bash
rumo <file>
rumo run <file>
rumo build <file> [output_file]
rumo -o <output_file> <file>
rumo version
rumo help
```

- `rumo <file>` runs either a `.rumo` source file or compiled rumo bytecode file.
- `rumo run <file>` does the same explicitly.
- `rumo build <file> [output_file]` compiles a source file into rumo bytecode.
- `rumo -o <output_file> <file>` is a shorthand for choosing the build output file.
- `rumo version` prints the embedded version string.
- Running `rumo` with no arguments starts the REPL.

## Compiling and Executing rumo Code

You can directly execute rumo source code:

```bash
rumo myapp.rumo
# equivalent to
rumo run myapp.rumo
```

You can also compile the code into a rumo bytecode file and execute it later:

```bash
rumo build myapp.rumo myapp.out
rumo myapp.out
```

The `-o` shorthand is also supported:

```bash
rumo -o myapp.out myapp.rumo
rumo myapp.out
```

## Resolving Relative Import Paths

When file imports are enabled by the embedding application, relative imports are
resolved from the importing file. This version also rejects imports that escape
that initial import root.

## Rumo REPL

You can run the rumo [REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop)
by running `rumo` with no arguments.

```bash
rumo
```
