//go:build !tinygo

package os

import (
	"os"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

func init() {
	Module = Module.Func("getgroups() (gids []int)", os.Getgroups)

	makeOSFileOrig := makeOSFile
	makeOSFile = func(file *os.File) *vm.Map {
		m := makeOSFileOrig(file)

		// chown(uid int, gid int) => true/error
		m["chown"] = &vm.BuiltinFunction{Name: "chown", Value: module.Func(file.Chown)}

		return m
	}

	makeOSProcessStateOrig := makeOSProcessState
	makeOSProcessState = func(state *os.ProcessState) *vm.Map {
		m := makeOSProcessStateOrig(state)

		// additional tinygo-specific modifications can be added here
		m["pid"] = &vm.BuiltinFunction{Name: "pid", Value: module.Func(state.Pid)}

		return m
	}
}
