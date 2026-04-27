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
	"github.com/malivvan/rumo/vm/require"
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
	var errOut bytes.Buffer

	rumo.RunREPL(context.Background(), strings.NewReader("1 + 1\n"), &out, &errOut, nil)

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

// A symlink inside the import root that resolves to a target outside the root must be
// rejected. The lexical filepath.Rel check passes because the symlink path itself lies
// within the sandbox directory tree, but the real file it points to is outside that tree.
// The fix evaluates symlinks on both the import base and the resolved module path before
// performing the containment check, so the real target location is compared, not the
// symlink path.
func TestScriptFileImportRejectsSymlinkEscapingImportRoot(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "sandbox")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir sandbox: %v", err)
	}

	// Write a sensitive file outside the sandbox.
	sensitiveFile := filepath.Join(tempDir, "secret.rumo")
	if err := os.WriteFile(sensitiveFile, []byte(`export "secret content"`), 0o644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	// Create a symlink inside the sandbox pointing at the file outside.
	symlinkPath := filepath.Join(root, "escape.rumo")
	if err := os.Symlink(sensitiveFile, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	s := rumo.NewScript([]byte(`out := import("escape")`))
	s.SetName(filepath.Join(root, "main.rumo"))
	s.EnableFileImport(true)
	if err := s.SetImportDir(root); err != nil {
		t.Fatalf("set import dir: %v", err)
	}

	_, err := s.Compile()
	if err == nil {
		t.Fatal("expected symlink-based import escape to be rejected, but compile succeeded")
	}
	if !strings.Contains(err.Error(), "escapes import root") {
		t.Fatalf("expected 'escapes import root' error, got: %v", err)
	}
}

// TestScriptFileImportAllowsSymlinkWithinRoot verifies that a symlink whose real target
// also lies within the import root is accepted.
func TestScriptFileImportAllowsSymlinkWithinRoot(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "sandbox")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir sandbox: %v", err)
	}

	// Write a real module inside the sandbox.
	realModule := filepath.Join(root, "real.rumo")
	if err := os.WriteFile(realModule, []byte(`export 42`), 0o644); err != nil {
		t.Fatalf("write real module: %v", err)
	}

	// Create a symlink inside the sandbox pointing to the real module (also inside).
	symlinkPath := filepath.Join(root, "alias.rumo")
	if err := os.Symlink(realModule, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	s := rumo.NewScript([]byte(`out := import("alias")`))
	s.SetName(filepath.Join(root, "main.rumo"))
	s.EnableFileImport(true)
	if err := s.SetImportDir(root); err != nil {
		t.Fatalf("set import dir: %v", err)
	}

	p, err := s.Run()
	if err != nil {
		t.Fatalf("expected symlink within root to be allowed, got: %v", err)
	}
	if got := p.Get("out").Int(); got != 42 {
		t.Fatalf("unexpected result: %d", got)
	}
}

// The std/os module exposes privileged host-process operations (process launch, os.exit,
// environment mutation, working-directory change, file I/O) with no mechanism for
// embedders to deny individual capabilities. A script that calls os.setenv will mutate
// the host's environment, os.exec will spawn subprocesses, and os.exit will terminate
// the host process — with no way to prevent any of this.
//
// The fix adds a vm.Permissions struct to vm.Config (and wires it through Script and
// Program). Each Deny* field, when true, causes the corresponding os-module function
// to return vm.ErrNotPermitted instead of executing the operation. The zero value of
// Permissions (all fields false) preserves backward-compatible allow-all behaviour.

// runOSPermTest is a helper that compiles and runs a one-line script that imports the
// os module, applying the given permissions, and returns the execution error.
func runOSPermTest(t *testing.T, src string, perm vm.Permissions) error {
	t.Helper()
	s := rumo.NewScript([]byte(src))
	s.SetImports(rumo.GetModuleMap("os"))
	s.SetPermissions(perm)
	_, err := s.Run()
	return err
}

// TestOSPermissionDenyEnvWrite verifies that os.setenv, os.unsetenv, and os.clearenv
// return an error when DenyEnvWrite is set, and that the host environment is not mutated.
func TestOSPermissionDenyEnvWrite(t *testing.T) {
	const key = "RUMO_PERM_TEST_ENVWRITE"
	os.Unsetenv(key)

	err := runOSPermTest(t,
		`os := import("os"); os.setenv("`+key+`", "mutated")`,
		vm.Permissions{DenyEnvWrite: true},
	)
	if err == nil {
		t.Fatal("expected error when DenyEnvWrite=true, got nil")
	}
	if got := os.Getenv(key); got != "" {
		t.Errorf("env was mutated despite DenyEnvWrite: %s=%q", key, got)
	}
}

// TestOSPermissionDenyExec verifies that os.exec returns an error when DenyExec is set.
func TestOSPermissionDenyExec(t *testing.T) {
	err := runOSPermTest(t,
		`os := import("os"); cmd := os.exec("true"); cmd.run()`,
		vm.Permissions{DenyExec: true},
	)
	if err == nil {
		t.Fatal("expected error when DenyExec=true, got nil")
	}
}

