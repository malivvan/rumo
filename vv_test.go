package vv_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malivvan/vv"
)

func TestRunREPLWritesEvaluationToProvidedWriter(t *testing.T) {
	var out bytes.Buffer

	vv.RunREPL(context.Background(), strings.NewReader("1 + 1\n"), &out, ">> ")

	got := out.String()
	if !strings.Contains(got, "2\n") {
		t.Fatalf("expected repl output to contain evaluated result, got %q", got)
	}
	if !strings.HasPrefix(got, ">> ") {
		t.Fatalf("expected repl output to start with prompt, got %q", got)
	}
}

func TestScriptRunSupportsShebangSource(t *testing.T) {
	tempDir := t.TempDir()
	s := vv.NewScript([]byte("#!/usr/bin/env vv\nanswer := 40 + 2\n"))
	s.SetName(filepath.Join(tempDir, "script.vv"))

	p, err := s.Run()
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	if got := p.Get("answer").Int(); got != 42 {
		t.Fatalf("unexpected result: %d", got)
	}
}

func TestScriptFileImportAllowsNestedRelativeImportsWithinRoot(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "sandbox")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "shared.vv"), []byte(`export 40`), 0o644); err != nil {
		t.Fatalf("write shared module: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "entry.vv"), []byte(`base := import("../shared"); export base + 2`), 0o644); err != nil {
		t.Fatalf("write entry module: %v", err)
	}

	s := vv.NewScript([]byte(`out := import("./sub/entry")`))
	s.SetName(filepath.Join(root, "main.vv"))
	s.EnableFileImport(true)
	if err := s.SetImportDir(root); err != nil {
		t.Fatalf("set import dir: %v", err)
	}

	p, err := s.Run()
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	if got := p.Get("out").Int(); got != 42 {
		t.Fatalf("unexpected import result: %d", got)
	}
}

func TestScriptFileImportRejectsEscapingImportRoot(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "sandbox")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "outside.vv"), []byte(`export 99`), 0o644); err != nil {
		t.Fatalf("write outside module: %v", err)
	}

	s := vv.NewScript([]byte(`out := import("../outside")`))
	s.SetName(filepath.Join(root, "main.vv"))
	s.EnableFileImport(true)
	if err := s.SetImportDir(root); err != nil {
		t.Fatalf("set import dir: %v", err)
	}

	_, err := s.Compile()
	if err == nil {
		t.Fatal("expected file import traversal to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes import root") {
		t.Fatalf("unexpected error: %v", err)
	}
}
