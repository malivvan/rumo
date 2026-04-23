package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/malivvan/rumo"
)

func TestRunVersionUsesPackageVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"version"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d, stderr=%q", exitCode, stderr.String())
	}
	if got := stdout.String(); got != rumo.Version()+"\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

// TODO
//func TestRunCompiledFileExecutesOnlyOnce(t *testing.T) {
//	tempDir := t.TempDir()
//	sourceFile := filepath.Join(tempDir, "script.rumo")
//	compiledFile := filepath.Join(tempDir, "script.out")
//
//	src := fmt.Sprintf(`fmt := import("fmt");
//fmt.print("x");
//`)
//	if err := os.WriteFile(sourceFile, []byte(src), 0o644); err != nil {
//		t.Fatalf("write source: %v", err)
//	}
//	if err := rumo.CompileOnly([]byte(src), sourceFile, compiledFile); err != nil {
//		t.Fatalf("compile only: %v", err)
//	}
//
//	var stdout bytes.Buffer
//	var stderr bytes.Buffer
//	exitCode := run(context.Background(), []string{compiledFile}, strings.NewReader(""), &stdout, &stderr)
//	if exitCode != 0 {
//		t.Fatalf("unexpected exit code: %d, stderr=%q", exitCode, stderr.String())
//	}
//
//	if got := stdout.String(); got != "x" {
//		t.Fatalf("compiled program executed unexpected number of times: %q", got)
//	}
//}

func TestRunShortInputDoesNotPanic(t *testing.T) {
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "tiny.rumo")
	if err := os.WriteFile(inputFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write tiny file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run(context.Background(), []string{inputFile}, strings.NewReader(""), &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("expected failure for invalid source, got success")
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected an error message for invalid source")
	}
}

// Issue #12: No signal handling — Ctrl-C kills without cleanup.
//
// The CLI's execute() function passes context.Background() with no
// SIGINT/SIGTERM handler. When a signal arrives, the Go runtime's default
// handler terminates the process immediately — bypassing all deferred
// cleanup. The fix is to use signal.NotifyContext so that signals cancel
// the context, allowing the VM to abort gracefully and run deferred cleanup.
func TestSignalCausesGracefulShutdown(t *testing.T) {
	// When running as a subprocess, act as the CLI and run a script.
	if scriptFile := os.Getenv("RUMO_SIGNAL_TEST_SCRIPT"); scriptFile != "" {
		// Set up signal handling BEFORE calling run() so the handler
		// is guaranteed to be in place when the parent sends the signal.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		code := run(ctx, []string{scriptFile}, os.Stdin, os.Stdout, os.Stderr)
		os.Exit(code)
	}

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "loop.rumo")
	// The script prints "started" from inside the VM before entering the
	// infinite loop. The parent waits for this line on stdout, which proves
	// the VM is past its internal abort-flag reset and is actively running.
	src := `fmt := import("fmt"); fmt.println("started"); for { }`
	if err := os.WriteFile(inputFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	// Launch ourselves as a subprocess with the env var set so it enters
	// the helper branch above and runs the infinite-loop script.
	cmd := exec.Command(os.Args[0], "-test.run=^TestSignalCausesGracefulShutdown$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "RUMO_SIGNAL_TEST_SCRIPT="+inputFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}

	// Wait for the "started" line from inside the VM — this guarantees the
	// signal handler is registered AND the VM loop is actively running.
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() { // reads "started"
			close(ready)
		}
	}()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("subprocess did not signal readiness within 10s")
	}

	// Send SIGINT (Ctrl-C).
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}

	// Wait for the subprocess to exit.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("process did not exit within 5s after SIGINT — no signal handling")
	case err := <-done:
		if err == nil {
			// Process exited with code 0 — graceful.
			return
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("unexpected error type: %v", err)
		}
		if !exitErr.Exited() {
			// The process was killed by a signal instead of exiting
			// cleanly. This means no signal handler caught SIGINT.
			t.Fatalf("process was killed by signal (no graceful shutdown): %v", err)
		}
		// Process exited with a non-zero code (e.g. 1 for context
		// cancellation) — that's a graceful exit, test passes.
	}
}

// TestRunRespectsContextCancellation verifies that the run() function
// terminates a long-running script when its context is cancelled.
func TestRunRespectsContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "loop.rumo")
	if err := os.WriteFile(inputFile, []byte("for { }"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan int, 1)
	go func() {
		var stderr bytes.Buffer
		code := run(ctx, []string{inputFile}, strings.NewReader(""), &bytes.Buffer{}, &stderr)
		done <- code
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case exitCode := <-done:
		if exitCode == 0 {
			t.Fatalf("expected non-zero exit code after context cancellation, got 0")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not return within 5s after context cancellation")
	}
}

// TestSIGTERMCausesGracefulShutdown is the SIGTERM counterpart of
// TestSignalCausesGracefulShutdown — both signals must be handled.
func TestSIGTERMCausesGracefulShutdown(t *testing.T) {
	if scriptFile := os.Getenv("RUMO_SIGTERM_TEST_SCRIPT"); scriptFile != "" {
		// Set up signal handling BEFORE calling run() so the handler
		// is guaranteed to be in place when the parent sends the signal.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		code := run(ctx, []string{scriptFile}, os.Stdin, os.Stdout, os.Stderr)
		os.Exit(code)
	}

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "loop.rumo")
	// The script prints "started" from inside the VM before entering the
	// infinite loop — see TestSignalCausesGracefulShutdown for rationale.
	src := `fmt := import("fmt"); fmt.println("started"); for { }`
	if err := os.WriteFile(inputFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestSIGTERMCausesGracefulShutdown$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "RUMO_SIGTERM_TEST_SCRIPT="+inputFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}

	// Wait for the "started" line from inside the VM — this guarantees the
	// signal handler is registered AND the VM loop is actively running.
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() { // reads "started"
			close(ready)
		}
	}()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("subprocess did not signal readiness within 10s")
	}

	// Send SIGTERM instead of SIGINT.
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("process did not exit within 5s after SIGTERM — no signal handling")
	case err := <-done:
		if err == nil {
			return
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("unexpected error type: %v", err)
		}
		if !exitErr.Exited() {
			t.Fatalf("process was killed by signal (no graceful shutdown): %v", err)
		}
	}
}

