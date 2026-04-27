package os

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Const("o_rdonly int  			open the file read-only", int64(os.O_RDONLY)).
	Const("o_wronly int				open the file write-only", int64(os.O_WRONLY)).
	Const("o_rdwr int				open the file read-write", int64(os.O_RDWR)).
	Const("o_append int				append data to the file when writing", int64(os.O_APPEND)).
	Const("o_create int				create a new file if none exists", int64(os.O_CREATE)).
	Const("o_excl int				fail if the file already exists", int64(os.O_EXCL)).
	Const("o_sync int				open for synchronous I/O", int64(os.O_SYNC)).
	Const("o_trunc int				truncate regular writable file when opened", int64(os.O_TRUNC)).
	Const("mode_dir int", int64(os.ModeDir)).
	Const("mode_append int", int64(os.ModeAppend)).
	Const("mode_exclusive int", int64(os.ModeExclusive)).
	Const("mode_temporary int", int64(os.ModeTemporary)).
	Const("mode_symlink int", int64(os.ModeSymlink)).
	Const("mode_device int", int64(os.ModeDevice)).
	Const("mode_named_pipe int", int64(os.ModeNamedPipe)).
	Const("mode_socket int", int64(os.ModeSocket)).
	Const("mode_setuid int", int64(os.ModeSetuid)).
	Const("mode_setgui int", int64(os.ModeSetgid)).
	Const("mode_char_device int", int64(os.ModeCharDevice)).
	Const("mode_sticky int", int64(os.ModeSticky)).
	Const("mode_type int", int64(os.ModeType)).
	Const("mode_perm int", int64(os.ModePerm)).
	Const("path_separator string", string(os.PathSeparator)).
	Const("path_list_separator string", string(os.PathListSeparator)).
	Const("dev_null string", os.DevNull).
	Const("seek_set int", int64(io.SeekStart)).
	Const("seek_cur int", int64(io.SeekCurrent)).
	Const("seek_end int", int64(io.SeekEnd)).
	Func("args() (args []string)", osArgs).
	Func("chdir(dir string) error", osChdir).
	Func("chmod(name string, mode int) error", func(s string, i int64) error { return os.Chmod(s, os.FileMode(i)) }).
	Func("chown(name string, uid int, gid int) error", os.Chown).
	Func("clearenv()", osClearenv).
	Func("environ() (env []string)", os.Environ).
	Func("exit(code int)", osExit).
	Func("expand_env(s string) (result string)", osExpandEnv).
	Func("getegid() (egid int)", os.Getegid).
	Func("getenv(s string) (value string)", os.Getenv).
	Func("geteuid() (euid int)", os.Geteuid).
	Func("getgid() (gid int)", os.Getgid).
	Func("getgroups() (gids []int)", os.Getgroups).
	Func("getpagesize() (size int)", os.Getpagesize).
	Func("getpid() (pid int)", os.Getpid).
	Func("getppid() (ppid int)", os.Getppid).
	Func("getuid() (uid int)", os.Getuid).
	Func("getwd() (dir string)", os.Getwd).
	Func("hostname() (name string)", os.Hostname).
	Func("lchown(name string, uid int, gid int) error", os.Lchown).
	Func("link(oldname string, newname string) error", os.Link).
	Func("lookup_env(key string) (value string, found bool)", osLookupEnv).
	Func("mkdir(name string, perm int) error", func(s string, i int64) error { return os.Mkdir(s, os.FileMode(i)) }).
	Func("mkdir_all(name string, perm int) error", func(s string, i int64) error { return os.MkdirAll(s, os.FileMode(i)) }).
	Func("readlink(name string) (result string)", os.Readlink).
	Func("remove(name string) error", os.Remove).
	Func("remove_all(name string) error", os.RemoveAll).
	Func("rename(oldpath string, newpath string) error", os.Rename).
	Func("setenv(key string, value string) error", osSetenv).
	Func("symlink(oldname string, newname string) error", os.Symlink).
	Func("temp_dir() (dir string)", os.TempDir).
	Func("truncate(name string, size int) error", os.Truncate).
	Func("unsetenv(key string) error", osUnsetenv).
	Func("create(name string) (file imap(file))", osCreate).
	Func("open(name string) (file imap(file))", osOpen).
	Func("open_file(name string, flag int, perm int) (file imap(file))", osOpenFile).
	Func("find_process(pid int) (process imap(process))", osFindProcess).
	Func("start_process(name string, argv array(string), dir string, env array(string)) (process imap(process))", osStartProcess).
	Func("exec_look_path(file string) (result string)", module.Func(exec.LookPath)).
	Func("exec(name string, args ...string) (command imap(command))", osExec).
	Func("stat(name string) (fileinfo imap(fileinfo))", osStat).
	Func("read_file(name string) (content bytes)", osReadFile)

