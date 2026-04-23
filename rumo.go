package rumo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/malivvan/readline"
	"github.com/malivvan/readline/term"
	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
	"github.com/malivvan/rumo/vm/parser"
)

var (
	version string
	commit  string
)

// Version returns the version of rumo.
func Version() string {
	if version == "" {
		return "unknown"
	}
	return version
}

// Commit returns the commit hash of rumo.
func Commit() string {
	if commit == "" {
		return "unknown"
	}
	return commit
}

// Modules returns the lazily-initialized module map containing all standard
// library modules. The map is computed on first call and cached for subsequent
// calls. (Issue #11: avoids eager init that forces all stdlib into every binary.)
func Modules() *vm.ModuleMap {
	modulesOnce.Do(func() {
		modulesCache = GetModuleMap(AllModuleNames()...)
	})
	return modulesCache
}

// Exports returns the lazily-initialized export map of all standard library
// modules. The map is computed on first call and cached for subsequent calls.
func Exports() map[string]map[string]*module.Export {
	exportsOnce.Do(func() {
		exportsCache = GetExportMap(AllModuleNames()...)
	})
	return exportsCache
}

var (
	modulesOnce  sync.Once
	modulesCache *vm.ModuleMap
	exportsOnce  sync.Once
	exportsCache map[string]map[string]*module.Export
)

// CompileOnly compiles the source code and writes the compiled binary into
// outputFile.
func CompileOnly(data []byte, inputFile, outputFile string) (err error) {
	program, err := compileSrc(data, inputFile)
	if err != nil {
		return
	}

	if outputFile == "" {
		outputFile = basename(inputFile) + ".out"
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = out.Close()
		} else {
			err = out.Close()
		}
	}()

	b, err := program.Marshal()
	if err != nil {
		return
	}
	_, err = out.Write(b)
	if err != nil {
		return fmt.Errorf("error writing to output file %s: %w", outputFile, err)
	}

	return
}

// CompileAndRun compiles the source code and executes it.
func CompileAndRun(ctx context.Context, data []byte, inputFile string, args []string) (err error) {
	p, err := compileSrc(data, inputFile)
	if err != nil {
		return
	}
	p.SetArgs(args)
	err = p.RunContext(ctx)
	return
}

// RunCompiled reads the compiled binary from file and executes it.
func RunCompiled(ctx context.Context, data []byte, args []string) (err error) {
	p := &Program{}
	err = p.Unmarshal(data)
	if err != nil {
		return
	}
	p.SetArgs(args)
	err = p.RunContext(ctx)
	return
}

// replCompleter implements shell.AutoCompleter for the REPL, providing
// tab-completion for builtin functions, globally imported module names,
// user-defined symbols, and module member access (e.g. fmt.println).
type replCompleter struct {
	exports     map[string]map[string]*module.Export
	symbolTable *vm.SymbolTable
}

func (c *replCompleter) Do(line []rune, pos int) ([][]rune, int) {
	// Walk backwards from cursor to find the current token.
	start := pos
	for start > 0 && (isIdentChar(line[start-1]) || line[start-1] == '.') {
		start--
	}
	text := string(line[start:pos])

	// Module member completion (e.g. "fmt.pr" → "intln", "intf", …).
	if dotIdx := strings.LastIndex(text, "."); dotIdx >= 0 {
		modName := text[:dotIdx]
		memberPrefix := text[dotIdx+1:]
		exports, ok := c.exports[modName]
		if !ok {
			return nil, 0
		}
		var candidates [][]rune
		for name := range exports {
			if strings.HasPrefix(name, memberPrefix) {
				candidates = append(candidates, []rune(name[len(memberPrefix):]))
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			return string(candidates[i]) < string(candidates[j])
		})
		return candidates, len(memberPrefix)
	}

	// Top-level completion: symbol table contains builtins, globally imported
	// module names, and any user-defined variables from earlier REPL lines.
	prefix := text
	seen := make(map[string]bool)
	var candidates [][]rune
	for _, name := range c.symbolTable.Names() {
		if seen[name] || strings.HasPrefix(name, "__") {
			continue
		}
		if strings.HasPrefix(name, prefix) {
			seen[name] = true
			candidates = append(candidates, []rune(name[len(prefix):]))
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return string(candidates[i]) < string(candidates[j])
	})
	return candidates, len(prefix)
}

func isIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