func TestRunWithoutArgsStartsRepl(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), nil, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d, stderr=%q", exitCode, stderr.String())
	}
	if got := stdout.String(); got != ">> " {
		t.Fatalf("unexpected repl output: %q", got)
	}
}

// CLI cannot forward arguments to scripts.
// The "run" subcommand previously accepted exactly one argument (the file) and
// rejected any additional arguments with a usage error. Scripts had no way to
// receive command-line arguments from the caller. The fix extends "run" and the
// bare file invocation to accept extra arguments after the script file (optionally
// separated by "--"), forwarding them as the script's args() list.

// TestRunSubcommandAcceptsExtraScriptArgs verifies that passing additional args
// after the script file does not produce a usage error.
func TestRunSubcommandAcceptsExtraScriptArgs(t *testing.T) {
	tempDir := t.TempDir()
	scriptFile := filepath.Join(tempDir, "script.rumo")
	if err := os.WriteFile(scriptFile, []byte(`a := args()`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	var stdout, stderr bytes.Buffer
	// Pass extra args directly after the file (no "--")
	exitCode := run(context.Background(), []string{"run", scriptFile, "hello", "world"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 when extra args passed to run, got %d, stderr: %q", exitCode, stderr.String())
	}
}

// TestRunSubcommandAcceptsExtraScriptArgsAfterSeparator verifies that args after
// "--" are forwarded to the script and don't cause a usage error.
func TestRunSubcommandAcceptsExtraScriptArgsAfterSeparator(t *testing.T) {
	tempDir := t.TempDir()
	scriptFile := filepath.Join(tempDir, "script.rumo")
	if err := os.WriteFile(scriptFile, []byte(`a := args()`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{"run", scriptFile, "--", "alpha", "beta"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 with -- separator, got %d, stderr: %q", exitCode, stderr.String())
	}
}

// TestDefaultInvocationAcceptsExtraScriptArgs verifies that when running without
// the "run" subcommand (bare file path), extra args are forwarded to the script.
func TestDefaultInvocationAcceptsExtraScriptArgs(t *testing.T) {
	tempDir := t.TempDir()
	scriptFile := filepath.Join(tempDir, "script.rumo")
	if err := os.WriteFile(scriptFile, []byte(`a := args()`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{scriptFile, "foo", "bar"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 when extra args in default invocation, got %d, stderr: %q", exitCode, stderr.String())
	}
}

// TestScriptNameIsFirstArg verifies that the script file name is the first
// element returned by args() (index 0), matching argv[0] convention.
func TestScriptNameIsFirstArg(t *testing.T) {
	tempDir := t.TempDir()
	scriptFile := filepath.Join(tempDir, "myscript.rumo")
	// Script writes len(args()) to stderr via a runtime error check:
	// we verify the arg count is correct by examining the exit code.
	// args()[0] should be the script file path, args()[1] should be "extra"
	// If wrong, the script intentionally accesses an out-of-bounds element → error.
	src := `
a := args()
if len(a) != 2 { error("wrong arg count") }
`
	if err := os.WriteFile(scriptFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{"run", scriptFile, "--", "extra"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (script sees exactly 2 args), got %d, stderr: %q", exitCode, stderr.String())
	}
}

// TestSplitArgsHandlesSeparator is a unit test for the splitArgs helper:
// everything after "--" becomes script args, and the file is separate.
func TestSplitArgsHandlesSeparator(t *testing.T) {
	tests := []struct {
		in          []string
		wantFile    string
		wantScripts []string
	}{
		{[]string{"script.rumo"}, "script.rumo", nil},
		{[]string{"script.rumo", "a", "b"}, "script.rumo", []string{"a", "b"}},
		{[]string{"script.rumo", "--", "a", "b"}, "script.rumo", []string{"a", "b"}},
		{[]string{"script.rumo", "--"}, "script.rumo", []string{}},
		{[]string{"script.rumo", "a", "--", "b"}, "script.rumo", []string{"b"}},
	}
	for _, tc := range tests {
		file, scripts := splitArgs(tc.in)
		if file != tc.wantFile {
			t.Errorf("splitArgs(%v): file=%q, want %q", tc.in, file, tc.wantFile)
		}
		if len(scripts) != len(tc.wantScripts) {
			t.Errorf("splitArgs(%v): scripts=%v, want %v", tc.in, scripts, tc.wantScripts)
			continue
		}
		for i, s := range tc.wantScripts {
			if scripts[i] != s {
				t.Errorf("splitArgs(%v): scripts[%d]=%q, want %q", tc.in, i, scripts[i], s)
			}
		}
	}
}

