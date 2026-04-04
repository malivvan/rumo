package std

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/malivvan/vv/vm"
)

var osModule = map[string]vm.Object{
	"o_rdonly":            &vm.Int{Value: int64(os.O_RDONLY)},
	"o_wronly":            &vm.Int{Value: int64(os.O_WRONLY)},
	"o_rdwr":              &vm.Int{Value: int64(os.O_RDWR)},
	"o_append":            &vm.Int{Value: int64(os.O_APPEND)},
	"o_create":            &vm.Int{Value: int64(os.O_CREATE)},
	"o_excl":              &vm.Int{Value: int64(os.O_EXCL)},
	"o_sync":              &vm.Int{Value: int64(os.O_SYNC)},
	"o_trunc":             &vm.Int{Value: int64(os.O_TRUNC)},
	"mode_dir":            &vm.Int{Value: int64(os.ModeDir)},
	"mode_append":         &vm.Int{Value: int64(os.ModeAppend)},
	"mode_exclusive":      &vm.Int{Value: int64(os.ModeExclusive)},
	"mode_temporary":      &vm.Int{Value: int64(os.ModeTemporary)},
	"mode_symlink":        &vm.Int{Value: int64(os.ModeSymlink)},
	"mode_device":         &vm.Int{Value: int64(os.ModeDevice)},
	"mode_named_pipe":     &vm.Int{Value: int64(os.ModeNamedPipe)},
	"mode_socket":         &vm.Int{Value: int64(os.ModeSocket)},
	"mode_setuid":         &vm.Int{Value: int64(os.ModeSetuid)},
	"mode_setgid":         &vm.Int{Value: int64(os.ModeSetgid)},
	"mode_setgui":         &vm.Int{Value: int64(os.ModeSetgid)},
	"mode_char_device":    &vm.Int{Value: int64(os.ModeCharDevice)},
	"mode_sticky":         &vm.Int{Value: int64(os.ModeSticky)},
	"mode_irregular":      &vm.Int{Value: int64(os.ModeIrregular)},
	"mode_type":           &vm.Int{Value: int64(os.ModeType)},
	"mode_perm":           &vm.Int{Value: int64(os.ModePerm)},
	"path_separator":      &vm.Char{Value: os.PathSeparator},
	"path_list_separator": &vm.Char{Value: os.PathListSeparator},
	"dev_null":            &vm.String{Value: os.DevNull},
	"seek_set":            &vm.Int{Value: int64(io.SeekStart)},
	"seek_cur":            &vm.Int{Value: int64(io.SeekCurrent)},
	"seek_end":            &vm.Int{Value: int64(io.SeekEnd)},
	"args": &vm.BuiltinFunction{
		Name:  "args",
		Value: osArgs,
	}, // args() => array(string)
	"chdir": &vm.BuiltinFunction{
		Name:  "chdir",
		Value: FuncASRE(os.Chdir),
	}, // chdir(dir string) => error
	"chmod": osFuncASFmRE("chmod", os.Chmod), // chmod(name string, mode int) => error
	"chown": &vm.BuiltinFunction{
		Name:  "chown",
		Value: FuncASIIRE(os.Chown),
	}, // chown(name string, uid int, gid int) => error
	"clearenv": &vm.BuiltinFunction{
		Name:  "clearenv",
		Value: FuncAR(os.Clearenv),
	}, // clearenv()
	"environ": &vm.BuiltinFunction{
		Name:  "environ",
		Value: FuncARSs(os.Environ),
	}, // environ() => array(string)
	"exit": &vm.BuiltinFunction{
		Name:  "exit",
		Value: FuncAIR(os.Exit),
	}, // exit(code int)
	"expand_env": &vm.BuiltinFunction{
		Name:  "expand_env",
		Value: osExpandEnv,
	}, // expand_env(s string) => string
	"getegid": &vm.BuiltinFunction{
		Name:  "getegid",
		Value: FuncARI(os.Getegid),
	}, // getegid() => int
	"getenv": &vm.BuiltinFunction{
		Name:  "getenv",
		Value: FuncASRS(os.Getenv),
	}, // getenv(s string) => string
	"geteuid": &vm.BuiltinFunction{
		Name:  "geteuid",
		Value: FuncARI(os.Geteuid),
	}, // geteuid() => int
	"getgid": &vm.BuiltinFunction{
		Name:  "getgid",
		Value: FuncARI(os.Getgid),
	}, // getgid() => int
	"getgroups": &vm.BuiltinFunction{
		Name:  "getgroups",
		Value: FuncARIsE(os.Getgroups),
	}, // getgroups() => array(int)/error
	"getpagesize": &vm.BuiltinFunction{
		Name:  "getpagesize",
		Value: FuncARI(os.Getpagesize),
	}, // getpagesize() => int
	"getpid": &vm.BuiltinFunction{
		Name:  "getpid",
		Value: FuncARI(os.Getpid),
	}, // getpid() => int
	"getppid": &vm.BuiltinFunction{
		Name:  "getppid",
		Value: FuncARI(os.Getppid),
	}, // getppid() => int
	"getuid": &vm.BuiltinFunction{
		Name:  "getuid",
		Value: FuncARI(os.Getuid),
	}, // getuid() => int
	"getwd": &vm.BuiltinFunction{
		Name:  "getwd",
		Value: FuncARSE(os.Getwd),
	}, // getwd() => string/error
	"hostname": &vm.BuiltinFunction{
		Name:  "hostname",
		Value: FuncARSE(os.Hostname),
	}, // hostname() => string/error
	"lchown": &vm.BuiltinFunction{
		Name:  "lchown",
		Value: FuncASIIRE(os.Lchown),
	}, // lchown(name string, uid int, gid int) => error
	"link": &vm.BuiltinFunction{
		Name:  "link",
		Value: FuncASSRE(os.Link),
	}, // link(oldname string, newname string) => error
	"lookup_env": &vm.BuiltinFunction{
		Name:  "lookup_env",
		Value: osLookupEnv,
	}, // lookup_env(key string) => string/false
	"mkdir":     osFuncASFmRE("mkdir", os.Mkdir),        // mkdir(name string, perm int) => error
	"mkdir_all": osFuncASFmRE("mkdir_all", os.MkdirAll), // mkdir_all(name string, perm int) => error
	"readlink": &vm.BuiltinFunction{
		Name:  "readlink",
		Value: FuncASRSE(os.Readlink),
	}, // readlink(name string) => string/error
	"remove": &vm.BuiltinFunction{
		Name:  "remove",
		Value: FuncASRE(os.Remove),
	}, // remove(name string) => error
	"remove_all": &vm.BuiltinFunction{
		Name:  "remove_all",
		Value: FuncASRE(os.RemoveAll),
	}, // remove_all(name string) => error
	"rename": &vm.BuiltinFunction{
		Name:  "rename",
		Value: FuncASSRE(os.Rename),
	}, // rename(oldpath string, newpath string) => error
	"setenv": &vm.BuiltinFunction{
		Name:  "setenv",
		Value: FuncASSRE(os.Setenv),
	}, // setenv(key string, value string) => error
	"symlink": &vm.BuiltinFunction{
		Name:  "symlink",
		Value: FuncASSRE(os.Symlink),
	}, // symlink(oldname string newname string) => error
	"temp_dir": &vm.BuiltinFunction{
		Name:  "temp_dir",
		Value: FuncARS(os.TempDir),
	}, // temp_dir() => string
	"truncate": &vm.BuiltinFunction{
		Name:  "truncate",
		Value: FuncASI64RE(os.Truncate),
	}, // truncate(name string, size int) => error
	"unsetenv": &vm.BuiltinFunction{
		Name:  "unsetenv",
		Value: FuncASRE(os.Unsetenv),
	}, // unsetenv(key string) => error
	"create": &vm.BuiltinFunction{
		Name:  "create",
		Value: osCreate,
	}, // create(name string) => imap(file)/error
	"open": &vm.BuiltinFunction{
		Name:  "open",
		Value: osOpen,
	}, // open(name string) => imap(file)/error
	"open_file": &vm.BuiltinFunction{
		Name:  "open_file",
		Value: osOpenFile,
	}, // open_file(name string, flag int, perm int) => imap(file)/error
	"find_process": &vm.BuiltinFunction{
		Name:  "find_process",
		Value: osFindProcess,
	}, // find_process(pid int) => imap(process)/error
	"start_process": &vm.BuiltinFunction{
		Name:  "start_process",
		Value: osStartProcess,
	}, // start_process(name string, argv array(string), dir string, env array(string)) => imap(process)/error
	"exec_look_path": &vm.BuiltinFunction{
		Name:  "exec_look_path",
		Value: FuncASRSE(exec.LookPath),
	}, // exec_look_path(file) => string/error
	"exec": &vm.BuiltinFunction{
		Name:  "exec",
		Value: osExec,
	}, // exec(name, args...) => command
	"stat": &vm.BuiltinFunction{
		Name:  "stat",
		Value: osStat,
	}, // stat(name) => imap(fileinfo)/error
	"read_file": &vm.BuiltinFunction{
		Name:  "read_file",
		Value: osReadFile,
	}, // readfile(name) => array(byte)/error
}

