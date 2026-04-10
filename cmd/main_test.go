package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
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
// cleanup (e.g. terminal restore in cui/shell modules). The fix is to use
// signal.NotifyContext so that signals cancel the context, allowing the VM
// to abort gracefully and run deferred cleanup.
func TestSignalCausesGracefulShutdown(t *testing.T) {
	// When running as a subprocess, act as the CLI and run a script.
	if scriptFile := os.Getenv("RUMO_SIGNAL_TEST_SCRIPT"); scriptFile != "" {
		code := execute([]string{scriptFile}, os.Stdin, os.Stdout, os.Stderr)
		os.Exit(code)
	}

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "loop.rumo")
	if err := os.WriteFile(inputFile, []byte("for { }"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	// Launch ourselves as a subprocess with the env var set so it enters
	// the helper branch above and runs the infinite-loop script.
	cmd := exec.Command(os.Args[0], "-test.run=^TestSignalCausesGracefulShutdown$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "RUMO_SIGNAL_TEST_SCRIPT="+inputFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}

	// Give the subprocess time to start the script.
	time.Sleep(500 * time.Millisecond)

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
		code := execute([]string{scriptFile}, os.Stdin, os.Stdout, os.Stderr)
		os.Exit(code)
	}

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "loop.rumo")
	if err := os.WriteFile(inputFile, []byte("for { }"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestSIGTERMCausesGracefulShutdown$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "RUMO_SIGTERM_TEST_SCRIPT="+inputFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

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
