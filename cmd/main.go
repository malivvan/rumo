package main

import (
	"context"
	"fmt"
	"os"

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
	app, err := vv.NewCli(func(c *cli.Context) error {
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