func osReadFile(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	fname, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	bytes, err := os.ReadFile(fname)
	if err != nil {
		return wrapError(err), nil
	}
	if len(bytes) > vm.MaxBytesLen {
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
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	stat, err := os.Stat(fname)
	if err != nil {
		return wrapError(err), nil
	}
	fstat := &vm.ImmutableMap{
		Value: map[string]vm.Object{
			"name":  &vm.String{Value: stat.Name()},
			"mtime": &vm.Time{Value: stat.ModTime()},
			"size":  &vm.Int{Value: stat.Size()},
			"mode":  &vm.Int{Value: int64(stat.Mode())},
		},
	}
	if stat.IsDir() {
		fstat.Value["directory"] = vm.TrueValue
	} else {
		fstat.Value["directory"] = vm.FalseValue
	}
	return fstat, nil
}

func osCreate(ctx context.Context, args ...vm.Object) (vm.Object, error) {
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
	res, err := os.Create(s1)
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osOpen(ctx context.Context, args ...vm.Object) (vm.Object, error) {
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
	res, err := os.Open(s1)
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osOpenFile(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 3 {
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
	i2, ok := vm.ToInt(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := vm.ToInt(args[2])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
	}
	res, err := os.OpenFile(s1, i2, os.FileMode(i3))
	if err != nil {
		return wrapError(err), nil
	}
	return makeOSFile(res), nil
}

func osArgs(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	if len(args) != 0 {
		return nil, vm.ErrWrongNumArguments
	}
	arr := &vm.Array{}
	for _, osArg := range v.Args {
		if len(osArg) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		arr.Value = append(arr.Value, &vm.String{Value: osArg})
	}
	return arr, nil
}

func osFuncASFmRE(
	name string,
	fn func(string, os.FileMode) error,
) *vm.BuiltinFunction {
	return &vm.BuiltinFunction{
		Name: name,
		Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 2 {
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
			i2, ok := vm.ToInt64(args[1])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{
					Name:     "second",
					Expected: "int(compatible)",
					Found:    args[1].TypeName(),
				}
			}
			return wrapError(fn(s1, os.FileMode(i2))), nil
		},
	}
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
	if len(res) > vm.MaxStringLen {
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
	var vlen int
	var failed bool
	s := os.Expand(s1, func(k string) string {
		if failed {
			return ""
		}
		v := os.Getenv(k)

		// this does not count the other texts that are not being replaced
		// but the code checks the final length at the end
		vlen += len(v)
		if vlen > vm.MaxStringLen {
			failed = true
			return ""
		}
		return v
	})
	if failed || len(s) > vm.MaxStringLen {
		return nil, vm.ErrStringLimit
	}
	return &vm.String{Value: s}, nil
}

func osExec(ctx context.Context, args ...vm.Object) (vm.Object, error) {
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
		return wrapError(err), nil
	}
	return makeOSProcess(proc), nil
}

func osStartProcess(ctx context.Context, args ...vm.Object) (vm.Object, error) {
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
		return wrapError(err), nil
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
