//go:build tinygo

// TinyGo stub of the rumo `os` standard module.
//
// TinyGo's `os` and `os/exec` packages omit a number of Unix-specific
// symbols (Chmod, Chown, Getgroups, Getpagesize, Lchown, Link,
// File.Chown, ProcessState.Pid, …) that the full implementation in
// `os.go` reaches for. Rather than guard every call site with build
// tags, this file replaces the entire module with a minimal, mostly
// empty surface so that the rumo runtime still links under TinyGo.
//
// When a script imports `os` in a TinyGo build, the only symbols it
// will see are the platform-agnostic flag/constant accessors and a
// handful of safe environment helpers. Any filesystem / process
// facility returns "operation not supported on tinygo".
package os

import (
	"errors"
	"io"
	stdos "os"

	"github.com/malivvan/rumo/vm/module"
)

var errUnsupported = errors.New("os: operation not supported in tinygo build")

// Module is the TinyGo-friendly subset of the `os` module. It exposes the
// constant-only surface (open-flag accessors, FileMode bits, seek
// constants) plus environment getters/setters, which TinyGo supports.
// Anything that touches process state, file descriptors, or OS
// permissions is omitted entirely.
var Module = module.NewBuiltin().
	Func("o_rdonly() (value int)", func() int { return stdos.O_RDONLY }).
	Func("o_wronly() (value int)", func() int { return stdos.O_WRONLY }).
	Func("o_rdwr() (value int)", func() int { return stdos.O_RDWR }).
	Func("o_append() (value int)", func() int { return stdos.O_APPEND }).
	Func("o_create() (value int)", func() int { return stdos.O_CREATE }).
	Func("o_excl() (value int)", func() int { return stdos.O_EXCL }).
	Func("o_sync() (value int)", func() int { return stdos.O_SYNC }).
	Func("o_trunc() (value int)", func() int { return stdos.O_TRUNC }).
	Const("mode_dir int", int64(stdos.ModeDir)).
	Const("mode_append int", int64(stdos.ModeAppend)).
	Const("mode_exclusive int", int64(stdos.ModeExclusive)).
	Const("mode_temporary int", int64(stdos.ModeTemporary)).
	Const("mode_symlink int", int64(stdos.ModeSymlink)).
	Const("mode_device int", int64(stdos.ModeDevice)).
	Const("mode_named_pipe int", int64(stdos.ModeNamedPipe)).
	Const("mode_socket int", int64(stdos.ModeSocket)).
	Const("mode_setuid int", int64(stdos.ModeSetuid)).
	Const("mode_setgui int", int64(stdos.ModeSetgid)).
	Const("mode_char_device int", int64(stdos.ModeCharDevice)).
	Const("mode_sticky int", int64(stdos.ModeSticky)).
	Const("mode_type int", int64(stdos.ModeType)).
	Const("mode_perm int", int64(stdos.ModePerm)).
	Func("path_separator() (value string)", func() string { return string(stdos.PathSeparator) }).
	Func("path_list_separator() (value string)", func() string { return string(stdos.PathListSeparator) }).
	Func("dev_null() (value string)", func() string { return stdos.DevNull }).
	Const("seek_set int", int64(io.SeekStart)).
	Const("seek_cur int", int64(io.SeekCurrent)).
	Const("seek_end int", int64(io.SeekEnd)).
	Func("getenv(s string) (value string)", stdos.Getenv).
	Func("setenv(key string, value string) error", stdos.Setenv).
	Func("unsetenv(key string) error", stdos.Unsetenv).
	Func("environ() (env []string)", stdos.Environ).
	Func("exit(code int)", func(code int64) { stdos.Exit(int(code)) }).
	// Stub implementations for everything that requires unsupported
	// stdlib symbols. They surface a clear error string at script
	// runtime rather than disappearing silently.
	Func("chdir(dir string) error", func(string) error { return errUnsupported }).
	Func("getwd() (dir string)", func() (string, error) { return "", errUnsupported }).
	Func("hostname() (name string)", func() (string, error) { return "", errUnsupported }).
	Func("temp_dir() (dir string)", func() string { return "" })

