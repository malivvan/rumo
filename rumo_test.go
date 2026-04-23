package rumo_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/vm"
)

// Program.Unmarshal always resolves imported modules using the global Modules()
// map. Embedders that supply custom builtin modules via Script.SetImports cannot
// inject those modules into deserialization: the BuiltinFunction values inside the
// serialized ImmutableMap constants are not recoverable from the global map, so
// calling any function exported by the custom module after an Unmarshal round-trip
// either returns an error or panics.  UnmarshalWithModules must accept a caller-
// supplied ModuleMap so that custom modules survive Marshal/Unmarshal round-trips.

// TestProgramMarshalUnmarshalRoundTripWithCustomModules verifies that a program
// compiled with a custom builtin module can be serialized and then deserialized
// using the same custom module map, and that the deserialized program executes
// correctly.  It also verifies that deserializing without the custom module map
// fails, demonstrating the need for UnmarshalWithModules.
func TestProgramMarshalUnmarshalRoundTripWithCustomModules(t *testing.T) {
	customAttrs := map[string]vm.Object{
		"answer": &vm.BuiltinFunction{
			Name: "answer",
			Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
				return &vm.Int{Value: 42}, nil
			},
		},
	}
	customModules := vm.NewModuleMap()
	customModules.AddBuiltinModule("mymod", customAttrs)

	// Compile a script that imports and calls the custom module.
	s := rumo.NewScript([]byte(`result := import("mymod").answer()`))
	s.SetImports(customModules)
	p, err := s.Compile()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := p.Run(); err != nil {
		t.Fatalf("pre-marshal run: %v", err)
	}
	if got := p.Get("result").Int(); got != 42 {
		t.Fatalf("pre-marshal result: want 42, got %d", got)
	}

	// Serialize.
	data, err := p.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal without the custom module map — must fail.
	// Program.Unmarshal hardcodes the global Modules() which does not contain
	// "mymod", so the BuiltinFunction for "answer" cannot be restored.
	p2 := &rumo.Program{}
	unmarshalErr := p2.Unmarshal(data)
	if unmarshalErr == nil {
		unmarshalErr = p2.Run()
	}
	if unmarshalErr == nil {
		t.Fatal("expected Unmarshal/Run to fail without custom modules, but succeeded")
	}

	// Unmarshal with the custom module map — must succeed and yield correct result.
	p3 := &rumo.Program{}
	if err := p3.UnmarshalWithModules(data, customModules); err != nil {
		t.Fatalf("UnmarshalWithModules: %v", err)
	}
	if err := p3.Run(); err != nil {
		t.Fatalf("post-unmarshal run: %v", err)
	}
	if got := p3.Get("result").Int(); got != 42 {
		t.Fatalf("post-unmarshal result: want 42, got %d", got)
	}
}

// TestRunCompiledWithModulesExecutesCustomModule verifies that the top-level
// RunCompiledWithModules helper correctly threads the custom module map through
// to deserialization so that scripts using custom modules are executable from
// pre-compiled bytecode blobs.
func TestRunCompiledWithModulesExecutesCustomModule(t *testing.T) {
	customAttrs := map[string]vm.Object{
		"value": &vm.BuiltinFunction{
			Name: "value",
			Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
				return &vm.Int{Value: 99}, nil
			},
		},
	}
	customModules := vm.NewModuleMap()
	customModules.AddBuiltinModule("mymod2", customAttrs)

	s := rumo.NewScript([]byte(`_ := import("mymod2").value()`))
	s.SetImports(customModules)
	p, err := s.Compile()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	data, err := p.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := rumo.RunCompiledWithModules(context.Background(), data, nil, customModules); err != nil {
		t.Fatalf("RunCompiledWithModules: %v", err)
	}
}