// RunREPL starts REPL. If modules is non-nil, each named module is imported
// globally (available as a top-level variable without an explicit import call).
func RunREPL(ctx context.Context, in io.Reader, out io.Writer, prompt string, modules []string) {
	// Determine if we're running in an interactive terminal
	interactive := false
	if fin, ok := in.(*os.File); ok {
		if fout, ok := out.(*os.File); ok {
			interactive = term.IsTerminal(int(fin.Fd())) && term.IsTerminal(int(fout.Fd()))
		}
	}

	fileSet := parser.NewFileSet()
	globals := make([]vm.Object, vm.GlobalsSize)
	symbolTable := vm.NewSymbolTable()
	for idx, fn := range vm.GetAllBuiltinFunctions() {
		symbolTable.DefineBuiltin(idx, fn.Name)
	}

	// import modules globally
	for _, name := range modules {
		if mod, ok := BuiltinModules[name]; ok {
			sym := symbolTable.Define(name)
			globals[sym.Index] = (&vm.BuiltinModule{Attrs: mod.Objects()}).AsImmutableMap(name)
		} else if _, ok := SourceModules[name]; ok {
			s := NewScript([]byte(fmt.Sprintf(`__result__ := import("%s")`, name)))
			s.SetImports(Modules())
			p, err := s.RunContext(ctx)
			if err == nil {
				sym := symbolTable.Define(name)
				globals[sym.Index] = p.Get("__result__").Object()
			}
		}
	}

	rl, err := readline.NewFromConfig(&readline.Config{
		Prompt:          prompt,
		Stdin:           in,
		Stdout:          out,
		Stderr:          out,
		InterruptPrompt: "\n",
		EOFPrompt:       "\n",
		HistoryLimit:    1000,
		Undo:            true,
		FuncIsTerminal:  func() bool { return interactive },
		AutoComplete:    &replCompleter{exports: Exports(), symbolTable: symbolTable},
	})
	if err != nil {
		_, _ = fmt.Fprintln(out, err.Error())
		return
	}
	defer rl.Close()

	// embed println function
	symbol := symbolTable.Define("__repl_println__")
	globals[symbol.Index] = &vm.BuiltinFunction{
		Name: "println",
		Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
			var printArgs []interface{}
			for _, arg := range args {
				if _, isUndefined := arg.(*vm.Undefined); isUndefined {
					printArgs = append(printArgs, "<undefined>")
				} else {
					s, _ := vm.ToString(arg)
					printArgs = append(printArgs, s)
				}
			}
			printArgs = append(printArgs, "\n")
			_, _ = fmt.Fprint(rl.Stdout(), printArgs...)
			return
		},
	}

	var constants []vm.Object
	for {
		if ctx.Err() != nil {
			return
		}
		if !interactive {
			_, _ = fmt.Fprint(out, prompt)
		}
		line, readErr := rl.ReadLine()
		if readErr != nil {
			if readErr == readline.ErrInterrupt {
				continue
			}
			return // io.EOF or other error, exit REPL
		}

		srcFile := fileSet.AddFile("repl", -1, len(line))
		p := parser.NewParser(srcFile, []byte(line), nil)
		file, err := p.ParseFile()
		if err != nil {
			_, _ = fmt.Fprintln(rl.Stdout(), err.Error())
			continue
		}

		file = addPrints(file)
		c := vm.NewCompiler(srcFile, symbolTable, constants, Modules(), nil)
		if err := c.Compile(file); err != nil {
			_, _ = fmt.Fprintln(rl.Stdout(), err.Error())
			continue
		}

		bytecode := c.Bytecode()
		machine := vm.NewVM(ctx, bytecode, globals, -1)
		// Propagate the custom In/Out streams so that stdlib modules (e.g.
		// fmt.print/println) write to the provided writer instead of os.Stdout.
		machine.In = in
		machine.Out = rl.Stdout()
		machine.Args = []string{} // REPL scripts have no args by default
		if err := machine.Run(); err != nil {
			_, _ = fmt.Fprintln(rl.Stdout(), err.Error())
			continue
		}
		constants = bytecode.Constants
	}
}

func compileSrc(src []byte, inputFile string) (*Program, error) {
	s := NewScript(src)
	s.SetName(inputFile)
	s.SetImports(Modules())
	s.EnableFileImport(true)
	if err := s.SetImportDir(filepath.Dir(inputFile)); err != nil {
		return nil, fmt.Errorf("error setting import dir: %w", err)
	}
	return s.Compile()
}

func addPrints(file *parser.File) *parser.File {
	var stmts []parser.Stmt
	for _, s := range file.Stmts {
		switch s := s.(type) {
		case *parser.ExprStmt:
			stmts = append(stmts, &parser.ExprStmt{
				Expr: &parser.CallExpr{
					Func: &parser.Ident{Name: "__repl_println__"},
					Args: []parser.Expr{s.Expr},
				},
			})
		case *parser.AssignStmt:
			stmts = append(stmts, s)

			stmts = append(stmts, &parser.ExprStmt{
				Expr: &parser.CallExpr{
					Func: &parser.Ident{
						Name: "__repl_println__",
					},
					Args: s.LHS,
				},
			})
		default:
			stmts = append(stmts, s)
		}
	}
	return &parser.File{
		InputFile: file.InputFile,
		Stmts:     stmts,
	}
}

func basename(s string) string {
	s = filepath.Base(s)
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}