// permsFromCtx extracts the VM's Permissions from the execution context.
// Returns zero-value Permissions (all allowed) when no VM is present.
func permsFromCtx(ctx context.Context) vm.Permissions {
	v, ok := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	if !ok || v == nil {
		return vm.Permissions{}
	}
	return v.Permissions()
}

func osExit(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyExit {
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

func osChdir(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyChdir {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	dir, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return module.WrapError(os.Chdir(dir)), nil
}

func osSetenv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyEnvWrite {
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
	if permsFromCtx(ctx).DenyEnvWrite {
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
	if permsFromCtx(ctx).DenyEnvWrite {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 0 {
		return nil, vm.ErrWrongNumArguments
	}
	os.Clearenv()
	return vm.UndefinedValue, nil
}

func osReadFile(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if permsFromCtx(ctx).DenyFileRead {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	fname, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return module.WrapError(err), nil
	}
	if len(bytes) > vm.DefaultConfig.MaxBytesLen {
		return nil, vm.ErrBytesLimit
	}
	return &vm.Bytes{Value: bytes}, nil
}

func osStat(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	fname, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	stat, err := os.Stat(fname)
	if err != nil {
		return module.WrapError(err), nil
	}
	fstat := &vm.ImmutableMap{Value: map[string]vm.Object{"name": &vm.String{Value: stat.Name()}, "mtime": &vm.Time{Value: stat.ModTime()}, "size": &vm.Int{Value: stat.Size()}, "mode": &vm.Int{Value: int64(stat.Mode())}}}
	if stat.IsDir() {
		fstat.Value["directory"] = vm.TrueValue
	} else {
		fstat.Value["directory"] = vm.FalseValue
	}
	return fstat, nil
}

func osCreate(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyFileWrite {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	s1, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := os.Create(s1)
	if err != nil {
		return module.WrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osOpen(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyFileRead {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	s1, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := os.Open(s1)
	if err != nil {
		return module.WrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osOpenFile(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 3 {
		return nil, vm.ErrWrongNumArguments
	}
	s1, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	i2, ok := vm.ToInt(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
	}
	i3, ok := vm.ToInt(args[2])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
	}
	const writeFlags = os.O_WRONLY | os.O_RDWR | os.O_CREATE | os.O_TRUNC | os.O_EXCL | os.O_APPEND
	perms := permsFromCtx(ctx)
	if i2&writeFlags != 0 {
		if perms.DenyFileWrite {
			return nil, vm.ErrNotPermitted
		}
	} else {
		if perms.DenyFileRead {
			return nil, vm.ErrNotPermitted
		}
	}
	res, err := os.OpenFile(s1, i2, os.FileMode(i3))
	if err != nil {
		return module.WrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osArgs(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	if len(args) != 0 {
		return nil, fmt.Errorf("args() takes no arguments, got %d", len(args))
	}
	arr := &vm.Array{}
	for _, osArg := range v.Args {
		if len(osArg) > vm.DefaultConfig.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		arr.Value = append(arr.Value, &vm.String{Value: osArg})
	}
	return arr, nil
}

func osLookupEnv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	s1, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, ok := os.LookupEnv(s1)
	if !ok {
		return vm.FalseValue, nil
	}
	if len(res) > vm.DefaultConfig.MaxStringLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: res}, nil
}

func osExpandEnv(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	s1, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	// Use the per-VM MaxStringLen. permsFromCtx already reaches into the VM;
	// we do the same for Config to avoid reading DefaultConfig directly.
	maxLen := vm.DefaultConfig.MaxStringLen
	if v, ok2 := ctx.Value(vm.ContextKey("vm")).(*vm.VM); ok2 && v != nil {
		if cfg := v.Config(); cfg.MaxStringLen > 0 {
			maxLen = cfg.MaxStringLen
		}
	}

	// Seed the accumulator with the template length as a conservative upper
	// bound: the result is always ≤ len(template) + sum(len(substitutions)).
	// This ensures that a large template with an empty substitution is caught
	// early rather than only by the post-expand check.
	vlen := len(s1)
	var failed bool
	s := os.Expand(s1, func(k string) string {
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
	if failed || len(s) > maxLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: s}, nil
}

func osExec(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyExec {
		return nil, vm.ErrNotPermitted
	}
	if len(args) == 0 {
		return nil, vm.ErrWrongNumArguments
	}
	name, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	var execArgs []string
	for idx, arg := range args[1:] {
		execArg, ok := vm.ToString(arg)
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     fmt.Sprintf("args[%d]", idx),
				Expected: "string(compatible)",
				Found:    args[1+idx].TypeName(),
			}
		}
		execArgs = append(execArgs, execArg)
	}
	return makeOSExecCommand(exec.Command(name, execArgs...)), nil
}

func osFindProcess(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyExec {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	i1, ok := vm.ToInt(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	proc, err := os.FindProcess(i1)
	if err != nil {
		return module.WrapError(err), nil
	}
	return makeOSProcess(proc), nil
}

func osStartProcess(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if permsFromCtx(ctx).DenyExec {
		return nil, vm.ErrNotPermitted
	}
	if len(args) != 4 {
		return nil, vm.ErrWrongNumArguments
	}
	name, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	var argv []string
	var err error
	switch arg1 := args[1].(type) {
	case *vm.Array:
		argv, err = stringArray(arg1.Value, "second")
		if err != nil {
			return nil, err
		}
	case *vm.ImmutableArray:
		argv, err = stringArray(arg1.Value, "second")
		if err != nil {
			return nil, err
		}
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "array",
			Found:    arg1.TypeName(),
		}
	}

	dir, ok := vm.ToString(args[2])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
	}

	var env []string
	switch arg3 := args[3].(type) {
	case *vm.Array:
		env, err = stringArray(arg3.Value, "fourth")
		if err != nil {
			return nil, err
		}
	case *vm.ImmutableArray:
		env, err = stringArray(arg3.Value, "fourth")
		if err != nil {
			return nil, err
		}
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "array",
			Found:    arg3.TypeName(),
		}
	}

	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Dir: dir,
		Env: env,
	})
	if err != nil {
		return module.WrapError(err), nil
	}
	return makeOSProcess(proc), nil
}

func stringArray(arr []vm.Object, argName string) ([]string, error) {
	var sarr []string
	for idx, elem := range arr {
		str, ok := elem.(*vm.String)
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     fmt.Sprintf("%s[%d]", argName, idx),
				Expected: "string",
				Found:    elem.TypeName(),
			}
		}
		sarr = append(sarr, str.Value)
	}
	return sarr, nil
}

func makeOSExecCommand(cmd *exec.Cmd) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			// combined_output() => bytes/error
			"combined_output": &vm.BuiltinFunction{Name: "combined_output", Value: module.Func(cmd.CombinedOutput)},
			// output() => bytes/error
			"output": &vm.BuiltinFunction{Name: "output", Value: module.Func(cmd.Output)}, //
			// run() => error
			"run": &vm.BuiltinFunction{Name: "run", Value: module.Func(cmd.Run)}, //
			// start() => error
			"start": &vm.BuiltinFunction{Name: "start", Value: module.Func(cmd.Start)}, //
			// wait() => error
			"wait": &vm.BuiltinFunction{Name: "wait", Value: module.Func(cmd.Wait)}, //
			// set_path(path string)
			"set_path": &vm.BuiltinFunction{
				Name: "set_path",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					s1, ok := vm.ToString(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					cmd.Path = s1
					return vm.UndefinedValue, nil
				},
			},
			// set_dir(dir string)
			"set_dir": &vm.BuiltinFunction{
				Name: "set_dir",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					s1, ok := vm.ToString(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "string(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					cmd.Dir = s1
					return vm.UndefinedValue, nil
				},
			},
			// set_env(env array(string))
			"set_env": &vm.BuiltinFunction{
				Name: "set_env",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}

					var env []string
					var err error
					switch arg0 := args[0].(type) {
					case *vm.Array:
						env, err = stringArray(arg0.Value, "first")
						if err != nil {
							return nil, err
						}
					case *vm.ImmutableArray:
						env, err = stringArray(arg0.Value, "first")
						if err != nil {
							return nil, err
						}
					default:
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "array",
							Found:    arg0.TypeName(),
						}
					}
					cmd.Env = env
					return vm.UndefinedValue, nil
				},
			},
			// process() => imap(process)
			"process": &vm.BuiltinFunction{
				Name: "process",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 0 {
						return nil, vm.ErrWrongNumArguments
					}
					return makeOSProcess(cmd.Process), nil
				},
			},
		},
	}
}

