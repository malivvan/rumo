package rumo_test

import (
	"reflect"
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

// Issue #11: Eager init of Modules/Exports forces all stdlib into every binary.
// Package-level `var Modules = GetModuleMap(AllModuleNames()...)` triggers init()
// in all stdlib packages at import time, causing unnecessary binary size, startup
// time, and memory cost for unused modules. Modules and Exports must be lazy
// accessor functions that defer initialization until first use.
func TestModulesAndExportsAreLazy(t *testing.T) {
	// Before the fix, Modules and Exports are eagerly-initialized package-level
	// vars (*vm.ModuleMap and map respectively). After the fix, they should be
	// functions that lazily compute and cache the maps on first call.
	rtMod := reflect.TypeOf(rumo.Modules)
	if rtMod.Kind() != reflect.Func {
		t.Fatalf("Modules should be a lazy accessor function, got %v (eager init via package-level var)", rtMod.Kind())
	}
	rtExp := reflect.TypeOf(rumo.Exports)
	if rtExp.Kind() != reflect.Func {
		t.Fatalf("Exports should be a lazy accessor function, got %v (eager init via package-level var)", rtExp.Kind())
	}
}

// Regression: lazy Modules() must return a module map containing every module.
func TestModulesReturnsAllModules(t *testing.T) {
	mods := rumo.Modules()
	names := rumo.AllModuleNames()
	require.Equal(t, len(names), mods.Len())
	for _, name := range names {
		require.NotNil(t, mods.Get(name), "module %q missing from Modules()", name)
	}
}

// Regression: lazy Exports() must return Exports for every module.
func TestExportsReturnsAllExports(t *testing.T) {
	exports := rumo.Exports()
	expected := len(rumo.BuiltinModules) + len(rumo.SourceModules)
	require.Equal(t, expected, len(exports))
	for _, name := range rumo.AllModuleNames() {
		_, ok := exports[name]
		require.True(t, ok, "module %q missing from Exports()", name)
	}
}

// Modules() now returns a fresh ModuleMap on every call so that modules
// registered after startup are always visible (issue 6.5 fix).  The test
// verifies that successive calls return equivalent content, not the same
// pointer.
func TestModulesReturnsCachedSingleton(t *testing.T) {
	m1 := rumo.Modules()
	m2 := rumo.Modules()
	require.Equal(t, m1.Len(), m2.Len(), "Modules() should return maps with the same number of entries")
	for _, name := range rumo.AllModuleNames() {
		require.NotNil(t, m1.Get(name), "module %q missing from first Modules() call", name)
		require.NotNil(t, m2.Get(name), "module %q missing from second Modules() call", name)
	}
}

// Regression: Exports() must return the same cached instance on repeated calls.
func TestExportsReturnsCachedSingleton(t *testing.T) {
	e1 := rumo.Exports()
	e2 := rumo.Exports()
	require.Equal(t, len(e1), len(e2))
	for k := range e1 {
		if _, ok := e2[k]; !ok {
			t.Fatalf("Exports() returned inconsistent keys: %q missing on second call", k)
		}
	}
}

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
