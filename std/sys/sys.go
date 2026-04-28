// Package sys exposes a cross-platform subset of operating-system /
// system-call functionality to rumo scripts. The same surface compiles for
// every target Go supports — including js/wasm — because it relies on the
// portable os and runtime packages plus a curated subset of the syscall
// package's Errno and Signal constants whose values are stable across
// linux, darwin, *bsd, windows and js/wasm.
//
// Anything that is genuinely platform-specific (e.g. SIGHUP, SIGSEGV,
// raw system call numbers, ptrace, mmap, …) is intentionally omitted so
// the bytecode produced by `rumo build` stays portable.
package sys

import (
	"context"
	"os"
	"runtime"
	"syscall"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().

	// ----- platform / runtime info -------------------------------------
	// goos/goarch/compiler differ between builds, so they are functions:
	// the value is read at script execution time rather than baked into
	// the bytecode.
	Func("os() (value string)        target operating system (runtime.GOOS)", func() string { return runtime.GOOS }).
	Func("arch() (value string)      target architecture (runtime.GOARCH)", func() string { return runtime.GOARCH }).
	Func("compiler() (value string)    Go compiler used to build the host (gc/gccgo)", func() string { return runtime.Compiler }).
	Func("go_version() (value string)  Go runtime version", func() string { return runtime.Version() }).
	Func("num_cpu() (value int)        number of logical CPUs available", runtime.NumCPU).
	Func("num_goroutine() (value int)  number of currently running goroutines", runtime.NumGoroutine).
	Func("page_size() (value int)      memory page size in bytes", os.Getpagesize).

	// ----- process info -----------------------------------------------
	Func("getpid() (pid int)              calling process id", os.Getpid).
	Func("getppid() (ppid int)            parent process id", os.Getppid).
	Func("getuid() (uid int)              real user id (-1 if unsupported)", os.Getuid).
	Func("geteuid() (euid int)            effective user id (-1 if unsupported)", os.Geteuid).
	Func("getgid() (gid int)              real group id (-1 if unsupported)", os.Getgid).
	Func("getegid() (egid int)            effective group id (-1 if unsupported)", os.Getegid).
	Func("getgroups() (gids []int, err error)  supplementary group ids", os.Getgroups).
	Func("hostname() (name string, err error)  host name reported by the kernel", os.Hostname).
	Func("getwd() (dir string, err error)      current working directory", os.Getwd).

	// ----- environment ------------------------------------------------
	Func("getenv(key string) (value string)                     value of an env var ('' if unset)", osGetenv).
	Func("setenv(key string, value string) error                set an env var", osSetenv).
	Func("unsetenv(key string) error                            remove an env var", osUnsetenv).
	Func("clearenv()                                            remove every env var", osClearenv).
	Func("environ() (env []string)                              all 'KEY=VAL' env entries", os.Environ).
	Func("lookup_env(key string) (value string, found bool)     env var with explicit presence flag", osLookupEnv).
	Func("expand_env(s string) (result string)                  ${var}/$var substitution against the current env", osExpandEnv).

	// ----- exit -------------------------------------------------------
	Func("exit(code int)  terminate the host process immediately", osExit).

	// ----- errno ------------------------------------------------------
	// Standard POSIX-style error numbers. The numeric *values* of these
	// constants differ between platforms (linux vs. darwin vs. windows
	// vs. js); always go through `errno_str` to render a human-readable
	// message rather than comparing raw integers.
	Const("EPERM int", int64(syscall.EPERM)).
	Const("ENOENT int", int64(syscall.ENOENT)).
	Const("ESRCH int", int64(syscall.ESRCH)).
	Const("EINTR int", int64(syscall.EINTR)).
	Const("EIO int", int64(syscall.EIO)).
	Const("ENXIO int", int64(syscall.ENXIO)).
	Const("E2BIG int", int64(syscall.E2BIG)).
	Const("EBADF int", int64(syscall.EBADF)).
	Const("EAGAIN int", int64(syscall.EAGAIN)).
	Const("ENOMEM int", int64(syscall.ENOMEM)).
	Const("EACCES int", int64(syscall.EACCES)).
	Const("EFAULT int", int64(syscall.EFAULT)).
	Const("EBUSY int", int64(syscall.EBUSY)).
	Const("EEXIST int", int64(syscall.EEXIST)).
	Const("EXDEV int", int64(syscall.EXDEV)).
	Const("ENODEV int", int64(syscall.ENODEV)).
	Const("ENOTDIR int", int64(syscall.ENOTDIR)).
	Const("EISDIR int", int64(syscall.EISDIR)).
	Const("EINVAL int", int64(syscall.EINVAL)).
	Const("ENFILE int", int64(syscall.ENFILE)).
	Const("EMFILE int", int64(syscall.EMFILE)).
	Const("EFBIG int", int64(syscall.EFBIG)).
	Const("ENOSPC int", int64(syscall.ENOSPC)).
	Const("ESPIPE int", int64(syscall.ESPIPE)).
	Const("EROFS int", int64(syscall.EROFS)).
	Const("EMLINK int", int64(syscall.EMLINK)).
	Const("EPIPE int", int64(syscall.EPIPE)).
	Const("ENAMETOOLONG int", int64(syscall.ENAMETOOLONG)).
	Const("ENOSYS int", int64(syscall.ENOSYS)).
	Const("ENOTEMPTY int", int64(syscall.ENOTEMPTY)).
	Const("ELOOP int", int64(syscall.ELOOP)).
	Const("ETIMEDOUT int", int64(syscall.ETIMEDOUT)).
	Const("ECONNREFUSED int", int64(syscall.ECONNREFUSED)).
	Const("ECONNRESET int", int64(syscall.ECONNRESET)).
	Const("EHOSTUNREACH int", int64(syscall.EHOSTUNREACH)).
	Const("ENETUNREACH int", int64(syscall.ENETUNREACH)).
	Const("EPROTONOSUPPORT int", int64(syscall.EPROTONOSUPPORT)).
	Const("EAFNOSUPPORT int", int64(syscall.EAFNOSUPPORT)).
	Const("EADDRINUSE int", int64(syscall.EADDRINUSE)).
	Const("EADDRNOTAVAIL int", int64(syscall.EADDRNOTAVAIL)).
	Const("ENOTSOCK int", int64(syscall.ENOTSOCK)).
	Const("EALREADY int", int64(syscall.EALREADY)).
	Const("EINPROGRESS int", int64(syscall.EINPROGRESS)).
	Const("EISCONN int", int64(syscall.EISCONN)).
	Const("ENOTCONN int", int64(syscall.ENOTCONN)).
	Const("EMSGSIZE int", int64(syscall.EMSGSIZE)).
	Func("errno_str(errno int) (msg string)  human-readable error string for the given errno", sysErrnoStr).

	// ----- signals ----------------------------------------------------
	// POSIX signal *numbers*, hard-coded so the same constants are
	// available on every target — including js/wasm and windows where
	// only a tiny subset of syscall.SIGxxx is defined. Sending a signal
	// that the host kernel does not understand is an error at runtime,
	// not at compile time.
	Const("SIGHUP int", int64(1)).
	Const("SIGINT int", int64(2)).
	Const("SIGQUIT int", int64(3)).
	Const("SIGILL int", int64(4)).
	Const("SIGTRAP int", int64(5)).
	Const("SIGABRT int", int64(6)).
	Const("SIGFPE int", int64(8)).
	Const("SIGKILL int", int64(9)).
	Const("SIGSEGV int", int64(11)).
	Const("SIGPIPE int", int64(13)).
	Const("SIGALRM int", int64(14)).
	Const("SIGTERM int", int64(15)).
	Func("kill(pid int, sig int) error  send a signal to a process", sysKill)

// permsFromCtx mirrors std/os: extract the VM permissions from the call
// context, falling back to unrestricted when no VM is present (direct calls
// from tests etc.).
func permsFromCtx(ctx context.Context) vm.Permissions {
	v, ok := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	if !ok || v == nil {
		return vm.UnrestrictedPermissions()
	}
	return v.Permissions()
}

// ----- env wrappers (permission-aware) ------------------------------------

func osGetenv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	k, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	v := os.Getenv(k)
	if len(v) > vm.ConfigFromContext(ctx).MaxStringLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: v}, nil
}

