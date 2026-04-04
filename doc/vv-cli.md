---
title: cli
---


VV is designed as an embedding script language for Go, but, it can also be
compiled and executed as native binary using `vv` CLI tool.

## Installing VV CLI

To install `vv` tool, run:

```bash
go get github.com/malivvan/vv/cmd/vv
```

Or, you can download the precompiled binaries from
[here](https://github.com/malivvan/vv/releases/latest).

## Compiling and Executing VV Code

You can directly execute the VV source code by running `vv` tool with
your VV source file (`*.vv`).

```bash
vv myapp.vv
```

Or, you can compile the code into a binary file and execute it later.

```bash
vv -o myapp myapp.vv   # compile 'myapp.vv' into binary file 'myapp'
vv myapp                  # execute the compiled binary `myapp`
```

Or, you can make vv source file executable

```bash
# copy vv executable to a dir where PATH environment variable includes
cp vv /usr/local/bin/

# add shebang line to source file
cat > myapp.vv << EOF
#!/usr/local/bin/vv
fmt := import("fmt")
fmt.println("Hello World!")
EOF

# make myapp.vv file executable
chmod +x myapp.vv

# run your script
./myapp.vv
```

**Note: Your source file must have `.vv` extension.**

## Resolving Relative Import Paths

If there are vv source module files which are imported with relative import
paths, CLI has `-resolve` flag. Flag enables to import a module relative to
importing file. This behavior will be default at version 3.

## VV REPL

You can run VV [REPL](https://en.wikipedia.org/wiki/Read–eval–print_loop)
if you run `vv` with no arguments.

```bash
vv
```
