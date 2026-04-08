package rumo_test

import (
	"testing"

	"github.com/malivvan/rumo"
	"github.com/malivvan/rumo/vm/require"
)

func TestAllModuleNames(t *testing.T) {
	names := rumo.AllModuleNames()
	require.Equal(t,
		len(rumo.BuiltinModules)+len(rumo.SourceModules),
		len(names))
}

// TODO
//func TestModulesRun(t *testing.T) {
//	// os.File
//	expect(t, `
//os := import("os")
//out := ""
//
//write_file := func(filename, data) {
//	file := os.create(filename)
//	if !file { return file }
//
//	if res := file.write(bytes(data)); is_error(res) {
//		return res
//	}
//
//	return file.close()
//}
//
//read_file := func(filename) {
//	file := os.open(filename)
//	if !file { return file }
//
//	data := bytes(100)
//	cnt := file.read(data)
//	if  is_error(cnt) {
//		return cnt
//	}
//
//	file.close()
//	return data[:cnt]
//}
//
//if write_file("./temp", "foobar") {
//	out = string(read_file("./temp"))
//}
//
//os.remove("./temp")
//`, "foobar")
//
//	// exec.command
//	expect(t, `
//out := ""
//os := import("os")
//cmd := os.exec("echo", "foo", "bar")
//if !is_error(cmd) {
//	out = cmd.output()
//}
//`, []byte("foo bar\n"))
//
//}

func TestGetModules(t *testing.T) {
	mods := rumo.GetModuleMap()
	require.Equal(t, 0, mods.Len())

	mods = rumo.GetModuleMap("hex", "rand")
	require.Equal(t, 2, mods.Len())
	require.NotNil(t, mods.Get("hex"))
	require.NotNil(t, mods.Get("rand"))

	mods = rumo.GetModuleMap("text", "text")
	require.Equal(t, 1, mods.Len())
	require.NotNil(t, mods.Get("text"))

	mods = rumo.GetModuleMap("nonexisting", "text")
	require.Equal(t, 1, mods.Len())
	require.NotNil(t, mods.Get("text"))
}
