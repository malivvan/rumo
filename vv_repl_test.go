package vv_test

import (
	"bytes"
	"context"
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