func makeOSFile(file *os.File) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			// chdir() => true/error
			"chdir": &vm.BuiltinFunction{Name: "chdir", Value: module.Func(file.Chdir)}, //
			// chown(uid int, gid int) => true/error
			"chown": &vm.BuiltinFunction{Name: "chown", Value: module.Func(file.Chown)}, //
			// close() => error
			"close": &vm.BuiltinFunction{Name: "close", Value: module.Func(file.Close)}, //
			// name() => string
			"name": &vm.BuiltinFunction{Name: "name", Value: module.Func(file.Name)}, //
			// readdirnames(n int) => array(string)/error
			"readdirnames": &vm.BuiltinFunction{Name: "readdirnames", Value: module.Func(file.Readdirnames)}, //
			// sync() => error
			"sync": &vm.BuiltinFunction{Name: "sync", Value: module.Func(file.Sync)}, //
			// write(bytes) => int/error
			"write": &vm.BuiltinFunction{Name: "write", Value: module.Func(file.Write)}, //
			// write(string) => int/error
			"write_string": &vm.BuiltinFunction{Name: "write_string", Value: module.Func(file.WriteString)}, //
			// read(bytes) => int/error
			"read": &vm.BuiltinFunction{Name: "read", Value: module.Func(file.Read)}, //
			// chmod(mode int) => error
			"chmod": &vm.BuiltinFunction{
				Name: "chmod",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					i1, ok := vm.ToInt64(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					return module.WrapError(file.Chmod(os.FileMode(i1))), nil
				},
			},
			// seek(offset int, whence int) => int/error
			"seek": &vm.BuiltinFunction{
				Name: "seek",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 2 {
						return nil, vm.ErrWrongNumArguments
					}
					i1, ok := vm.ToInt64(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					i2, ok := vm.ToInt(args[1])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "second",
							Expected: "int(compatible)",
							Found:    args[1].TypeName(),
						}
					}
					res, err := file.Seek(i1, i2)
					if err != nil {
						return module.WrapError(err), nil
					}
					return &vm.Int{Value: res}, nil
				},
			},
			// stat() => imap(fileinfo)/error
			"stat": &vm.BuiltinFunction{
				Name: "stat",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 0 {
						return nil, vm.ErrWrongNumArguments
					}
					return osStat(ctx, &vm.String{Value: file.Name()})
				},
			},
		},
	}
}

