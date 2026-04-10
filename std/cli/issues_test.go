package cli

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Issue #15: cli: Hard dependency on os.Args
// ExecuteC() falls back to os.Args[1:]. In WASI/JS, os.Args may be empty
// or synthetic. The filepath.Base(os.Args[0]) guard assumes native path
// semantics.
// ---------------------------------------------------------------------------

func TestIssue15_ExecuteCWithEmptyOsArgs(t *testing.T) {
	origOsArgs := OsArgs
	defer func() { OsArgs = origOsArgs }()

	// Simulate empty os.Args (as in WASI/JS)
	OsArgs = func() []string { return []string{} }

	ran := false
	cmd := &Command{
		Use: "test",
		Run: func(cmd *Command, args []string) {
			ran = true
		},
	}
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("ExecuteC should handle empty os.Args gracefully: %v", err)
	}
	if !ran {
		t.Fatal("Run should have been invoked")
	}
}

func TestIssue15_ExecuteCWithSingleOsArg(t *testing.T) {
	origOsArgs := OsArgs
	defer func() { OsArgs = origOsArgs }()

	// Simulate synthetic os.Args with just the program name
	OsArgs = func() []string { return []string{"/wasi/program"} }

	ran := false
	cmd := &Command{
		Use: "test",
		Run: func(cmd *Command, args []string) {
			ran = true
		},
	}
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatalf("ExecuteC should handle single os.Args: %v", err)
	}
	if !ran {
		t.Fatal("Run should have been invoked")
	}
}

func TestIssue15_SetArgsOverridesOsArgs(t *testing.T) {
	origOsArgs := OsArgs
	defer func() { OsArgs = origOsArgs }()

	// Ensure OsArgs is never called when SetArgs is used
	OsArgs = func() []string {
		t.Fatal("OsArgs should not be called when SetArgs was used")
		return nil
	}

	var gotArgs []string
	cmd := &Command{
		Use: "test",
		Run: func(cmd *Command, args []string) {
			gotArgs = args
		},
	}
	cmd.SetArgs([]string{"foo", "bar"})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Fatal(err)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "foo" || gotArgs[1] != "bar" {
		t.Fatalf("expected [foo bar], got %v", gotArgs)
	}
}

// ---------------------------------------------------------------------------
// Issue #16: cli: Unconditional os.Getenv throughout
// Environment reads scattered across flag defaults, active help,
// completions, debug logging. No abstraction layer.
// ---------------------------------------------------------------------------

func TestIssue16_EnvLookupFuncUsedInActiveHelp(t *testing.T) {
	origEnvLookup := EnvLookupFunc
	defer func() { EnvLookupFunc = origEnvLookup }()

	EnvLookupFunc = func(key string) string {
		if key == "CLI_ACTIVE_HELP" {
			return "0" // globally disabled
		}
		return ""
	}

	cmd := &Command{Use: "testcmd"}
	cfg := GetActiveHelpConfig(cmd)
	if cfg != "0" {
		t.Fatalf("GetActiveHelpConfig should use EnvLookupFunc and return '0', got '%s'", cfg)
	}

	// Now test per-program override
	EnvLookupFunc = func(key string) string {
		if key == "CLI_ACTIVE_HELP" {
			return "" // not globally disabled
		}
		if key == "TESTCMD_ACTIVE_HELP" {
			return "1"
		}
		return ""
	}

	cfg = GetActiveHelpConfig(cmd)
	if cfg != "1" {
		t.Fatalf("GetActiveHelpConfig should read per-program var via EnvLookupFunc, got '%s'", cfg)
	}
}

func TestIssue16_EnvLookupFuncUsedInGetEnvConfig(t *testing.T) {
	origEnvLookup := EnvLookupFunc
	defer func() { EnvLookupFunc = origEnvLookup }()

	EnvLookupFunc = func(key string) string {
		if key == "MYAPP_COMPLETION_DESCRIPTIONS" {
			return "false"
		}
		return ""
	}

	cmd := &Command{Use: "myapp"}
	cfg := getEnvConfig(cmd, configEnvVarSuffixDescriptions)
	if cfg != "false" {
		t.Fatalf("getEnvConfig should use EnvLookupFunc, got '%s'", cfg)
	}
}