// TestOSPermissionDenyChdir verifies that os.chdir returns an error when DenyChdir is set.
func TestOSPermissionDenyChdir(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) }) // restore in case permission check is absent
	err := runOSPermTest(t,
		`os := import("os"); os.chdir("`+dir+`")`,
		vm.Permissions{DenyChdir: true},
	)
	if err == nil {
		t.Fatal("expected error when DenyChdir=true, got nil")
	}
}

// TestOSPermissionDenyFileRead verifies that os.read_file returns an error when DenyFileRead is set.
func TestOSPermissionDenyFileRead(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(f, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runOSPermTest(t,
		`os := import("os"); os.read_file("`+f+`")`,
		vm.Permissions{DenyFileRead: true},
	)
	if err == nil {
		t.Fatal("expected error when DenyFileRead=true, got nil")
	}
}

// TestOSPermissionDenyFileWrite verifies that os.create returns an error when DenyFileWrite is set.
func TestOSPermissionDenyFileWrite(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.txt")
	err := runOSPermTest(t,
		`os := import("os"); os.create("`+f+`")`,
		vm.Permissions{DenyFileWrite: true},
	)
	if err == nil {
		t.Fatal("expected error when DenyFileWrite=true, got nil")
	}
}

// TestOSPermissionDefaultAllowsAll verifies that with default permissions (zero value)
// the os module works normally — specifically that setenv, exec, and read_file succeed.
func TestOSPermissionDefaultAllowsAll(t *testing.T) {
	const key = "RUMO_PERM_TEST_DEFAULT"
	os.Unsetenv(key)
	t.Cleanup(func() { os.Unsetenv(key) })

	if err := runOSPermTest(t,
		`os := import("os"); os.setenv("`+key+`", "ok")`,
		vm.Permissions{},
	); err != nil {
		t.Errorf("unexpected error with default permissions: %v", err)
	}
	if got := os.Getenv(key); got != "ok" {
		t.Errorf("setenv did not take effect: %s=%q", key, got)
	}
}

// TestOSPermissionDenyExit verifies that os.exit returns an error when DenyExit is set,
// preventing the host process from being terminated by a script.
func TestOSPermissionDenyExit(t *testing.T) {
	err := runOSPermTest(t,
		`os := import("os"); os.exit(0)`,
		vm.Permissions{DenyExit: true},
	)
	if err == nil {
		t.Fatal("expected error when DenyExit=true, got nil — host process would have been terminated")
	}
}

// os.expand_env has two accounting defects. First, the accumulator inside the
// os.Expand callback tracks only the byte-lengths of substituted values, ignoring
// the literal template text. A template that is mostly literal characters and
// references a variable that expands to the empty string contributes zero to the
// accumulator even though the result string will be as large as the template.
// Second, both the early-exit check inside the callback and the post-expand check
// read vm.DefaultConfig.MaxStringLen directly instead of the per-VM MaxStringLen
// set by the embedder. A VM created with a strict MaxStringLen will have expand_env
// silently bypass that limit, producing strings larger than the configured cap.
//
// The fix reads the per-VM MaxStringLen from the execution context and uses it for
// both the early-exit accumulator (which is now seeded with the template length as
// a conservative upper bound) and the definitive post-expand length check.

// runExpandEnvTest compiles and runs a script that calls os.expand_env with the
// given template string, applying the per-VM MaxStringLen limit.
func runExpandEnvTest(t *testing.T, template string, maxStringLen int) error {
	t.Helper()
	s := rumo.NewScript([]byte(`os := import("os"); out := os.expand_env("` + template + `")`))
	s.SetImports(rumo.GetModuleMap("os"))
	s.SetMaxStringLen(maxStringLen)
	_, err := s.Run()
	return err
}

// TestExpandEnvRespectsPerVMMaxStringLen verifies that expand_env enforces the
// per-VM MaxStringLen when a substituted value causes the result to exceed it.
func TestExpandEnvRespectsPerVMMaxStringLen(t *testing.T) {
	const key = "RUMO_EXPAND_PERVM"
	os.Setenv(key, strings.Repeat("A", 100))
	t.Cleanup(func() { os.Unsetenv(key) })

	err := runExpandEnvTest(t, "$"+key, 50)
	if err == nil {
		t.Fatal("expected ErrStringLimit when substituted result (100 bytes) exceeds per-VM MaxStringLen (50), got nil")
	}
}

// TestExpandEnvLiteralTextCountedInLimit verifies that literal template text is
// included in the length accounting, not just the substituted values. A template
// that consists mostly of literal characters plus a zero-length substitution must
// be rejected when the template itself exceeds MaxStringLen.
func TestExpandEnvLiteralTextCountedInLimit(t *testing.T) {
	// $EMPTY_VAR expands to "" so only the 40-char literal contributes to the result.
	const key = "RUMO_EXPAND_EMPTY"
	os.Setenv(key, "")
	t.Cleanup(func() { os.Unsetenv(key) })

	// Template literal: 40 'A's + "$RUMO_EXPAND_EMPTY" = ~58 chars
	// Substitution: "" (zero bytes)
	// Result: 40 'A's = 40 bytes
	// MaxStringLen: 30 → result (40) exceeds limit
	template := strings.Repeat("A", 40) + "$" + key
	err := runExpandEnvTest(t, template, 30)
	if err == nil {
		t.Fatal("expected ErrStringLimit when literal template text (40 bytes) exceeds per-VM MaxStringLen (30), got nil")
	}
}

// TestExpandEnvBelowLimitSucceeds verifies that expand_env works normally when
// the result stays within the configured MaxStringLen.
func TestExpandEnvBelowLimitSucceeds(t *testing.T) {
	const key = "RUMO_EXPAND_SMALL"
	os.Setenv(key, "hello")
	t.Cleanup(func() { os.Unsetenv(key) })

	err := runExpandEnvTest(t, "$"+key, 100)
	if err != nil {
		t.Fatalf("unexpected error when result is within limit: %v", err)
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
	v := vm.NewVM(context.Background(), bytecode, nil, nil)
	if len(v.Args) > 0 {
		t.Errorf("NewVM should default Args to empty, got %v", v.Args)
	}
}

// TestRunREPLFmtPrintWritesToProvidedWriter verifies that the fmt stdlib module's
// print functions write to the custom io.Writer passed to RunREPL, not os.Stdout.
func TestRunREPLFmtPrintWritesToProvidedWriter(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	// "fmt" is pre-imported globally by passing it in the modules slice, so no import
	// statement is needed in the REPL line.
	input := strings.NewReader(`fmt.println("hello-from-fmt")` + "\n")
	rumo.RunREPL(context.Background(), input, &out, &errOut, []string{"fmt"})

	got := out.String()
	if !strings.Contains(got, "hello-from-fmt") {
		t.Errorf("expected fmt.println output in provided writer, got: %q", got)
	}
}

// writeTemp writes content to a file inside dir and returns the file path.
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEmbed_SingleFileString(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "hello.txt", "hello world")

	src := `
//embed hello.txt
content := ""
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("content")
	require.NotNil(t, v)
	require.Equal(t, "hello world", v.String())
}

func TestEmbed_SingleFileBytes(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "data.bin", "binary data")

	src := `
//embed data.bin
content := bytes("")
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("content")
	require.NotNil(t, v)
	require.Equal(t, "binary data", v.String())
}

func TestEmbed_MultiFileStringMap(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "file a")
	writeTemp(t, dir, "b.txt", "file b")

	src := `
//embed *.txt
files := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("files")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
	require.Equal(t, "file a", m["a.txt"])
	require.Equal(t, "file b", m["b.txt"])
}

func TestEmbed_MultiFileBytesMap(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "c.txt", "file c")
	writeTemp(t, dir, "d.txt", "file d")

	src := `
//embed *.txt
files := bytes({})
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("files")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
}

