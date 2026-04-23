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

// sourceStringLiteral returns a Go expression that evaluates to s.
// It uses a raw string literal (backtick-delimited) for readability, and splits
// around any backtick characters that would otherwise terminate the literal early.
// For example, a source containing a backtick is rendered as:
//
//	`before` + "`" + `after`
func sourceStringLiteral(s string) string {
	if !strings.ContainsRune(s, '`') {
		return "`" + s + "`"
	}
	parts := strings.Split(s, "`")
	var sb strings.Builder
	for i, part := range parts {
		if i > 0 {
			sb.WriteString(" + \"`\" + ")
		}
		sb.WriteByte('`')
		sb.WriteString(part)
		sb.WriteByte('`')
	}
	return sb.String()
}

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
	"sort"

`)
	for _, name := range builtins {
		out.WriteString(fmt.Sprintf("	\"github.com/malivvan/rumo/std/%s\"\n", name))
	}
	out.WriteString(`	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)
`)
	out.WriteString(`
// BuiltinModules are builtin type standard library modules.
var BuiltinModules = map[string]*module.BuiltinModule{` + "\n")
	for _, name := range builtins {
		out.WriteString(fmt.Sprintf("	\"%s\":   %s.Module,\n", name, name))
	}
	out.WriteString("}\n")
	out.WriteString(`
// SourceModules are source type standard library modules.
var SourceModules = map[string]*module.SourceModule{` + "\n")
	for modName, modSrc := range modules {
		out.WriteString("\t\"" + modName + "\": module.NewSource(" + sourceStringLiteral(modSrc) + "),\n")
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
`)

	target := "stdlib.go"
	if err := os.WriteFile(target, out.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
	println("generated " + target + " (" + strconv.Itoa(len(out.Bytes())) + ")")

}
