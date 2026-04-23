package rumo

import (
	"sort"

	"github.com/malivvan/rumo/std/base64"
	"github.com/malivvan/rumo/std/fmt"
	"github.com/malivvan/rumo/std/hex"
	"github.com/malivvan/rumo/std/json"
	"github.com/malivvan/rumo/std/math"
	"github.com/malivvan/rumo/std/rand"
	"github.com/malivvan/rumo/std/text"
	"github.com/malivvan/rumo/std/times"
	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

// BuiltinModules are builtin type standard library modules.
var BuiltinModules = map[string]*module.BuiltinModule{
	"base64":   base64.Module,
	"fmt":   fmt.Module,
	"hex":   hex.Module,
	"json":   json.Module,
	"math":   math.Module,
	"rand":   rand.Module,
	"text":   text.Module,
	"times":   times.Module,
}

// SourceModules are source type standard library modules.
var SourceModules = map[string]*module.SourceModule{
	"enum": module.NewSource("is_enumerable := func(x) {\n  return is_array(x) || is_map(x) || is_immutable_array(x) || is_immutable_map(x)\n}\n\nis_array_like := func(x) {\n  return is_array(x) || is_immutable_array(x)\n}\n\nexport {\n  // all(x map, fn func(key, value) bool) (result bool)\n  // all returns true if the given function 'fn' evaluates to a truthy value on\n  // all of the items in 'x'. It returns undefined if 'x' is not enumerable.\n  all: func(x, fn) {\n    if !is_enumerable(x) { return undefined }\n\n    for k, v in x {\n      if !fn(k, v) { return false }\n    }\n\n    return true\n  },\n\n  // any(x map, fn func(key, value) bool) (result bool)\n  // any returns true if the given function 'fn' evaluates to a truthy value on\n  // any of the items in 'x'. It returns undefined if 'x' is not enumerable.\n  any: func(x, fn) {\n    if !is_enumerable(x) { return undefined }\n\n    for k, v in x {\n      if fn(k, v) { return true }\n    }\n\n    return false\n  },\n\n  // chunk(x array, size int) (result array)\n  // chunk returns an array of elements split into groups the length of size.\n  // If 'x' can't be split evenly, the final chunk will be the remaining elements.\n  // It returns undefined if 'x' is not array.\n  chunk: func(x, size) {\n    if !is_array_like(x) || !size { return undefined }\n\n    numElements := len(x)\n    if !numElements { return [] }\n\n    res := []\n    idx := 0\n    for idx < numElements {\n      res = append(res, x[idx:idx+size])\n      idx += size\n    }\n\n    return res\n  },\n\n  // at(x map, key string) (result any)\n  // at returns an element at the given index (if 'x' is array) or\n  // key (if 'x' is map). It returns undefined if 'x' is not enumerable.\n  at: func(x, key) {\n    if !is_enumerable(x) { return undefined }\n\n    if is_array_like(x) {\n        if !is_int(key) { return undefined }\n    } else {\n        if !is_string(key) { return undefined }\n    }\n\n    return x[key]\n  },\n\n  // each(x map, fn func(key, value)) (result undefined)\n  // each iterates over elements of 'x' and invokes 'fn' for each element. 'fn' is\n  // invoked with two arguments: 'key' and 'value'. 'key' is an int index\n  // if 'x' is array. 'key' is a string key if 'x' is map. It does not iterate\n  // and returns undefined if 'x' is not enumerable.\n  each: func(x, fn) {\n    if !is_enumerable(x) { return undefined }\n\n    for k, v in x {\n      fn(k, v)\n    }\n  },\n\n  // filter(x map, fn func(key, value) bool) (result array)\n  // filter iterates over elements of 'x', returning an array of all elements 'fn'\n  // returns truthy for. 'fn' is invoked with two arguments: 'key' and 'value'.\n  // 'key' is an int index if 'x' is array. 'key' is a string key if 'x' is map.\n  // It returns undefined if 'x' is not enumerable.\n  filter: func(x, fn) {\n    if !is_array_like(x) { return undefined }\n\n    dst := []\n    for k, v in x {\n      if fn(k, v) { dst = append(dst, v) }\n    }\n\n    return dst\n  },\n\n  // find(x map, fn func(key, value) bool) (result any)\n  // find iterates over elements of 'x', returning value of the first element 'fn'\n  // returns truthy for. 'fn' is invoked with two arguments: 'key' and 'value'.\n  // 'key' is an int index if 'x' is array. 'key' is a string key if 'x' is map.\n  // It returns undefined if 'x' is not enumerable.\n  find: func(x, fn) {\n    if !is_enumerable(x) { return undefined }\n\n    for k, v in x {\n      if fn(k, v) { return v }\n    }\n  },\n\n  // find_key(x map, fn func(key, value) bool) (result any)\n  // find_key iterates over elements of 'x', returning key or index of the first\n  // element 'fn' returns truthy for. 'fn' is invoked with two arguments: 'key'\n  // and 'value'. 'key' is an int index if 'x' is array. 'key' is a string key if\n  // 'x' is map. It returns undefined if 'x' is not enumerable.\n  find_key: func(x, fn) {\n    if !is_enumerable(x) { return undefined }\n\n    for k, v in x {\n      if fn(k, v) { return k }\n    }\n  },\n\n  // map(x map, fn func(key, value) any) (result array)\n  // map creates an array of values by running each element in 'x' through 'fn'.\n  // 'fn' is invoked with two arguments: 'key' and 'value'. 'key' is an int index\n  // if 'x' is array. 'key' is a string key if 'x' is map. It returns undefined\n  // if 'x' is not enumerable.\n  map: func(x, fn) {\n    if !is_enumerable(x) { return undefined }\n\n    dst := []\n    for k, v in x {\n      dst = append(dst, fn(k, v))\n    }\n\n    return dst\n  },\n\n  // key(k, v) (result any)\n  // key returns the first argument.\n  key: func(k, _) { return k },\n\n  // value(k, v) (result any)\n  // value returns the second argument.\n  value: func(_, v) { return v }\n}\n"),
}

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range BuiltinModules {
		names = append(names, name)
	}
	for name := range SourceModules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetModuleMap returns the module map that includes all modules
// for the given module names.
func GetModuleMap(names ...string) *vm.ModuleMap {
	modules := vm.NewModuleMap()
	for _, name := range names {
		if mod := BuiltinModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod.Objects())
		}
		if mod := SourceModules[name]; mod != nil {
			modules.AddSourceModule(name, mod.Module())
		}
	}
	return modules
}

// GetExportMap returns the export map of all modules for the given module names.
func GetExportMap(names ...string) map[string]map[string]*module.Export {
	exports := make(map[string]map[string]*module.Export)
	for _, name := range names {
		if mod := BuiltinModules[name]; mod != nil {
			exports[name] = mod.Exports()
		}
		if mod := SourceModules[name]; mod != nil {
			exports[name] = mod.Exports()
		}
	}
	return exports
}
