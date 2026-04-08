package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func cdRoot() bool {
	for i := 0; i < 10; i++ {
		info, err := os.Stat("go.mod")
		if err != nil || info.IsDir() {
			os.Chdir("..")
		} else {
			return true
		}
	}
	return false
}

func main() {
	if !cdRoot() {
		println("failed to find project root")
		os.Exit(1)
	}
	if len(os.Args) < 1 {
		println("usage: go run ./std")
		os.Exit(1)
	}
	var builtins []string
	var sources []string
	entries, err := os.ReadDir("./std")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "test" {
			builtins = append(builtins, entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".rumo") {
			sources = append(sources, strings.TrimSuffix(entry.Name(), ".rumo"))
		}
	}

	fmt.Println("builtin modules:", builtins)

	fmt.Println("source modules:", sources)

	modules := make(map[string]string)

	for _, source := range sources {
		body, err := os.ReadFile(filepath.Join("./std", source+".rumo"))
		if err != nil {
			log.Fatal(err)
		}
		modules[source] = string(body)
	}

	var out bytes.Buffer
	out.WriteString(`package rumo

import (
`)
	for _, name := range builtins {
		out.WriteString(fmt.Sprintf("	\"github.com/malivvan/rumo/std/%s\"\n", name))
	}
	out.WriteString(`	"github.com/malivvan/rumo/vm"
)
`)
	out.WriteString(`
// BuiltinModules are source type standard library modules.
var BuiltinModules = map[string]map[string]vm.Object{` + "\n")
	for _, name := range builtins {
		out.WriteString(fmt.Sprintf("	\"%s\":   %s.Module.Objects(),\n", name, name))
	}
	out.WriteString("}\n")
	out.WriteString(`
// SourceModules are source type standard library modules.
var SourceModules = map[string]string{` + "\n")
	for modName, modSrc := range modules {
		out.WriteString("\t\"" + modName + "\": `" + modSrc + "`,\n")
	}
	out.WriteString("}\n")
	out.WriteString(`
// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	var names []string
	for name := range BuiltinModules {
		names = append(names, name)
	}
	for name := range SourceModules {
		names = append(names, name)
	}
	return names
}

// GetModuleMap returns the module map that includes all modules
// for the given module names.
func GetModuleMap(names ...string) *vm.ModuleMap {
	modules := vm.NewModuleMap()
	for _, name := range names {
		if mod := BuiltinModules[name]; mod != nil {
			modules.AddBuiltinModule(name, mod)
		}
		if mod := SourceModules[name]; mod != "" {
			modules.AddSourceModule(name, []byte(mod))
		}
	}
	return modules
}
`)

	target := "stdlib.go"
	if err := os.WriteFile(target, out.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
	println("generated " + target + " (" + strconv.Itoa(len(out.Bytes())) + ")")

}
