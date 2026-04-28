package rumo

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

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

// Modules returns a fresh module map containing all currently registered
// standard library modules. A new map is built on every call so that modules
// added to BuiltinModules or SourceModules after startup are always reflected.
// Callers that need a stable snapshot should capture the result once and reuse
// it; callers that want to pick up late-registered modules should call Modules
// each time they create a new Script or VM.
func Modules() *vm.ModuleMap {
	return GetModuleMap(AllModuleNames()...)
}

// Exports returns a fresh export map of all currently registered standard
// library modules. Like Modules, it is recomputed on every call so that
// modules added after startup are visible.
func Exports() map[string]map[string]*module.Export {
	return GetExportMap(AllModuleNames()...)
}

// CompileOnly compiles the script at inputFile and writes the compiled binary
// into outputFile.
func CompileOnly(inputFile, outputFile string) (err error) {
	program, err := compileSrc(inputFile)
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

// CompileAndRun compiles the script at inputFile and executes it.
func CompileAndRun(ctx context.Context, inputFile string, args []string) (err error) {
	p, err := compileSrc(inputFile)
	if err != nil {
		return
	}
	p.SetArgs(args)
	err = p.RunContext(ctx)
	return
}

// RunCompiled reads the compiled binary from file and executes it.
func RunCompiled(ctx context.Context, data []byte, args []string) (err error) {
	return RunCompiledWithModules(ctx, data, args, Modules())
}

// RunCompiledWithModules reads the compiled binary and executes it using the
// provided module map for deserialization.  Use this variant when the compiled
// script imports custom builtin modules that are not part of the standard
// library; pass a ModuleMap that contains both the standard modules and any
// custom ones required by the bytecode.
func RunCompiledWithModules(ctx context.Context, data []byte, args []string, modules *vm.ModuleMap) (err error) {
	p := &Program{}
	err = p.UnmarshalWithModules(data, modules)
	if err != nil {
		return
	}
	p.SetArgs(args)
	err = p.RunContext(ctx)
	return
}

// Completer implements shell.AutoCompleter for the REPL, providing
// tab-completion for builtin functions, globally imported module names,
// user-defined symbols, and module member access (e.g. fmt.println).
type Completer struct {
	exports     map[string]map[string]*module.Export
	symbolTable *vm.SymbolTable
}

func (c *Completer) Do(line []rune, pos int) ([][]rune, int) {
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

type ReadLine interface {
	ReadLine() (line string, err error)
}

type readLine struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	prompt string
	buffer bytes.Buffer
}

func (rl *readLine) ReadLine() (line string, err error) {
	_, err = rl.stdout.Write([]byte(rl.prompt))
	r := bufio.NewReader(rl.stdin)
	b, err := r.ReadBytes('\n')
	if err != nil {
		return "", err
	}
	line = string(b[:len(b)-1])
	return line, nil
}

var newReadline = func(prompt string, stdin io.Reader, stdout, stderr io.Writer) func(completer *Completer) (ReadLine, error) {
	return func(completer *Completer) (ReadLine, error) {
		return &readLine{stdin: stdin, stdout: stdout, stderr: stderr, prompt: prompt}, nil
	}
}

// RunREPL starts REPL. If modules is non-nil, each named module is imported
// globally (available as a top-level variable without an explicit import call).
func RunREPL(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer, modules []string) {
	fileSet := parser.NewFileSet()
	globals := make([]vm.Object, vm.DefaultConfig.GlobalsSize)
	symbolTable := vm.NewSymbolTable()
	for idx, fn := range vm.GetAllBuiltinFunctions() {
		symbolTable.DefineBuiltin(idx, fn.Name)
	}

	// import modules globally
	for _, name := range modules {
		if mod, ok := BuiltinModules[name]; ok {
			sym := symbolTable.Define(name)
			globals[sym.Index] = (&vm.BuiltinModule{Attrs: mod.Objects()}).AsFrozenMap(name)
		} else if _, ok := SourceModules[name]; ok {
			src := fmt.Sprintf(`__result__ := import("%s")`, name)
			s := NewScript(MapFS(map[string][]byte{"__repl_mod__.rumo": []byte(src)}), "__repl_mod__.rumo")
			s.SetImports(Modules())
			p, err := s.RunContext(ctx)
			if err == nil {
				sym := symbolTable.Define(name)
				globals[sym.Index] = p.Get("__result__").Object()
			}
		}
	}

	rl, err := newReadline(">> ", stdin, stdout, stderr)(&Completer{
		exports:     Exports(),
		symbolTable: symbolTable,
	})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return
	}

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
			_, _ = fmt.Fprint(stdout, printArgs...)
			return
		},
	}

	var constants []vm.Object
	for {
		if ctx.Err() != nil {
			return
		}
		line, readErr := rl.ReadLine()
		if readErr != nil {
			if strings.ToLower(readErr.Error()) == "interrupt" {
				continue
			}
			return // io.EOF or other error, exit REPL
		}

		srcFile := fileSet.AddFile("repl", -1, len(line))
		p := parser.NewParser(srcFile, []byte(line), nil)
		file, err := p.ParseFile()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			continue
		}

		file = addPrints(file)
		c := vm.NewCompiler(srcFile, symbolTable, constants, Modules(), nil)
		if err := c.Compile(file); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			continue
		}

		bytecode := c.Bytecode()
		machine := vm.NewVM(ctx, bytecode, globals, nil)
		// Propagate the custom In/Out streams so that stdlib modules (e.g.
		// fmt.print/println) write to the provided writer instead of os.Stdout.
		machine.In = stdin
		machine.Out = stdout
		machine.Args = []string{} // REPL scripts have no args by default
		if err := machine.Run(); err != nil {
			_, _ = fmt.Fprintln(stdout, err.Error())
			continue
		}
		constants = bytecode.Constants
	}
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

func isIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

func compileSrc(inputFile string) (*Program, error) {
	s := NewScript(nil, inputFile)
	s.SetImports(Modules())
	return s.Compile()
}

func basename(s string) string {
	// Strip directory component.
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' || s[i] == '\\' {
			s = s[i+1:]
			break
		}
	}
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}