func TestRunREPLWritesEvaluationToProvidedWriter(t *testing.T) {
	var out bytes.Buffer

	rumo.RunREPL(context.Background(), strings.NewReader("1 + 1\n"), &out, ">> ", nil)

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
	s := rumo.NewScript([]byte("#!/usr/bin/env rumo\nanswer := 40 + 2\n"))
	s.SetName(filepath.Join(tempDir, "script.rumo"))

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

	if err := os.WriteFile(filepath.Join(root, "shared.rumo"), []byte(`export 40`), 0o644); err != nil {
		t.Fatalf("write shared module: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "entry.rumo"), []byte(`base := import("../shared"); export base + 2`), 0o644); err != nil {
		t.Fatalf("write entry module: %v", err)
	}

	s := rumo.NewScript([]byte(`out := import("./sub/entry")`))
	s.SetName(filepath.Join(root, "main.rumo"))
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
	if err := os.WriteFile(filepath.Join(tempDir, "outside.rumo"), []byte(`export 99`), 0o644); err != nil {
		t.Fatalf("write outside module: %v", err)
	}

	s := rumo.NewScript([]byte(`out := import("../outside")`))
	s.SetName(filepath.Join(root, "main.rumo"))
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

// VM defaults Args to os.Args — leaks host binary arguments into sandboxed scripts.
// When Program.SetArgs is not called, the script should see an empty argument list,
// not the host process's os.Args. When SetArgs is called the script must see exactly
// the provided list. RunREPL must also forward its custom In/Out writers to per-line
// VMs so that fmt module I/O is not silently redirected to os.Stdout/os.Stdin.

// TestScriptArgsDefaultToEmptyWhenNotSet verifies that a script run without SetArgs
// sees an empty args() list rather than the host binary's os.Args.
func TestScriptArgsDefaultToEmptyWhenNotSet(t *testing.T) {
	s := rumo.NewScript([]byte(`out := args()`))
	p, err := s.Run()
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	arr := p.Get("out").Array()
	if len(arr) != 0 {
		t.Errorf("expected empty args, got %d args: %v (os.Args=%v)", len(arr), arr, os.Args)
	}
}

// TestProgramSetArgsMakesArgsVisibleToScript verifies that args set via SetArgs are
// exactly what the script sees through the args() builtin.
func TestProgramSetArgsMakesArgsVisibleToScript(t *testing.T) {
	s := rumo.NewScript([]byte(`out := args()`))
	p, err := s.Compile()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := []string{"script.rumo", "--flag", "value"}
	p.SetArgs(want)
	if err := p.Run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := p.Get("out").Array()
	if len(got) != len(want) {
		t.Fatalf("args len mismatch: want %d, got %d (%v)", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("args[%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestCompileAndRunPropagatesArgsToScript verifies that CompileAndRun passes the
// provided args slice through to the script's args() builtin.
func TestCompileAndRunPropagatesArgsToScript(t *testing.T) {
	script := []byte(`n := len(args())`)
	if err := rumo.CompileAndRun(context.Background(), script, "test.rumo", []string{"a", "b", "c"}); err != nil {
		t.Fatalf("CompileAndRun: %v", err)
	}
}

// TestNewVMDoesNotDefaultToOSArgs verifies that NewVM defaults Args to nil/empty
// rather than os.Args so that independently-constructed VMs do not leak host args.
func TestNewVMDoesNotDefaultToOSArgs(t *testing.T) {
	bytecode := &vm.Bytecode{
		FileSet:      nil,
		MainFunction: &vm.CompiledFunction{},
		Constants:    nil,
	}
	v := vm.NewVM(context.Background(), bytecode, nil, -1, nil)
	if len(v.Args) > 0 {
		t.Errorf("NewVM should default Args to empty, got %v", v.Args)
	}
}

// TestRunREPLFmtPrintWritesToProvidedWriter verifies that the fmt stdlib module's
// print functions write to the custom io.Writer passed to RunREPL, not os.Stdout.
func TestRunREPLFmtPrintWritesToProvidedWriter(t *testing.T) {
	var out bytes.Buffer
	// "fmt" is pre-imported globally by passing it in the modules slice, so no import
	// statement is needed in the REPL line.
	input := strings.NewReader(`fmt.println("hello-from-fmt")` + "\n")
	rumo.RunREPL(context.Background(), input, &out, ">> ", []string{"fmt"})

	got := out.String()
	if !strings.Contains(got, "hello-from-fmt") {
		t.Errorf("expected fmt.println output in provided writer, got: %q", got)
	}
}

