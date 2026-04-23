package vm_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/parser"
)

// -------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------

// compileAndRunNative parses src, compiles + runs it and returns the value
// bound to the global variable `out` (or an error).  Native statements
// require the real VM, not the lightweight helpers used by most other
// tests, because the loader dlopens a shared library at runtime.
func compileAndRunNative(t *testing.T, src string) (vm.Object, error) {
	t.Helper()

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("test", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	st := vm.NewSymbolTable()
	for idx, fn := range vm.GetAllBuiltinFunctions() {
		st.DefineBuiltin(idx, fn.Name)
	}
	outSym := st.Define("out")
	globals := make([]vm.Object, vm.DefaultConfig.GlobalsSize)
	globals[outSym.Index] = vm.UndefinedValue

	c := vm.NewCompiler(srcFile, st, nil, nil, nil)
	if err := c.Compile(file); err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}

	bc := c.Bytecode()

	machine := vm.NewVM(context.Background(), bc, globals, nil)
	if err := machine.Run(); err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	return globals[outSym.Index], nil
}

// buildTestNativeLib compiles a small C source file into a shared library
// the test can `dlopen`.  It mirrors the helper used by the purego tests.
func buildTestNativeLib(t *testing.T, csrc string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("native test compiles a POSIX shared object; skipping on Windows")
	}

	ccBytes, err := exec.Command("go", "env", "CC").Output()
	if err != nil {
		t.Skipf("go env CC failed: %v", err)
	}
	cc := strings.TrimSpace(string(ccBytes))
	if cc == "" {
		t.Skip("no C compiler configured (go env CC is empty)")
	}
	if _, err := exec.LookPath(cc); err != nil {
		t.Skipf("C compiler %q not found in PATH: %v", cc, err)
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "rumotest.c")
	libPath := filepath.Join(dir, "librumotest.so")
	if runtime.GOOS == "darwin" {
		libPath = filepath.Join(dir, "librumotest.dylib")
	}
	if err := os.WriteFile(srcPath, []byte(csrc), 0644); err != nil {
		t.Fatalf("write c source: %v", err)
	}

	args := []string{"-shared", "-fPIC", "-O2", "-o", libPath, srcPath}
	if runtime.GOOS == "darwin" {
		args = append(args, "-Wl,-install_name,"+libPath)
	}
	cmd := exec.Command(cc, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("unable to build shared lib: %v\n%s", err, string(out))
	}
	return libPath
}

// -------------------------------------------------------------------------
// Parser / compiler sanity
// -------------------------------------------------------------------------

