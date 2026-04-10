// Demo showing how to run a shell.Instance (the rumo line-editor) inside a
// cui terminal widget using a pty, without spawning any external process.
//
// Architecture:
//
//	┌────────────────────┐       ┌──────────────────────┐
//	│   shell.Instance   │◄─────►│  pty slave (*os.File) │
//	│  (readline loop)   │ stdin │                      │
//	│                    │ stdout│                      │
//	└────────────────────┘       └──────────┬───────────┘
//	                                        │ kernel pty layer
//	                             ┌──────────┴───────────┐
//	                             │  pty master (Pty)     │
//	                             │  io.ReadWriteCloser   │
//	                             └──────────┬───────────┘
//	                                        │
//	                             ┌──────────┴───────────┐
//	                             │  vte.Terminal widget  │
//	                             │  (cui application)    │
//	                             └──────────────────────┘
//
//go:build !windows

package main

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/malivvan/rumo/std/cui"
	"github.com/malivvan/rumo/std/cui/vte"
	"github.com/malivvan/rumo/std/cui/vte/pty"
	"github.com/malivvan/rumo/std/shell"
	"github.com/malivvan/rumo/std/shell/term"
)

func main() {
	// Create a pty pair.
	p, err := pty.New()
	if err != nil {
		panic(fmt.Sprintf("failed to open pty: %s", err))
	}
	defer p.Close()

	// Get the slave side for the shell instance to use as its terminal.
	up := p.(pty.UnixPty)
	slave := up.Slave()
	slaveFd := int(slave.Fd())

	// rawState guards the slave's raw-mode state so that shell.Instance
	// can call MakeRaw/Restore on the pty slave instead of real stdin.
	var rawState struct {
		sync.Mutex
		state *term.State
	}

	// Configure the shell instance to use the slave side of the pty.
	cfg := &shell.Config{
		Prompt: "\033[32mrumo\033[0m> ",

		// I/O goes through the pty slave.
		Stdin:  slave,
		Stdout: slave,
		Stderr: slave,

		// The slave fd is always a terminal.
		FuncIsTerminal: func() bool { return true },

		// Put the slave into raw mode (shell needs this for char-at-a-time input).
		FuncMakeRaw: func() error {
			rawState.Lock()
			defer rawState.Unlock()
			st, err := term.MakeRaw(slaveFd)
			if err != nil {
				return err
			}
			rawState.state = st
			return nil
		},
		FuncExitRaw: func() error {
			rawState.Lock()
			defer rawState.Unlock()
			if rawState.state == nil {
				return nil
			}
			err := term.Restore(slaveFd, rawState.state)
			if err == nil {
				rawState.state = nil
			}
			return err
		},

		// Report the slave's terminal size.
		FuncGetSize: func() (int, int) {
			w, h, err := term.GetSize(slaveFd)
			if err != nil {
				return 80, 24
			}
			return w, h
		},

		// No-op: the VTE widget handles resize via pty.Resize and the
		// shell will pick up the new size the next time it queries.
		FuncOnWidthChanged: func(func()) {},

		Undo: true,
	}

	inst, err := shell.NewFromConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create shell instance: %s", err))
	}

	// Run the readline loop in a goroutine. It drives the slave side of the
	// pty. When the user types a line, we echo it back as a simple demo.
	go func() {
		defer func() {
			inst.Close()
			// Close the slave so the master sees EOF and the VT fires EventClosed.
			slave.Close()
		}()

		fmt.Fprintf(inst.Stdout(), "Welcome to the rumo shell demo!\r\nType 'exit' or press Ctrl-D to quit.\r\n")

		for {
			line, err := inst.ReadLine()
			if err != nil {
				if err == io.EOF {
					fmt.Fprintf(inst.Stdout(), "\r\nGoodbye!\r\n")
				}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if line == "exit" || line == "quit" {
				fmt.Fprintf(inst.Stdout(), "Goodbye!\r\n")
				return
			}
			// Echo the input back as a simple demonstration.
			fmt.Fprintf(inst.Stdout(), "you typed: %s\r\n", line)
		}
	}()

	// Set up the cui application with the VTE terminal widget.
	app := cui.NewApp()
	defer app.HandlePanic()

	app.EnableMouse(true)

	terminal := vte.NewTerminalFromPty(app, p)
	terminal.SetBorder(true)
	terminal.SetTitle(" rumo shell (no subprocess) ")

	app.SetRoot(terminal, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

