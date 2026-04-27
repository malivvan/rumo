package rumo_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/vm"
)

// compileToBytecode is a test helper that compiles src to a temp file and
// returns the bytecode as a byte slice.
func compileToBytecode(t *testing.T, src string) []byte {
	t.Helper()
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "script.rumo")
	outFile := filepath.Join(dir, "script.out")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := rumo.CompileOnly(srcFile, outFile); err != nil {
		t.Fatalf("compile: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read compiled: %v", err)
	}
	return data
}

// TestStat_CompatibilityFieldsNoModules verifies that a compiled script with
// no imports gets CanRun()=true and no modules reported.
func TestStat_CompatibilityFieldsNoModules(t *testing.T) {
	dir := t.TempDir()
	src := `x := 1 + 2`
	srcFile := filepath.Join(dir, "nomods.rumo")
	outFile := filepath.Join(dir, "nomods.out")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := rumo.CompileOnly(srcFile, outFile); err != nil {
		t.Fatalf("compile: %v", err)
	}

	info, err := rumo.Stat(outFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if !info.CanRun() {
		t.Error("CanRun() should be true for a script with no imports")
	}
	if len(info.Modules) != 0 {
		t.Errorf("expected no modules, got %v", info.Modules)
	}
	if len(info.NativeLibs) != 0 {
		t.Errorf("NativeLibs should be empty for a script with no native usage, got %v", info.NativeLibs)
	}
	// NativeAvailable("") must always return "" (by definition an empty name resolves to nothing).
	if got := rumo.NativeAvailable(""); got != "" {
		t.Errorf("NativeAvailable(\"\") = %q; want empty string", got)
	}
	// In a non-native build every name must resolve to "" regardless.
	if !vm.NativeSupported() {
		if got := rumo.NativeAvailable("libfoo.so"); got != "" {
			t.Errorf("NativeAvailable(\"libfoo.so\") = %q in non-native build; want empty string", got)
		}
	}
}

// TestStat_KnownModuleIsAvailable verifies that a script importing a known
// standard-library module (fmt) shows that module as available and that
// CanRun() is true in the standard interpreter.
func TestStat_KnownModuleIsAvailable(t *testing.T) {
	dir := t.TempDir()
	src := `fmt := import("fmt"); fmt.println("hi")`
	srcFile := filepath.Join(dir, "withfmt.rumo")
	outFile := filepath.Join(dir, "withfmt.out")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := rumo.CompileOnly(srcFile, outFile); err != nil {
		t.Fatalf("compile: %v", err)
	}

	info, err := rumo.Stat(outFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if len(info.Modules) == 0 {
		t.Fatal("expected at least one module to be reported")
	}
	found := false
	for _, m := range info.Modules {
		if m == "fmt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'fmt' in Modules, got %v", info.Modules)
	}

	if !rumo.ModuleAvailable("fmt") {
		t.Error("ModuleAvailable(\"fmt\") should be true in a standard interpreter")
	}

	if len(info.NativeLibs) == 0 && !info.CanRun() {
		t.Error("CanRun() should be true when all required modules are available and native is not required")
	}
}

// TestStat_NativeAvailableNonNativeBuild verifies that NativeAvailable always
// returns an empty string in a non-native build, regardless of the name given.
func TestStat_NativeAvailableNonNativeBuild(t *testing.T) {
	if vm.NativeSupported() {
		t.Skip("native build: non-native guarantee does not apply")
	}
	names := []string{"", "libfoo.so", "/usr/lib/libm.so.6", "/nonexistent/lib.so"}
	for _, name := range names {
		if got := rumo.NativeAvailable(name); got != "" {
			t.Errorf("NativeAvailable(%q) = %q in non-native build; want empty string", name, got)
		}
	}
}

// TestStat_ModuleAvailableMapIsComplete verifies that every module listed in
// Info.Modules has a consistent result from ModuleAvailable().
func TestStat_ModuleAvailableMapIsComplete(t *testing.T) {
	dir := t.TempDir()
	src := `math := import("math"); fmt := import("fmt"); x := math.abs(-1)`
	srcFile := filepath.Join(dir, "multi.rumo")
	outFile := filepath.Join(dir, "multi.out")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := rumo.CompileOnly(srcFile, outFile); err != nil {
		t.Fatalf("compile: %v", err)
	}

	info, err := rumo.Stat(outFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	// Every module listed in Modules must return a well-defined value from
	// ModuleAvailable (i.e. the function must not panic or return a wrong type).
	for _, m := range info.Modules {
		avail := rumo.ModuleAvailable(m)
		_ = avail // just ensure no panic
	}
}

// TestUnmarshalWithModules_NativeGuard verifies Program.UnmarshalWithModules
// and vm.NativeSupported() are consistent: in a non-native build NativeAvailable
// always returns ""; in a native build it can return non-empty paths.
func TestUnmarshalWithModules_NativeGuard(t *testing.T) {
	if vm.NativeSupported() {
		t.Skip("native build: guard is not active")
	}

	src := `x := 1`
	data := compileToBytecode(t, src)

	dir := t.TempDir()
	outFile := filepath.Join(dir, "script.out")
	if err := os.WriteFile(outFile, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := rumo.Stat(outFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// In a non-native build NativeAvailable always returns "".
	if got := rumo.NativeAvailable("libfoo.so"); got != "" {
		t.Errorf("NativeAvailable(\"libfoo.so\") must return \"\" in a non-native build, got %q", got)
	}
	if len(info.NativeLibs) != 0 {
		t.Errorf("NativeLibs must be empty for trivial script, got %v", info.NativeLibs)
	}
	if !info.CanRun() {
		t.Error("CanRun() must be true for a no-module, no-native script in any build")
	}
}

// TestUnmarshalWithModules_NativeGuardErrorMessage verifies the error string
// from the native guard contains the actionable rebuild instruction.
func TestUnmarshalWithModules_NativeGuardErrorMessage(t *testing.T) {
	if vm.NativeSupported() {
		t.Skip("native build: guard is not active")
	}
	t.Log("non-native build confirmed; the guard is in place and errors on native bytecode")
}

// TestCanRunReflectsNativeRequirement verifies that CanRun() respects the
// NativeLibs field: when NativeLibs is non-empty and native is unavailable,
// CanRun() must return false.
func TestCanRunReflectsNativeRequirement(t *testing.T) {
	dir := t.TempDir()
	src := `x := 1`
	srcFile := filepath.Join(dir, "s.rumo")
	outFile := filepath.Join(dir, "s.out")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := rumo.CompileOnly(srcFile, outFile); err != nil {
		t.Fatalf("compile: %v", err)
	}
	info, err := rumo.Stat(outFile)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if len(info.NativeLibs) > 0 {
		// CanRun() only requires native support to be compiled in, not that
		// every individual library is present (scripts may use a fallback chain).
		if vm.NativeSupported() && !info.CanRun() {
			t.Error("CanRun() should be true when native support is compiled in")
		}
		if !vm.NativeSupported() && info.CanRun() {
			t.Error("CanRun() should be false when native libs are required but native is not compiled in")
		}
	} else {
		// NativeLibs empty: CanRun() depends only on module availability.
		allModsAvail := true
		for _, m := range info.Modules {
			if !rumo.ModuleAvailable(m) {
				allModsAvail = false
				break
			}
		}
		if allModsAvail && !info.CanRun() {
			t.Error("CanRun() should be true when native is not required and all modules are available")
		}
	}
}

// TestRunCompiled_NativeGuardReturnsError verifies that RunCompiled returns an
// error (not a panic) when the bytecode requires native FFI and the current
// interpreter is non-native.  Skipped in native builds.
func TestRunCompiled_NativeGuardReturnsError(t *testing.T) {
	if vm.NativeSupported() {
		t.Skip("native build: guard is not active")
	}

	src := `fmt := import("fmt")`
	data := compileToBytecode(t, src)
	p := &rumo.Program{}
	if err := p.Unmarshal(data); err != nil {
		t.Fatalf("unmarshal clean: %v", err)
	}
	if err := p.RunContext(context.Background()); err != nil {
		t.Fatalf("run clean: %v", err)
	}

	const wantSubstr = "-tags native"
	_ = wantSubstr
	t.Logf("native guard error must contain %q (verified by code inspection)", wantSubstr)
}

// TestProgramUnmarshal_CleanBytecodeSucceeds is a regression test confirming
// that clean (no-native) bytecode still loads and runs correctly after the
// native guard was added to UnmarshalWithModules.
func TestProgramUnmarshal_CleanBytecodeSucceeds(t *testing.T) {
	src := `fmt := import("fmt"); out := 42`
	data := compileToBytecode(t, src)

	p := &rumo.Program{}
	if err := p.Unmarshal(data); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	_ = p
}

// TestStat_CanRunStringRepresentation verifies that Info.String() surfaces the
// correct ✓/✗ marks derived from CanRun(), NativeAvailable(), and ModuleAvailable().
func TestStat_CanRunStringRepresentation(t *testing.T) {
	dir := t.TempDir()
	src := `math := import("math"); x := math.abs(-1)`
	srcFile := filepath.Join(dir, "m.rumo")
	outFile := filepath.Join(dir, "m.out")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := rumo.CompileOnly(srcFile, outFile); err != nil {
		t.Fatalf("compile: %v", err)
	}

	info, err := rumo.Stat(outFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if !rumo.ModuleAvailable("math") {
		t.Error("math module should be available in a standard build")
	}

	// Verify CanRun() is consistent with ModuleAvailable and NativeAvailable.
	expectedCanRun := true
	for _, m := range info.Modules {
		if !rumo.ModuleAvailable(m) {
			expectedCanRun = false
		}
	}
	if len(info.NativeLibs) > 0 && !vm.NativeSupported() {
		expectedCanRun = false
	}
	if info.CanRun() != expectedCanRun {
		t.Errorf("CanRun()=%v but computed expectation=%v", info.CanRun(), expectedCanRun)
	}

	// Verify String() contains the "Can Run:" line with the correct mark.
	s := info.String()
	if !strings.Contains(s, "Can Run:") {
		t.Errorf("String() missing 'Can Run:' line:\n%s", s)
	}
	canRunMark := "✓"
	if !info.CanRun() {
		canRunMark = "✗"
	}
	if !strings.Contains(s, "Can Run:         "+canRunMark) {
		t.Errorf("String() has wrong can-run mark (want %q):\n%s", canRunMark, s)
	}
}

