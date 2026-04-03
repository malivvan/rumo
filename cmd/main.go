package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/malivvan/vv"
	"github.com/malivvan/vv/pkg/cli"
)

var (
	serial   string
	commit   string
	version  string
	compiled string
)

func main() {
	ctx := context.Background()
	app, err := NewCli(func(c *cli.Context) error {
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating app: %s\n", err.Error())
		os.Exit(1)
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func NewCli(ui cli.ActionFunc) (*cli.App, error) {
	app := &cli.App{
		Name:      "vv",
		Usage:     "a general-purpose programming language",
		Version:   version,
		Reader:    os.Stdin,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "home",
				Usage:   "VV home directory",
				EnvVars: []string{"VVHOME"},
				Value:   filepath.Join(os.Getenv("HOME"), ".vv"),
			},
		},
	}

	app.Action = ui
	app.Commands = []*cli.Command{
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "run a VV program",
			Action: func(ctx *cli.Context) error {
				if ctx.Args().Len() != 1 {
					return fmt.Errorf("run command requires exactly one argument")
				}
				inputFile := ctx.Args().Get(0)
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("error reading input file %s: %w", inputFile, err)
				}
				if string(data[:len(vv.Magic)]) == vv.Magic {
					return vv.RunCompiled(ctx.Context, data)
				}
				return vv.CompileAndRun(ctx.Context, data, inputFile)
			},
		},
		{
			Name:    "build",
			Aliases: []string{"b"},
			Usage:   "build a VV program",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Usage:   "output file name",
					Value:   "",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return fmt.Errorf("build command requires exactly one argument")
				}
				inputFile := c.Args().Get(0)
				outputFile := c.String("output")
				if outputFile == "" {
					outputFile = filepath.Base(inputFile) + ".out"
				}
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("error reading input file %s: %w", inputFile, err)
				}
				if err := vv.CompileOnly(data, inputFile, outputFile); err != nil {
					return fmt.Errorf("error compiling program: %w", err)
				}
				fmt.Printf("Compiled %s to %s\n", inputFile, outputFile)
				return nil
			},
		},
	}
	return app, nil
}
