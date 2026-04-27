package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/malivvan/rumo"
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func execute(args []string, in io.Reader, out, errOut io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, args, in, out, errOut)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		rumo.RunREPL(ctx, stdin, stdout, stderr, rumo.AllModuleNames())
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		usage(stderr)
		return 0
	case "version":
		_, _ = fmt.Fprintln(stdout, rumo.Version())
		return 0
	case "run":
		if len(args) < 2 {
			_, _ = fmt.Fprintln(stderr, "usage: rumo run <input_file> [-- args...]")
			return 1
		}
		file, scriptArgs := splitArgs(args[1:])
		return runFile(ctx, file, scriptArgs, stderr)
	case "build":
		switch len(args) {
		case 2:
			return buildFile(args[1], "", stdout, stderr)
		case 3:
			return buildFile(args[1], args[2], stdout, stderr)
		default:
			_, _ = fmt.Fprintln(stderr, "usage: rumo build <input_file> [output_file]")
			return 1
		}
	case "-o":
		if len(args) != 3 {
			_, _ = fmt.Fprintln(stderr, "usage: rumo -o <output_file> <input_file>")
			return 1
		}
		return buildFile(args[2], args[1], stdout, stderr)
	default:
		file, scriptArgs := splitArgs(args)
		return runFile(ctx, file, scriptArgs, stderr)
	}
}

func usage(out io.Writer) {
	_, _ = fmt.Fprintf(out, "usage: %s [run <file> | build <file> [output] | -o <output> <file> | version]\n", os.Args[0])
}

func runFile(ctx context.Context, inputFile string, scriptArgs []string, errOut io.Writer) int {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "Error reading input file %s: %s\n", inputFile, err.Error())
		return 1
	}

	// Build the args list: [scriptName, scriptArgs...]
	args := append([]string{inputFile}, scriptArgs...)

	if bytes.HasPrefix(data, []byte(rumo.Magic)) {
		if err := rumo.RunCompiled(ctx, data, args); err != nil {
			_, _ = fmt.Fprintf(errOut, "Error running %s: %s\n", inputFile, err.Error())
			return 1
		}
		return 0
	}

	if err := rumo.CompileAndRun(ctx, data, inputFile, args); err != nil {
		_, _ = fmt.Fprintf(errOut, "Error: %s: %s\n", inputFile, err.Error())
		return 1
	}
	return 0
}

// splitArgs separates [file, ...rest] into (file, scriptArgs). If "--" is present,
// everything after it becomes scriptArgs. Otherwise all remaining args after the
// file are treated as script args.
func splitArgs(args []string) (file string, scriptArgs []string) {
	file = args[0]
	rest := args[1:]
	for i, a := range rest {
		if a == "--" {
			return file, rest[i+1:]
		}
	}
	return file, rest
}

func buildFile(inputFile, outputFile string, out, errOut io.Writer) int {
	if outputFile == "" {
		outputFile = basename(inputFile) + ".out"
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

func basename(s string) string {
	s = filepath.Base(s)
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}