func TestIssue16_EnvLookupFuncCustomImplementation(t *testing.T) {
	origEnvLookup := EnvLookupFunc
	defer func() { EnvLookupFunc = origEnvLookup }()

	// Simulate WASI environment with no env var support
	EnvLookupFunc = func(key string) string {
		return "" // always empty
	}

	cmd := &Command{Use: "wasiapp"}
	cfg := GetActiveHelpConfig(cmd)
	if cfg != "" {
		t.Fatalf("EnvLookupFunc returning empty should produce empty config, got '%s'", cfg)
	}
}

// ---------------------------------------------------------------------------
// Issue #17: cli: Direct os.Stdin/os.Stdout/os.Stderr usage
// Default I/O falls back to OS file descriptors. Nil writers never checked.
// WASI/JS may have nil/closed stdio. VM's custom I/O streams not wired through.
// ---------------------------------------------------------------------------

func TestIssue17_NilDefaultStdioSafe(t *testing.T) {
	origStdout := DefaultStdout
	origStderr := DefaultStderr
	origStdin := DefaultStdin
	defer func() {
		DefaultStdout = origStdout
		DefaultStderr = origStderr
		DefaultStdin = origStdin
	}()

	// Simulate nil stdio (as in WASI/JS)
	DefaultStdout = nil
	DefaultStderr = nil
	DefaultStdin = nil

	cmd := &Command{Use: "test"}

	// OutOrStdout should return a safe writer even when DefaultStdout is nil
	out := cmd.OutOrStdout()
	if out == nil {
		t.Fatal("OutOrStdout should never return nil")
	}

	// ErrOrStderr should return a safe writer even when DefaultStderr is nil
	errW := cmd.ErrOrStderr()
	if errW == nil {
		t.Fatal("ErrOrStderr should never return nil")
	}

	// InOrStdin should return a safe reader even when DefaultStdin is nil
	in := cmd.InOrStdin()
	if in == nil {
		t.Fatal("InOrStdin should never return nil")
	}
}

func TestIssue17_CustomDefaultStdio(t *testing.T) {
	origStdout := DefaultStdout
	origStderr := DefaultStderr
	origStdin := DefaultStdin
	defer func() {
		DefaultStdout = origStdout
		DefaultStderr = origStderr
		DefaultStdin = origStdin
	}()

	customOut := new(bytes.Buffer)
	customErr := new(bytes.Buffer)
	customIn := new(bytes.Buffer)

	DefaultStdout = customOut
	DefaultStderr = customErr
	DefaultStdin = customIn

	cmd := &Command{Use: "test"}

	if cmd.OutOrStdout() != customOut {
		t.Fatal("OutOrStdout should use DefaultStdout")
	}
	if cmd.ErrOrStderr() != customErr {
		t.Fatal("ErrOrStderr should use DefaultStderr")
	}
	if cmd.InOrStdin() != customIn {
		t.Fatal("InOrStdin should use DefaultStdin")
	}
}

func TestIssue17_CommandSetOutOverridesDefault(t *testing.T) {
	origStdout := DefaultStdout
	defer func() { DefaultStdout = origStdout }()

	DefaultStdout = new(bytes.Buffer)
	customOut := new(bytes.Buffer)

	cmd := &Command{Use: "test"}
	cmd.SetOut(customOut)

	if cmd.OutOrStdout() != customOut {
		t.Fatal("SetOut should override DefaultStdout")
	}
}

// ---------------------------------------------------------------------------
// Issue #18: cli: os.Exit(1) in CheckErr
// CheckErr called from default help command terminates the entire process —
// not just the CLI, the entire VM host. Bypasses all defers. Not supported on
// WASI.
// ---------------------------------------------------------------------------

