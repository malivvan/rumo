package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

// --- Doc generation ---

// regexes to extract the first string arg of Func("...") and Const("...") calls.
// Note: in chained style the leading dot is on the previous line, so we match
// Func/Const without requiring a preceding dot.
var reFuncDef = regexp.MustCompile(`\bFunc\("([^"]+)"`)
var reConstDef = regexp.MustCompile(`\bConst\("([^"]+)"`)

// regexes to parse a function/const definition string
var reFuncSig = regexp.MustCompile(`^\s*(\w+)\s*\(([^)]*)\)\s*(?:\(([^)]*)\))?\s*(.*?)\s*$`)
var reConstSig = regexp.MustCompile(`^\s*(\w+)\s+(\w+)\s*(.*?)\s*$`)

type exportKind int

const (
	kindFunc  exportKind = iota
	kindConst exportKind = iota
)

type exportDoc struct {
	kind    exportKind
	name    string
	params  string // functions: raw param string, e.g. "b bytes, n int"
	ret     string // functions: return type string, e.g. "string"
	comment string
}

// extractReturnType picks the type from an output spec like "s string" or "time".
func extractReturnType(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.Fields(s)
	return parts[len(parts)-1]
}

func parseFuncDef(def string) exportDoc {
	m := reFuncSig.FindStringSubmatch(strings.TrimSpace(def))
	if m == nil {
		return exportDoc{kind: kindFunc, name: def}
	}
	return exportDoc{
		kind:    kindFunc,
		name:    m[1],
		params:  strings.TrimSpace(m[2]),
		ret:     extractReturnType(m[3]),
		comment: strings.TrimSpace(m[4]),
	}
}

func parseConstDef(def string) exportDoc {
	m := reConstSig.FindStringSubmatch(strings.TrimSpace(def))
	if m == nil {
		return exportDoc{kind: kindConst, name: def}
	}
	return exportDoc{
		kind:    kindConst,
		name:    m[1],
		comment: strings.TrimSpace(m[3]),
	}
}

// extractModuleDocs scans all non-test .go files under ./std/<modName>/
// and returns exportDocs in source order.
func extractModuleDocs(modName string) []exportDoc {
	var docs []exportDoc
	files, err := filepath.Glob(filepath.Join("./std", modName, "*.go"))
	if err != nil {
		return nil
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if m := reFuncDef.FindStringSubmatch(line); m != nil {
				docs = append(docs, parseFuncDef(m[1]))
			} else if m := reConstDef.FindStringSubmatch(line); m != nil {
				docs = append(docs, parseConstDef(m[1]))
			}
		}
	}
	return docs
}

// generateModuleDoc produces the markdown content for a single module.
func generateModuleDoc(name string, docs []exportDoc) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("---\ntitle: Standard Library - %s\n---\n\n", name))
	sb.WriteString(fmt.Sprintf("## Import\n\n```golang\n%s := import(\"%s\")\n```\n", name, name))

	var consts, funcs []exportDoc
	for _, d := range docs {
		if d.kind == kindConst {
			consts = append(consts, d)
		} else {
			funcs = append(funcs, d)
		}
	}

	if len(consts) > 0 {
		sb.WriteString("\n## Constants\n\n")
		for _, c := range consts {
			if c.comment != "" {
				sb.WriteString(fmt.Sprintf("- `%s`: %s\n", c.name, c.comment))
			} else {
				sb.WriteString(fmt.Sprintf("- `%s`\n", c.name))
			}
		}
	}

	if len(funcs) > 0 {
		sb.WriteString("\n## Functions\n\n")
		for _, f := range funcs {
			sig := fmt.Sprintf("%s(%s)", f.name, f.params)
			if f.ret != "" {
				sig += fmt.Sprintf(" => %s", f.ret)
			}
			if f.comment != "" {
				sb.WriteString(fmt.Sprintf("- `%s`: %s\n", sig, f.comment))
			} else {
				sb.WriteString(fmt.Sprintf("- `%s`\n", sig))
			}
		}
	}

	return sb.String()
}

func main() {
	if !cdRoot() {
		println("failed to find project root")
		os.Exit(1)
	}

	// Collect the filter set from command-line arguments (if any).
	filter := make(map[string]bool)
	for _, arg := range os.Args[1:] {
		filter[arg] = true
	}

	var builtins []string
	var sources []string
	entries, err := os.ReadDir("./std")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "test" {
			if len(filter) == 0 || filter[entry.Name()] {
				builtins = append(builtins, entry.Name())
			}
		} else if strings.HasSuffix(entry.Name(), ".rumo") {
			name := strings.TrimSuffix(entry.Name(), ".rumo")
			if len(filter) == 0 || filter[name] {
				sources = append(sources, name)
			}
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

	// Generate doc/std-*.md for each builtin module
	for _, name := range builtins {
		docs := extractModuleDocs(name)
		docContent := generateModuleDoc(name, docs)
		docPath := filepath.Join("./doc", "std-"+name+".md")
		if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
			log.Fatal(err)
		}
		println("generated " + docPath + " (" + strconv.Itoa(len(docContent)) + ")")
	}
}
