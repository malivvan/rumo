package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/std/shell"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 {
		rumo.RunREPL(ctx, in, out, ">> ")
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		usage(errOut)
		return 0
	case "version":
		_, _ = fmt.Fprintln(out, rumo.Version())
		return 0
	case "shell":
		// see readline.NewFromConfig for advanced options:
		rl, err := shell.New("> ")
		if err != nil {
			log.Fatal(err)
		}
		defer rl.Close()
		log.SetOutput(rl.Stderr()) // redraw the prompt correctly after log output

		for {
			line, err := rl.ReadLine()
			// `err` is either nil, io.EOF, readline.ErrInterrupt, or an unexpected
			// condition in stdin:
			if err != nil {
				panic(err)
			}
			// `line` is returned without the terminating \n or CRLF:
			fmt.Fprintf(rl, "you wrote: %s\n", line)
		}

	case "run":
		if len(args) != 2 {
			_, _ = fmt.Fprintln(errOut, "usage: rumo run <input_file>")
			return 1
		}
		return runFile(ctx, args[1], errOut)
	case "build":
		switch len(args) {
		case 2:
			return buildFile(args[1], "", out, errOut)
		case 3:
			return buildFile(args[1], args[2], out, errOut)
		default:
			_, _ = fmt.Fprintln(errOut, "usage: rumo build <input_file> [output_file]")
			return 1
		}
	case "-o":
		if len(args) != 3 {
			_, _ = fmt.Fprintln(errOut, "usage: rumo -o <output_file> <input_file>")
			return 1
		}
		return buildFile(args[2], args[1], out, errOut)
	default:
		return runFile(ctx, args[0], errOut)
	}
}

func usage(out io.Writer) {
	_, _ = fmt.Fprintf(out, "usage: %s [run <file> | build <file> [output] | -o <output> <file> | version]\n", os.Args[0])
}

func runFile(ctx context.Context, inputFile string, errOut io.Writer) int {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "Error reading input file %s: %s\n", inputFile, err.Error())
		return 1
	}

	if bytes.HasPrefix(data, []byte(rumo.Magic)) {
		if err := rumo.RunCompiled(ctx, data); err != nil {
			_, _ = fmt.Fprintf(errOut, "Error running %s: %s\n", inputFile, err.Error())
			return 1
		}
		return 0
	}

	if err := rumo.CompileAndRun(ctx, data, inputFile); err != nil {
		_, _ = fmt.Fprintf(errOut, "Error compiling %s: %s\n", inputFile, err.Error())
		return 1
	}
	return 0
}

func buildFile(inputFile, outputFile string, out, errOut io.Writer) int {
	if outputFile == "" {
		outputFile = filepath.Base(inputFile) + ".out"
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "Error reading input file %s: %s\n", inputFile, err.Error())
		return 1
	}

	if err := rumo.CompileOnly(data, inputFile, outputFile); err != nil {
		_, _ = fmt.Fprintf(errOut, "Error compiling %s: %s\n", inputFile, err.Error())
		return 1
	}

	_, _ = fmt.Fprintf(out, "Compiled %s to %s\n", inputFile, outputFile)
	return 0
}