func makeOSProcessState(state *os.ProcessState) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"exited":  &vm.BuiltinFunction{Name: "exited", Value: module.Func(state.Exited)},
			"pid":     &vm.BuiltinFunction{Name: "pid", Value: module.Func(state.Pid)},
			"string":  &vm.BuiltinFunction{Name: "string", Value: module.Func(state.String)},
			"success": &vm.BuiltinFunction{Name: "success", Value: module.Func(state.Success)},
		},
	}
}

func makeOSProcess(proc *os.Process) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"kill":    &vm.BuiltinFunction{Name: "kill", Value: module.Func(proc.Kill)},
			"release": &vm.BuiltinFunction{Name: "release", Value: module.Func(proc.Release)},
			"signal": &vm.BuiltinFunction{
				Name: "signal",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 1 {
						return nil, vm.ErrWrongNumArguments
					}
					i1, ok := vm.ToInt64(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "first",
							Expected: "int(compatible)",
							Found:    args[0].TypeName(),
						}
					}
					return module.WrapError(proc.Signal(syscall.Signal(i1))), nil
				},
			},
			"wait": &vm.BuiltinFunction{
				Name: "wait",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					if len(args) != 0 {
						return nil, vm.ErrWrongNumArguments
					}
					state, err := proc.Wait()
					if err != nil {
						return module.WrapError(err), nil
					}
					return makeOSProcessState(state), nil
				},
			},
		},
	}
}
