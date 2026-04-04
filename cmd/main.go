package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/malivvan/vv"
)

var version string

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	ctx := context.Background()
	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "run":
		if len(os.Args) < 3 {
			println("usage: vv run <input_file>")
			os.Exit(1)
		}
		inputFile := os.Args[2]
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file %s: %s\n", inputFile, err.Error())
			os.Exit(1)
		}
		if string(data[:len(vv.Magic)]) == vv.Magic {
			err = vv.RunCompiled(ctx, data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error compiling %s: %s\n", inputFile, err.Error())
				os.Exit(1)
			}
		}
		err = vv.CompileAndRun(ctx, data, inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error compiling %s: %s\n", inputFile, err.Error())
			os.Exit(1)
		}
	case "build":
		if len(os.Args) < 4 {
			println("usage: vv build <input_file> <output_file>")
			os.Exit(1)
		}
		inputFile := os.Args[2]
		outputFile := os.Args[3]
		if outputFile == "" {
			outputFile = filepath.Base(inputFile) + ".out"
		}
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file %s: %s\n", inputFile, err.Error())
			os.Exit(1)
		}
		if err := vv.CompileOnly(data, inputFile, outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error compiling %s: %s\n", inputFile, err.Error())
			os.Exit(1)
		}
		fmt.Printf("Compiled %s to %s\n", inputFile, outputFile)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [build|run|version] [OPTIONS]\n", os.Args[0])
}
