// Demo showing how to run an SSH server backed by a shell.Instance (the rumo
// line-editor), while displaying an in-process SSH client connection in a cui
// terminal widget. No external process is spawned anywhere.
//
// Architecture:
//
//	┌──────────────────────────────────────────────────────────┐
//	│                    cui Application                       │
//	│  ┌─────────────┐  ┌──────────────────────────────────┐  │
//	│  │  Server Log  │  │  vte.Terminal (SSH client session)│  │
//	│  │  (TextView)  │  │                                  │  │
//	│  └─────────────┘  └──────────┬───────────────────────┘  │
//	│                               │                          │
//	│                    ┌──────────┴───────────┐              │
//	│                    │  sshPty wrapper       │              │
//	│                    │  (io.ReadWriteCloser) │              │
//	│                    └──────────┬───────────┘              │
//	│                               │ golang.org/x/crypto/ssh  │
//	│                    ┌──────────┴───────────┐              │
//	│                    │  SSH Server           │              │
//	│                    │  (charmbracelet/ssh)  │              │
//	│                    │  handler:             │              │
//	│                    │    shell.Instance     │              │
//	│                    │    (no subprocess)    │              │
//	│                    └──────────────────────┘              │
//	└──────────────────────────────────────────────────────────┘
//
//go:build !windows

package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	charmssh "github.com/charmbracelet/ssh"
	"github.com/malivvan/rumo/std/cui"
	"github.com/malivvan/rumo/std/cui/vte"
	"github.com/malivvan/rumo/std/shell"
	gossh "golang.org/x/crypto/ssh"
)

// sshPty wraps an SSH client session as an io.ReadWriteCloser with Resize
// support, suitable for use with vte.NewTerminalFromPty.
type sshPty struct {
	session *gossh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
}

func (s *sshPty) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *sshPty) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *sshPty) Close() error {
	_ = s.stdin.Close()
	return s.session.Close()
}

// Resize sends a window-change request to the SSH server.
// VT calls Resize(w, h); gossh.Session.WindowChange takes (h, w).
func (s *sshPty) Resize(w, h int) error {
	return s.session.WindowChange(h, w)
}

func main() {
	app := cui.NewApp()
	defer app.HandlePanic()
	app.EnableMouse(true)

	// Log view for SSH server events.
	logView := cui.NewTextView()
	logView.SetDynamicColors(true)
	logView.SetScrollable(true)
	logView.SetBorder(true)
	logView.SetTitle(" Server Log ")
	logView.SetChangedFunc(func() { app.Draw() })

	logf := func(format string, args ...interface{}) {
		fmt.Fprintf(logView, "[gray]> [-]"+format+"\n", args...)
	}

	// Bind a TCP listener on a random port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %s", err))
	}
	addr := ln.Addr().String()

	// Configure the SSH server. The handler uses shell.Instance directly
	// with the SSH session as its I/O — no subprocess is spawned.
	srv := &charmssh.Server{
		Handler: func(s charmssh.Session) {
			logf("[green]new session[-] from %s", s.RemoteAddr())

			ptyReq, winCh, isPty := s.Pty()
			if !isPty {
				io.WriteString(s, "No PTY requested.\n")
				s.Exit(1)
				logf("[red]rejected[-] %s (no PTY)", s.RemoteAddr())
				return
			}

			// Track the current window size for shell.Instance.
			var winSize struct {
				sync.Mutex
				width, height int
			}
			winSize.width = ptyReq.Window.Width
			winSize.height = ptyReq.Window.Height

			cfg := &shell.Config{
				Prompt: "\033[32mrumo\033[0m> ",

				// I/O goes directly through the SSH session channel.
				// With emulated PTY, the session wraps writes with \n → \r\n.
				Stdin:  s,
				Stdout: s,
				Stderr: s.Stderr(),

				// The SSH channel behaves like a terminal.
				FuncIsTerminal: func() bool { return true },

				// No-ops: the SSH session handles terminal mode negotiation;
				// no real terminal fd to put into raw mode.
				FuncMakeRaw: func() error { return nil },
				FuncExitRaw: func() error { return nil },

				// Report the size from the most recent PTY/window-change request.
				FuncGetSize: func() (int, int) {
					winSize.Lock()
					defer winSize.Unlock()
					return winSize.width, winSize.height
				},

				// Forward window-change notifications from the SSH client.
				FuncOnWidthChanged: func(f func()) {
					go func() {
						for win := range winCh {
							winSize.Lock()
							winSize.width = win.Width
							winSize.height = win.Height
							winSize.Unlock()
							f()
						}
					}()
				},

				Undo: true,
			}

			inst, err := shell.NewFromConfig(cfg)
			if err != nil {
				logf("[red]error[-] creating shell: %s", err)
				return
			}
			defer inst.Close()

			fmt.Fprintf(inst.Stdout(), "Welcome to the rumo shell demo over SSH!\nType 'exit' or press Ctrl-D to quit.\n")

			for {
				line, err := inst.ReadLine()
				if err != nil {
					if err == io.EOF {
						fmt.Fprintf(inst.Stdout(), "\nGoodbye!\n")
					}
					break
				}
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if line == "exit" || line == "quit" {
					fmt.Fprintf(inst.Stdout(), "Goodbye!\n")
					break
				}
				fmt.Fprintf(inst.Stdout(), "you typed: %s\n", line)
			}
			logf("[blue]session closed[-] %s", s.RemoteAddr())
		},
	}

	// Start the SSH server in the background.
	go func() {
		logf("SSH server listening on [yellow]%s[-]", addr)
		if err := srv.Serve(ln); err != nil && err != charmssh.ErrServerClosed {
			logf("[red]SSH server error:[-] %s", err)
		}
	}()

	// Connect an in-process SSH client to the server.
	clientConfig := &gossh.ClientConfig{
		User:            "demo",
		Auth:            []gossh.AuthMethod{}, // server has no auth
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	client, err := gossh.Dial("tcp", addr, clientConfig)
	if err != nil {
		panic(fmt.Sprintf("SSH client dial failed: %s", err))
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		panic(fmt.Sprintf("SSH client session failed: %s", err))
	}

	// Request a PTY on the remote side.
	if err := session.RequestPty("xterm-256color", 24, 80, gossh.TerminalModes{}); err != nil {
		panic(fmt.Sprintf("SSH client pty request failed: %s", err))
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		panic(fmt.Sprintf("SSH client stdin pipe failed: %s", err))
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("SSH client stdout pipe failed: %s", err))
	}

	if err := session.Shell(); err != nil {
		panic(fmt.Sprintf("SSH client shell request failed: %s", err))
	}

	// Wrap the SSH session as an io.ReadWriteCloser with Resize support.
	sshRWC := &sshPty{
		session: session,
		stdin:   stdin,
		stdout:  stdout,
	}

	// Create a VTE terminal widget driven by the SSH client session.
	term := vte.NewTerminalFromPty(app, sshRWC)
	term.SetBorder(true)
	term.SetTitle(" SSH Client → rumo shell (no subprocess) ")

	// Layout: log on the left, terminal on the right.
	layout := cui.NewFlex()
	layout.AddItem(logView, 0, 1, false)
	layout.AddItem(term, 0, 2, true)

	app.SetRoot(layout, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