func TestIssue18_CheckErrUsesExitFunc(t *testing.T) {
	origExit := ExitFunc
	origStderr := DefaultStderr
	defer func() {
		ExitFunc = origExit
		DefaultStderr = origStderr
	}()

	// Suppress output
	DefaultStderr = io.Discard

	var exitCalled bool
	var exitCode int
	ExitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	CheckErr(fmt.Errorf("test error"))

	if !exitCalled {
		t.Fatal("CheckErr should call ExitFunc, not os.Exit directly")
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestIssue18_CheckErrNilIsNoop(t *testing.T) {
	origExit := ExitFunc
	defer func() { ExitFunc = origExit }()

	ExitFunc = func(code int) {
		t.Fatal("ExitFunc should not be called for nil error")
	}

	CheckErr(nil)
}

func TestIssue18_CheckErrOutputUsesDefaultStderr(t *testing.T) {
	origExit := ExitFunc
	origStderr := DefaultStderr
	defer func() {
		ExitFunc = origExit
		DefaultStderr = origStderr
	}()

	ExitFunc = func(code int) {} // no-op

	buf := new(bytes.Buffer)
	DefaultStderr = buf

	CheckErr("boom")

	if !bytes.Contains(buf.Bytes(), []byte("boom")) {
		t.Fatalf("expected error message in output, got: %s", buf.String())
	}
}

// ---------------------------------------------------------------------------
// Issue #19: cli: Global mutable state without synchronisation
// initializers, finalizers, EnablePrefixMatching, EnableCommandSorting,
// templateFuncs, etc. are unsynchronised package-level vars. Rumo's
// concurrent routines race on them.
// ---------------------------------------------------------------------------

func TestIssue19_ConcurrentOnInitializeAndExecute(t *testing.T) {
	// Reset globals to known state
	globalMu.Lock()
	origInit := initializers
	origFinal := finalizers
	initializers = nil
	finalizers = nil
	globalMu.Unlock()
	defer func() {
		globalMu.Lock()
		initializers = origInit
		finalizers = origFinal
		globalMu.Unlock()
	}()

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			OnInitialize(func() {})
			OnFinalize(func() {})
		}()
	}

	// Concurrent reads via command execution
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := &Command{
				Use: "test",
				Run: func(cmd *Command, args []string) {},
			}
			cmd.SetArgs([]string{})
			cmd.Execute()
		}()
	}

	wg.Wait()
}

func TestIssue19_ConcurrentTemplateFuncAccess(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			AddTemplateFunc(fmt.Sprintf("testfn%d", n), func() string { return "" })
		}(i)
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := &Command{
				Use: "test",
				Run: func(cmd *Command, args []string) {},
			}
			cmd.SetArgs([]string{})
			cmd.Execute()
		}()
	}

	wg.Wait()
}

func TestIssue19_ConcurrentConfigVarAccess(t *testing.T) {
	// This test verifies no data race when reading/writing config vars
	// concurrently. Run with -race to detect races.
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			SetEnablePrefixMatching(n%2 == 0)
			SetEnableCommandSorting(n%2 == 0)
			SetEnableCaseInsensitive(n%2 == 0)
			SetEnableTraverseRunHooks(n%2 == 0)
		}(i)
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = GetEnablePrefixMatching()
			_ = GetEnableCommandSorting()
			_ = GetEnableCaseInsensitive()
			_ = GetEnableTraverseRunHooks()
		}()
	}

	wg.Wait()
}

// Regression: verify OnInitialize/OnFinalize callbacks are actually invoked
func TestIssue19_OnInitializeAndOnFinalizeStillWork(t *testing.T) {
	globalMu.Lock()
	origInit := initializers
	origFinal := finalizers
	initializers = nil
	finalizers = nil
	globalMu.Unlock()
	defer func() {
		globalMu.Lock()
		initializers = origInit
		finalizers = origFinal
		globalMu.Unlock()
	}()

	initCalled := false
	finalCalled := false
	OnInitialize(func() { initCalled = true })
	OnFinalize(func() { finalCalled = true })

	cmd := &Command{
		Use: "test",
		Run: func(cmd *Command, args []string) {},
	}
	cmd.SetArgs([]string{})
	cmd.Execute()

	if !initCalled {
		t.Fatal("OnInitialize callback should have been called")
	}
	if !finalCalled {
		t.Fatal("OnFinalize callback should have been called")
	}
}

