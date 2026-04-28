package sys_test

import (
	"runtime"
	"syscall"
	"testing"

	"github.com/malivvan/rumo/vm/require"
)

func TestSysRuntime(t *testing.T) {
	require.Module(t, "sys").Call("os").Expect(runtime.GOOS)
	require.Module(t, "sys").Call("arch").Expect(runtime.GOARCH)
	require.Module(t, "sys").Call("compiler").Expect(runtime.Compiler)
	require.Module(t, "sys").Call("go_version").Expect(runtime.Version())
	require.Module(t, "sys").Call("num_cpu").Expect(int64(runtime.NumCPU()))
}

func TestSysProcess(t *testing.T) {
	require.Module(t, "sys").Call("getpid").ExpectNoError()
	require.Module(t, "sys").Call("getppid").ExpectNoError()
	require.Module(t, "sys").Call("getuid").ExpectNoError()
	require.Module(t, "sys").Call("getgid").ExpectNoError()
	require.Module(t, "sys").Call("geteuid").ExpectNoError()
	require.Module(t, "sys").Call("getegid").ExpectNoError()
	require.Module(t, "sys").Call("hostname").ExpectNoError()
	require.Module(t, "sys").Call("page_size").ExpectNoError()
}

func TestSysEnv(t *testing.T) {
	require.Module(t, "sys").Call("setenv", "RUMO_SYS_TEST", "hello").ExpectNoError()
	require.Module(t, "sys").Call("getenv", "RUMO_SYS_TEST").Expect("hello")
	require.Module(t, "sys").Call("expand_env", "value=$RUMO_SYS_TEST").Expect("value=hello")
	require.Module(t, "sys").Call("unsetenv", "RUMO_SYS_TEST").ExpectNoError()
	require.Module(t, "sys").Call("getenv", "RUMO_SYS_TEST").Expect("")
}

func TestSysErrno(t *testing.T) {
	require.Module(t, "sys").Call("errno_str", int64(syscall.ENOENT)).Expect(syscall.ENOENT.Error())
}
