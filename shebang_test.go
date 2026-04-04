package vv_test

import (
	"path/filepath"
	"testing"

	"github.com/malivvan/vv"
)

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