func osLookupEnv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	k, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	v, found := os.LookupEnv(k)
	if !found {
		return vm.FalseValue, nil
	}
	if len(v) > vm.ConfigFromContext(ctx).MaxStringLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: v}, nil
}

func osSetenv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if !permsFromCtx(ctx).AllowEnvWrite {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	k, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	v, ok := vm.ToString(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	return module.WrapError(os.Setenv(k, v)), nil
}

func osUnsetenv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if !permsFromCtx(ctx).AllowEnvWrite {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	k, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return module.WrapError(os.Unsetenv(k)), nil
}

func osClearenv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if !permsFromCtx(ctx).AllowEnvWrite {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 0 {
		return nil, vm.ErrWrongNumArguments
	}
	os.Clearenv()
	return vm.UndefinedValue, nil
}

func osExpandEnv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	s, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	maxLen := vm.ConfigFromContext(ctx).MaxStringLen
	vlen := len(s)
	var failed bool
	out := os.Expand(s, func(k string) string {
		if failed {
			return ""
		}
		v := os.Getenv(k)
		vlen += len(v)
		if vlen > maxLen {
			failed = true
			return ""
		}
		return v
	})
	if failed || len(out) > maxLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: out}, nil
}

// osExit terminates the host process. Permission-gated, mirroring std/os.exit.
func osExit(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if !permsFromCtx(ctx).AllowExit {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	code, ok := vm.ToInt(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	os.Exit(code)
	return vm.UndefinedValue, nil
}

// sysErrnoStr returns the host-rendered message for an errno value.
func sysErrnoStr(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	n, ok := vm.ToInt64(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	s := syscall.Errno(uintptr(n)).Error()
	if len(s) > vm.ConfigFromContext(ctx).MaxStringLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: s}, nil
}

// sysKill sends signal `sig` to process `pid` via os.FindProcess + Signal.
// It returns nil on success or an error object on failure. On targets where
// signals are not supported (notably js/wasm) the underlying call returns an
// error which is surfaced unchanged.
func sysKill(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if !permsFromCtx(ctx).AllowExec {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	pid, ok := vm.ToInt(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	sig, ok := vm.ToInt64(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return module.WrapError(err), nil
	}
	return module.WrapError(proc.Signal(syscall.Signal(sig))), nil
}
