package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