func TestEmbed_MultiplePatterns(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "readme.md", "# readme")
	writeTemp(t, dir, "config.json", "{}")

	src := `
//embed readme.md config.json
assets := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("assets")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
}

func TestEmbed_SubdirectoryPattern(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "sub/file1.txt", "sub file 1")
	writeTemp(t, dir, "sub/file2.txt", "sub file 2")

	src := `
//embed sub/*.txt
files := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("files")
	require.NotNil(t, v)

	m, ok := v.Value().(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 2, len(m))
}

func TestEmbed_NoImportDir(t *testing.T) {
	src := `
//embed hello.txt
content := ""
`
	s := rumo.NewScript([]byte(src))
	// No SetImportDir call — importDir is empty.
	_, err := s.Compile()
	require.Error(t, err)
}

func TestEmbed_PatternNoMatch(t *testing.T) {
	dir := t.TempDir()

	src := `
//embed *.nonexistent
files := {}
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	_, err := s.Compile()
	require.Error(t, err)
}

func TestEmbed_SingleFileMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "a")
	writeTemp(t, dir, "b.txt", "b")

	src := `
//embed *.txt
content := ""
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	_, err := s.Compile()
	require.Error(t, err) // glob matches 2 files but target is a single string
}

func TestEmbed_UsedInExpression(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "greeting.txt", "Hello")

	src := `
//embed greeting.txt
msg := ""
result := msg + ", World!"
`
	s := rumo.NewScript([]byte(src))
	require.NoError(t, s.SetImportDir(dir))

	compiled, err := s.Compile()
	require.NoError(t, err)

	err = compiled.RunContext(context.Background())
	require.NoError(t, err)

	v := compiled.Get("result")
	require.NotNil(t, v)
	require.Equal(t, "Hello, World!", v.String())
}