// TestNative_ParseRoundTrip ensures the native statement parses, preserves
// its declared symbols and prints back in something close to the original
// form.
func TestNative_ParseRoundTrip(t *testing.T) {
	src := `
native libm = "libm.so.6" {
    sqrt: func(float) float
    pow: func(float, float) float
    atoi(string) int
    free(ptr)
    exit(int) void
}
`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("roundtrip", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var stmt *parser.NativeStmt
	for _, s := range file.Stmts {
		if ns, ok := s.(*parser.NativeStmt); ok {
			stmt = ns
			break
		}
	}
	if stmt == nil {
		t.Fatalf("no NativeStmt in parsed output")
	}
	if stmt.Name.Name != "libm" {
		t.Fatalf("expected binding name libm, got %s", stmt.Name.Name)
	}
	if stmt.Path != "libm.so.6" {
		t.Fatalf("expected path libm.so.6, got %s", stmt.Path)
	}
	wantFuncs := []string{"sqrt", "pow", "atoi", "free", "exit"}
	if len(stmt.Funcs) != len(wantFuncs) {
		t.Fatalf("expected %d funcs, got %d", len(wantFuncs), len(stmt.Funcs))
	}
	for i, want := range wantFuncs {
		if stmt.Funcs[i].Name.Name != want {
			t.Fatalf("func[%d]: want %s got %s", i, want, stmt.Funcs[i].Name.Name)
		}
	}
}

// TestNative_UnknownType catches invalid type keywords at compile time.
func TestNative_UnknownType(t *testing.T) {
	src := `
native bad = "x" {
    f(qux) int
}
`
	if _, err := compileAndRunNative(t, src); err == nil {
		t.Fatalf("expected compile error for unknown type")
	} else if !strings.Contains(err.Error(), "unknown parameter type") {
		t.Fatalf("expected unknown parameter type error, got: %v", err)
	}
}

// TestNative_DuplicateBinding rejects two functions with the same name.
func TestNative_DuplicateBinding(t *testing.T) {
	src := `
native bad = "x" {
    f(int) int
    f(int) int
}
`
	if _, err := compileAndRunNative(t, src); err == nil {
		t.Fatalf("expected compile error for duplicate binding")
	} else if !strings.Contains(err.Error(), "duplicate function binding") {
		t.Fatalf("expected duplicate binding error, got: %v", err)
	}
}

// TestNative_MissingLibrary verifies that loading fails gracefully at
// runtime when the shared object cannot be located.
func TestNative_MissingLibrary(t *testing.T) {
	src := `
native bad = "/definitely/not/a/real/library.so" {
    something(int) int
}
out = bad
`
	if _, err := compileAndRunNative(t, src); err == nil {
		t.Fatalf("expected runtime error for missing library")
	} else if !strings.Contains(err.Error(), "failed to open") {
		t.Fatalf("expected 'failed to open' error, got: %v", err)
	}
}

// -------------------------------------------------------------------------
// End-to-end: build a custom .so and invoke it through the native DSL.
// -------------------------------------------------------------------------

const testNativeCSource = `
#include <stdint.h>
#include <stddef.h>
#include <string.h>

int64_t rumo_add(int64_t a, int64_t b) { return a + b; }

double  rumo_scale(double x, double factor) { return x * factor; }

int rumo_truthy(int x) { return x != 0; }

int rumo_strlen(const char *s) {
    if (s == NULL) return -1;
    return (int)strlen(s);
}

static int64_t g_counter = 0;
int64_t rumo_bump(void) { return ++g_counter; }

void rumo_reset(void) { g_counter = 0; }

int64_t rumo_sum_bytes(const unsigned char *buf, int64_t n) {
    if (buf == NULL) return 0;
    int64_t sum = 0;
    for (int64_t i = 0; i < n; i++) sum += buf[i];
    return sum;
}
`

func TestNative_EndToEnd(t *testing.T) {
	libPath := buildTestNativeLib(t, testNativeCSource)

	src := fmt.Sprintf(`
native lib = %q {
    rumo_add(int, int) int
    rumo_scale(float, float) float
    rumo_truthy(int) bool
    rumo_strlen(string) int
    rumo_bump() int
    rumo_reset() void
    rumo_sum_bytes(bytes, int) int
}

result := {}
result.add       = lib.rumo_add(40, 2)
result.scale     = lib.rumo_scale(3.0, 2.5)
result.truthy_f  = lib.rumo_truthy(0)
result.truthy_t  = lib.rumo_truthy(7)
result.strlen    = lib.rumo_strlen("hello")
lib.rumo_reset()
result.bump1     = lib.rumo_bump()
result.bump2     = lib.rumo_bump()
result.sum_bytes = lib.rumo_sum_bytes(bytes("ABC"), 3)
result.path      = lib.__path__
result.close     = lib.close
out = result
`, libPath)

	got, err := compileAndRunNative(t, src)
	if err != nil {
		t.Fatalf("native program failed: %v", err)
	}
	m, ok := got.(*vm.Map)
	if !ok {
		t.Fatalf("expected *vm.Map result, got %T: %s", got, got.String())
	}

	checkInt := func(key string, want int64) {
		t.Helper()
		v, ok := m.Value[key].(*vm.Int)
		if !ok {
			t.Fatalf("result[%q]: want Int, got %T (%v)", key, m.Value[key], m.Value[key])
		}
		if v.Value != want {
			t.Fatalf("result[%q]: want %d got %d", key, want, v.Value)
		}
	}
	checkFloat := func(key string, want float64) {
		t.Helper()
		v, ok := m.Value[key].(*vm.Float)
		if !ok {
			t.Fatalf("result[%q]: want Float, got %T (%v)", key, m.Value[key], m.Value[key])
		}
		if v.Value != want {
			t.Fatalf("result[%q]: want %v got %v", key, want, v.Value)
		}
	}

	checkInt("add", 42)
	checkFloat("scale", 7.5)
	if m.Value["truthy_f"] != vm.FalseValue {
		t.Fatalf("truthy_f: expected FalseValue got %s", m.Value["truthy_f"].String())
	}
	if m.Value["truthy_t"] != vm.TrueValue {
		t.Fatalf("truthy_t: expected TrueValue got %s", m.Value["truthy_t"].String())
	}
	checkInt("strlen", 5)
	checkInt("bump1", 1)
	checkInt("bump2", 2)
	checkInt("sum_bytes", int64('A')+int64('B')+int64('C'))

	path, _ := m.Value["path"].(*vm.String)
	if path == nil || path.Value != libPath {
		t.Fatalf("path mismatch: got %v", m.Value["path"])
	}

	// `close` should unload the library without error and be safe to call
	// a second time.
	closer, ok := m.Value["close"].(*vm.BuiltinFunction)
	if !ok {
		t.Fatalf("close not found on native map")
	}
	if _, err := closer.Call(context.Background()); err != nil {
		t.Fatalf("close() failed: %v", err)
	}
	if _, err := closer.Call(context.Background()); err != nil {
		t.Fatalf("double close() failed: %v", err)
	}
}

// TestNative_ArgTypeMismatch makes sure that passing a string where an int
// is expected surfaces as a runtime error rather than panicking.
func TestNative_ArgTypeMismatch(t *testing.T) {
	libPath := buildTestNativeLib(t, testNativeCSource)

	src := fmt.Sprintf(`
native lib = %q {
    rumo_add(int, int) int
}
out = lib.rumo_add("nope", 2)
`, libPath)

	_, err := compileAndRunNative(t, src)
	if err == nil {
		t.Fatalf("expected argument mismatch error")
	}
	if !strings.Contains(err.Error(), "argument 0") && !strings.Contains(err.Error(), "rumo_add") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestNative_NoFuncSpecs guards the degenerate case of an empty bindings
// block - it should still produce a usable (if trivially empty) map.
func TestNative_NoFuncSpecs(t *testing.T) {
	libPath := buildTestNativeLib(t, testNativeCSource)

	src := fmt.Sprintf(`
native lib = %q { }
out = lib.__path__
`, libPath)

	got, err := compileAndRunNative(t, src)
	if err != nil {
		t.Fatalf("native program failed: %v", err)
	}
	s, ok := got.(*vm.String)
	if !ok {
		t.Fatalf("expected string, got %T", got)
	}
	if s.Value != libPath {
		t.Fatalf("want %s got %s", libPath, s.Value)
	}
}

// TestNative_CompilerErrorTypes ensures `void` is rejected as a parameter.
func TestNative_VoidParam(t *testing.T) {
	src := `
native bad = "x" {
    f(void) int
}
`
	if _, err := compileAndRunNative(t, src); err == nil {
		t.Fatalf("expected error for void parameter")
	} else if !errors.Is(err, err) && !strings.Contains(err.Error(), "'void' is not allowed") {
		t.Fatalf("expected void-not-allowed error, got: %v", err)
	}
}
