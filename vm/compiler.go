package vm

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/malivvan/rumo/vm/parser"
	"github.com/malivvan/rumo/vm/token"
)

// compilationScope represents a compiled instructions and the last two
// instructions that were emitted.
type compilationScope struct {
	Instructions []byte
	SymbolInit   map[string]bool
	SourceMap    map[int]parser.Pos
	hasDefer     bool // true if this scope emitted at least one OpDefer
}

// CompilerWarning represents a non-fatal diagnostic emitted during compilation.
type CompilerWarning struct {
	FileSet *parser.SourceFileSet
	Node    parser.Node
	Message string
}

func (w *CompilerWarning) String() string {
	filePos := w.FileSet.Position(w.Node.Pos())
	return fmt.Sprintf("Compile Warning: %s\n\tat %s", w.Message, filePos)
}

// loop represents a loop construct that the compiler uses to track the current
// loop.
type loop struct {
	Continues []int
	Breaks    []int
}

// switchCtx tracks state needed while compiling a switch statement so that
// nested `fallthrough` and `break` statements can patch the correct jump
// targets.
type switchCtx struct {
	// Fallthroughs is the list of jump-instruction positions emitted by
	// fallthrough statements inside the *current* case clause. The switch
	// compiler patches them to point at the next clause's body start.
	Fallthroughs []int
	// InLastClause is true while compiling the body of the final clause; a
	// fallthrough statement is then a compile error (Go semantics).
	InLastClause bool
}

// CompilerError represents a compiler error.
type CompilerError struct {
	FileSet *parser.SourceFileSet
	Node    parser.Node
	Err     error
}

func (e *CompilerError) Error() string {
	filePos := e.FileSet.Position(e.Node.Pos())
	return fmt.Sprintf("Compile Error: %s\n\tat %s", e.Err.Error(), filePos)
}

// Compiler compiles the AST into a bytecode.
type Compiler struct {
	file            *parser.SourceFile
	parent          *Compiler
	modulePath      string
	importDir       string
	importBase      string
	importEntryDir  string // directory of the entrypoint file; used as base for display paths in FileSet
	importFS        fs.FS  // virtualised filesystem for imports and embeds; nil falls back to os.*
	constants       []Object
	symbolTable     *SymbolTable
	scopes          []compilationScope
	scopeIndex      int
	modules         *ModuleMap
	compiledModules map[string]*CompiledFunction
	allowFileImport bool
	loops           []*loop
	loopIndex       int
	switches        []*switchCtx
	trace           io.Writer
	indent          int
	warnings        []*CompilerWarning
	embeds          []EmbedFile // files baked in via //embed directives
}

// Warnings returns all non-fatal diagnostics emitted during compilation.
// It collects warnings from child (module) compilers as well.
func (c *Compiler) Warnings() []*CompilerWarning {
	// Walk up to root so that warnings collected there are returned.
	root := c
	for root.parent != nil {
		root = root.parent
	}
	return root.warnings
}

// addWarning records a warning on the root compiler so that module-forked
// child compilers also surface warnings to the caller.
func (c *Compiler) addWarning(node parser.Node, format string, args ...interface{}) {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	root.warnings = append(root.warnings, &CompilerWarning{
		FileSet: c.file.Set(),
		Node:    node,
		Message: fmt.Sprintf(format, args...),
	})
}

// NewCompiler creates a Compiler.
func NewCompiler(file *parser.SourceFile, symbolTable *SymbolTable, constants []Object, modules *ModuleMap, trace io.Writer) *Compiler {
	mainScope := compilationScope{
		SymbolInit: make(map[string]bool),
		SourceMap:  make(map[int]parser.Pos),
	}

	// symbol table
	if symbolTable == nil {
		symbolTable = NewSymbolTable()
	}

	// add builtin functions to the symbol table
	for idx, fn := range builtinFuncs {
		symbolTable.DefineBuiltin(idx, fn.Name)
	}

	// builtin modules
	if modules == nil {
		modules = NewModuleMap()
	}

	return &Compiler{
		file:            file,
		symbolTable:     symbolTable,
		constants:       constants,
		scopes:          []compilationScope{mainScope},
		scopeIndex:      0,
		loopIndex:       -1,
		trace:           trace,
		modules:         modules,
		compiledModules: make(map[string]*CompiledFunction),
	}
}

// Compile compiles the AST node.
func (c *Compiler) Compile(node parser.Node) error {
	if c.trace != nil {
		if node != nil {
			defer untracec(tracec(c, fmt.Sprintf("%s", node.String())))
		} else {
			defer untracec(tracec(c, "<nil>"))
		}
	}

	switch node := node.(type) {
	case *parser.File:
		for _, stmt := range node.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	case *parser.ExprStmt:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(node, parser.OpPop)
	case *parser.IncDecStmt:
		op := token.AddAssign
		if node.Token == token.Dec {
			op = token.SubAssign
		}
		return c.compileAssign(node, []parser.Expr{node.Expr},
			[]parser.Expr{&parser.IntLit{Value: 1}}, op)
	case *parser.ParenExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
	case *parser.BinaryExpr:
		if node.Token == token.LAnd || node.Token == token.LOr {
			return c.compileLogical(node)
		}
		if err := c.Compile(node.LHS); err != nil {
			return err
		}
		if err := c.Compile(node.RHS); err != nil {
			return err
		}

		switch node.Token {
		case token.Add:
			c.emit(node, parser.OpBinaryOp, int(token.Add))
		case token.Sub:
			c.emit(node, parser.OpBinaryOp, int(token.Sub))
		case token.Mul:
			c.emit(node, parser.OpBinaryOp, int(token.Mul))
		case token.Quo:
			c.emit(node, parser.OpBinaryOp, int(token.Quo))
		case token.Rem:
			c.emit(node, parser.OpBinaryOp, int(token.Rem))
		case token.Less:
			c.emit(node, parser.OpBinaryOp, int(token.Less))
		case token.LessEq:
			c.emit(node, parser.OpBinaryOp, int(token.LessEq))
		case token.Greater:
			c.emit(node, parser.OpBinaryOp, int(token.Greater))
		case token.GreaterEq:
			c.emit(node, parser.OpBinaryOp, int(token.GreaterEq))
		case token.Equal:
			c.emit(node, parser.OpEqual)
		case token.NotEqual:
			c.emit(node, parser.OpNotEqual)
		case token.And:
			c.emit(node, parser.OpBinaryOp, int(token.And))
		case token.Or:
			c.emit(node, parser.OpBinaryOp, int(token.Or))
		case token.Xor:
			c.emit(node, parser.OpBinaryOp, int(token.Xor))
		case token.AndNot:
			c.emit(node, parser.OpBinaryOp, int(token.AndNot))
		case token.Shl:
			c.emit(node, parser.OpBinaryOp, int(token.Shl))
		case token.Shr:
			c.emit(node, parser.OpBinaryOp, int(token.Shr))
		default:
			return c.errorf(node, "invalid binary operator: %s",
				node.Token.String())
		}
	case *parser.IntLit:
		c.emit(node, parser.OpConstant,
			c.addConstant(&Int{Value: node.Value}))
	case *parser.FloatLit:
		c.emit(node, parser.OpConstant,
			c.addConstant(&Float{Value: node.Value}))
	case *parser.BoolLit:
		if node.Value {
			c.emit(node, parser.OpTrue)
		} else {
			c.emit(node, parser.OpFalse)
		}
	case *parser.StringLit:
		if len(node.Value) > DefaultConfig.MaxStringLen {
			return c.error(node, ErrStringLimit)
		}
		c.emit(node, parser.OpConstant,
			c.addConstant(&String{Value: node.Value}))
	case *parser.CharLit:
		c.emit(node, parser.OpConstant,
			c.addConstant(&Char{Value: node.Value}))
	case *parser.UndefinedLit:
		c.emit(node, parser.OpNull)
	case *parser.UnaryExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}

		switch node.Token {
		case token.Not:
			c.emit(node, parser.OpLNot)
		case token.Sub:
			c.emit(node, parser.OpMinus)
		case token.Xor:
			c.emit(node, parser.OpBComplement)
		case token.Add:
			// do nothing?
		default:
			return c.errorf(node,
				"invalid unary operator: %s", node.Token.String())
		}
	case *parser.IfStmt:
		// open new symbol table for the statement
		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		if node.Init != nil {
			if err := c.Compile(node.Init); err != nil {
				return err
			}
		}
		if err := c.Compile(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1 := c.emit(node, parser.OpJumpFalsy, 0)
		if err := c.Compile(node.Body); err != nil {
			return err
		}
		if node.Else != nil {
			// second jump placeholder
			jumpPos2 := c.emit(node, parser.OpJump, 0)

			// update first jump offset
			curPos := len(c.currentInstructions())
			c.changeOperand(jumpPos1, curPos)
			if err := c.Compile(node.Else); err != nil {
				return err
			}

			// update second jump offset
			curPos = len(c.currentInstructions())
			c.changeOperand(jumpPos2, curPos)
		} else {
			// update first jump offset
			curPos := len(c.currentInstructions())
			c.changeOperand(jumpPos1, curPos)
		}
	case *parser.ForStmt:
		return c.compileForStmt(node)
	case *parser.ForInStmt:
		return c.compileForInStmt(node)
	case *parser.SwitchStmt:
		return c.compileSwitchStmt(node)
	case *parser.BranchStmt:
		if node.Token == token.Break {
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "break not allowed outside loop")
			}
			pos := c.emit(node, parser.OpJump, 0)
			curLoop.Breaks = append(curLoop.Breaks, pos)
		} else if node.Token == token.Continue {
			curLoop := c.currentLoop()
			if curLoop == nil {
				return c.errorf(node, "continue not allowed outside loop")
			}
			pos := c.emit(node, parser.OpJump, 0)
			curLoop.Continues = append(curLoop.Continues, pos)
		} else if node.Token == token.Fallthrough {
			sw := c.currentSwitch()
			if sw == nil {
				return c.errorf(node, "fallthrough not allowed outside switch")
			}
			if sw.InLastClause {
				return c.errorf(node, "cannot fallthrough final case in switch")
			}
			pos := c.emit(node, parser.OpJump, 0)
			sw.Fallthroughs = append(sw.Fallthroughs, pos)
		} else {
			panic(fmt.Errorf("invalid branch statement: %s",
				node.Token.String()))
		}
	case *parser.BlockStmt:
		if len(node.Stmts) == 0 {
			return nil
		}

		c.symbolTable = c.symbolTable.Fork(true)
		defer func() {
			c.symbolTable = c.symbolTable.Parent(false)
		}()

		for _, stmt := range node.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	case *parser.AssignStmt:
		err := c.compileAssign(node, node.LHS, node.RHS, node.Token)
		if err != nil {
			return err
		}
	case *parser.EmbedStmt:
		return c.compileEmbed(node)
	case *parser.NativeStmt:
		return c.compileNative(node)
	case *parser.TypeStmt:
		return c.compileType(node)
	case *parser.Ident:
		symbol, _, ok := c.symbolTable.Resolve(node.Name, false)
		if !ok {
			return c.errorf(node, "unresolved reference '%s'", node.Name)
		}

		switch symbol.Scope {
		case ScopeGlobal:
			c.emit(node, parser.OpGetGlobal, symbol.Index)
		case ScopeLocal:
			c.emit(node, parser.OpGetLocal, symbol.Index)
		case ScopeBuiltin:
			c.emit(node, parser.OpGetBuiltin, symbol.Index)
		case ScopeFree:
			c.emit(node, parser.OpGetFree, symbol.Index)
		}
	case *parser.ArrayLit:
		for _, elem := range node.Elements {
			if err := c.Compile(elem); err != nil {
				return err
			}
		}
		c.emit(node, parser.OpArray, len(node.Elements))
	case *parser.MapLit:
		for _, elt := range node.Elements {
			// key
			if len(elt.Key) > DefaultConfig.MaxStringLen {
				return c.error(node, ErrStringLimit)
			}
			c.emit(node, parser.OpConstant,
				c.addConstant(&String{Value: elt.Key}))

			// value
			if err := c.Compile(elt.Value); err != nil {
				return err
			}
		}
		c.emit(node, parser.OpMap, len(node.Elements)*2)

	case *parser.SelectorExpr: // selector on RHS side
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		if err := c.Compile(node.Sel); err != nil {
			return err
		}
		c.emit(node, parser.OpIndex)
	case *parser.IndexExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		if err := c.Compile(node.Index); err != nil {
			return err
		}
		c.emit(node, parser.OpIndex)
	case *parser.SliceExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		if node.Low != nil {
			if err := c.Compile(node.Low); err != nil {
				return err
			}
		} else {
			c.emit(node, parser.OpNull)
		}
		if node.High != nil {
			if err := c.Compile(node.High); err != nil {
				return err
			}
		} else {
			c.emit(node, parser.OpNull)
		}
		c.emit(node, parser.OpSliceIndex)
	case *parser.FuncLit:
		c.enterScope()

		for _, p := range node.Type.Params.List {
			s := c.symbolTable.Define(p.Name)

			// function arguments is not assigned directly.
			s.LocalAssigned = true
		}

		if err := c.Compile(node.Body); err != nil {
			return err
		}

		// code optimization
		c.optimizeFunc(node)

		// Issue 5.2: warn when a defer in a function suppresses tail-call
		// optimisation.  The VM only performs TCO when there are no pending
		// defers (vm/vm.go: `len(v.curFrame.defers) == 0`), so a function
		// that both uses defer and contains a tail-call pattern will silently
		// consume stack frames on every recursive call, eventually causing a
		// stack-overflow.  We cannot determine at compile time whether the
		// call is self-recursive, so we conservatively warn whenever defer
		// is combined with any tail-call-shaped instruction sequence.
		if c.scopes[c.scopeIndex].hasDefer &&
			scopeHasTailCallPattern(c.scopes[c.scopeIndex].Instructions) {
			c.addWarning(node,
				"function uses 'defer' which disables tail-call optimisation; "+
					"deep recursion inside this function may cause a stack overflow")
		}

		freeSymbols := c.symbolTable.FreeSymbols()
		numLocals := c.symbolTable.MaxSymbols()
		instructions, sourceMap := c.leaveScope()

		for _, s := range freeSymbols {
			switch s.Scope {
			case ScopeLocal:
				if !s.LocalAssigned {
					// Here, the closure is capturing a local variable that's
					// not yet assigned its value. One example is a local
					// recursive function:
					//
					//   func() {
					//     foo := func(x) {
					//       // ..
					//       return foo(x-1)
					//     }
					//   }
					//
					// which translate into
					//
					//   0000 GETL    0
					//   0002 CLOSURE ?     1
					//   0006 DEFL    0
					//
					// . So the local variable (0) is being captured before
					// it's assigned the value.
					//
					// Solution is to transform the code into something like
					// this:
					//
					//   func() {
					//     foo := undefined
					//     foo = func(x) {
					//       // ..
					//       return foo(x-1)
					//     }
					//   }
					//
					// that is equivalent to
					//
					//   0000 NULL
					//   0001 DEFL    0
					//   0003 GETL    0
					//   0005 CLOSURE ?     1
					//   0009 SETL    0
					//
					c.emit(node, parser.OpNull)
					c.emit(node, parser.OpDefineLocal, s.Index)
					s.LocalAssigned = true
				}
				c.emit(node, parser.OpGetLocalPtr, s.Index)
			case ScopeFree:
				c.emit(node, parser.OpGetFreePtr, s.Index)
			}
		}

		compiledFunction := &CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Type.Params.List),
			VarArgs:       node.Type.Params.VarArgs,
			SourceMap:     sourceMap,
		}
		if len(freeSymbols) > 0 {
			c.emit(node, parser.OpClosure,
				c.addConstant(compiledFunction), len(freeSymbols))
		} else {
			c.emit(node, parser.OpConstant, c.addConstant(compiledFunction))
		}
	case *parser.ReturnStmt:
		if c.symbolTable.Parent(true) == nil {
			// outside the function
			return c.errorf(node, "return not allowed outside function")
		}

		if node.Result == nil {
			c.emit(node, parser.OpReturn, 0)
		} else {
			if err := c.Compile(node.Result); err != nil {
				return err
			}
			c.emit(node, parser.OpReturn, 1)
		}
	case *parser.CallExpr:
		if err := c.Compile(node.Func); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Ellipsis.IsValid() {
			ellipsis = 1
		}
		c.emit(node, parser.OpCall, len(node.Args), ellipsis)
	case *parser.DeferStmt:
		if c.symbolTable.Parent(true) == nil {
			return c.errorf(node, "defer not allowed outside function")
		}
		// Mark the current scope so we can warn about suppressed tail-call
		// optimisation later (see FuncLit compilation below).
		c.scopes[c.scopeIndex].hasDefer = true
		if err := c.Compile(node.Call.Func); err != nil {
			return err
		}
		for _, arg := range node.Call.Args {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Call.Ellipsis.IsValid() {
			ellipsis = 1
		}
		c.emit(node, parser.OpDefer, len(node.Call.Args), ellipsis)
	case *parser.GoExpr:
		if err := c.Compile(node.Call.Func); err != nil {
			return err
		}
		for _, arg := range node.Call.Args {
			if err := c.Compile(arg); err != nil {
				return err
			}
		}
		ellipsis := 0
		if node.Call.Ellipsis.IsValid() {
			ellipsis = 1
		}
		c.emit(node, parser.OpRoutine, len(node.Call.Args), ellipsis)
	case *parser.ImportExpr:
		if node.ModuleName == "" {
			return c.errorf(node, "empty module name")
		}

		if mod := c.modules.Get(node.ModuleName); mod != nil {
			v, err := mod.Import(node.ModuleName)
			if err != nil {
				return err
			}

			switch v := v.(type) {
			case []byte: // module written in rumo
				compiled, err := c.compileModule(node,
					node.ModuleName, v, false)
				if err != nil {
					return err
				}
				c.emit(node, parser.OpConstant, c.addConstant(compiled))
				c.emit(node, parser.OpCall, 0, 0)
			case Object: // builtin module
				c.emit(node, parser.OpConstant, c.addConstant(v))
			default:
				panic(fmt.Errorf("invalid import value type: %T", v))
			}
		} else if c.allowFileImport {
			moduleName := node.ModuleName
			if filepath.IsAbs(moduleName) {
				return c.errorf(node, "absolute file imports are not allowed: %s", node.ModuleName)
			}
			if !strings.HasSuffix(moduleName, ".rumo") {
				moduleName += ".rumo"
			}

			modulePath, err := filepath.Abs(
				filepath.Join(c.importDir, moduleName))
			if err != nil {
				return c.errorf(node, "module file path error: %s",
					err.Error())
			}
			if c.importBase != "" {
				importAllowed, err := isPathWithinBase(c.importBase, modulePath)
				if err != nil {
					return c.errorf(node, "module file path error: %s", err.Error())
				}
				if !importAllowed {
					return c.errorf(node, "module file path escapes import root: %s", node.ModuleName)
				}
			}

			moduleSrc, err := c.fsReadFile(modulePath)
			if err != nil {
				return c.errorf(node, "module file read error: %s",
					err.Error())
			}

			compiled, err := c.compileModule(node, modulePath, moduleSrc, true)
			if err != nil {
				return err
			}
			c.emit(node, parser.OpConstant, c.addConstant(compiled))
			c.emit(node, parser.OpCall, 0, 0)
		} else {
			return c.errorf(node, "module '%s' not found", node.ModuleName)
		}
	case *parser.ExportStmt:
		// export statement must be in top-level scope
		if c.scopeIndex != 0 {
			return c.errorf(node, "export not allowed inside function")
		}

		// export statement is simply ignore when compiling non-module code
		if c.parent == nil {
			break
		}
		if err := c.Compile(node.Result); err != nil {
			return err
		}
		c.emit(node, parser.OpImmutable)
		c.emit(node, parser.OpReturn, 1)
	case *parser.ErrorExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(node, parser.OpError)
	case *parser.ImmutableExpr:
		if err := c.Compile(node.Expr); err != nil {
			return err
		}
		c.emit(node, parser.OpImmutable)
	case *parser.CondExpr:
		if err := c.Compile(node.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1 := c.emit(node, parser.OpJumpFalsy, 0)
		if err := c.Compile(node.True); err != nil {
			return err
		}

		// second jump placeholder
		jumpPos2 := c.emit(node, parser.OpJump, 0)

		// update first jump offset
		curPos := len(c.currentInstructions())
		c.changeOperand(jumpPos1, curPos)
		if err := c.Compile(node.False); err != nil {
			return err
		}

		// update second jump offset
		curPos = len(c.currentInstructions())
		c.changeOperand(jumpPos2, curPos)
	}
	return nil
}

// Bytecode returns a compiled bytecode.
func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		FileSet: c.file.Set(),
		MainFunction: &CompiledFunction{
			Instructions: append(c.currentInstructions(), parser.OpSuspend),
			SourceMap:    c.currentSourceMap(),
		},
		Constants: c.constants,
		Embeds:    c.embeds,
	}
}

// EnableFileImport enables or disables module loading from local files.
// Local file modules are disabled by default.
func (c *Compiler) EnableFileImport(enable bool) {
	c.allowFileImport = enable
}

// SetImportDir sets the initial import directory path for file imports.
// When no FS has been provided via SetImportFS, a default os.DirFS(dir) is
// created automatically so that all file reads are routed through the fs.FS
// abstraction.
func (c *Compiler) SetImportDir(dir string) {
	c.importDir = dir
	if c.importBase == "" {
		c.importBase = dir
		if c.importFS == nil {
			c.importFS = os.DirFS(dir)
		}
	} else if c.importEntryDir == "" {
		// Second call (importBase already set): record the entrypoint's directory
		// so that display paths in the FileSet are relative to where the script lives.
		c.importEntryDir = dir
	}
}

// SetImportFS sets a custom fs.FS implementation used to resolve import and
// embed paths.  The FS should be rooted at the same directory as importBase so
// that security containment checks (isPathWithinBase) still apply correctly.
// Call SetImportDir first so that importBase is established, then call
// SetImportFS to override the default os.DirFS with a virtualised filesystem.
// This enables sandboxed compilation (e.g. in WASM / wasip1 environments) where
// os.ReadFile is unavailable or restricted.
func (c *Compiler) SetImportFS(fsys fs.FS) {
	c.importFS = fsys
}

// fsReadFile reads the file at absPath. When importFS is configured, the path
// is translated to a path relative to importBase and read via the FS; otherwise
// os.ReadFile is used directly.
func (c *Compiler) fsReadFile(absPath string) ([]byte, error) {
	if c.importFS != nil && c.importBase != "" {
		rel, err := filepath.Rel(c.importBase, absPath)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		return fs.ReadFile(c.importFS, rel)
	}
	return os.ReadFile(absPath)
}

// fsGlob expands a glob pattern rooted at importDir. When importFS is
// configured the glob is evaluated against the FS and the matches are returned
// as absolute paths (importBase + rel match) so that callers need not change
// their path-handling logic; otherwise filepath.Glob is used directly.
func (c *Compiler) fsGlob(absPattern string) ([]string, error) {
	if c.importFS != nil && c.importBase != "" {
		relPattern, err := filepath.Rel(c.importBase, absPattern)
		if err != nil {
			return nil, err
		}
		relPattern = filepath.ToSlash(relPattern)
		relMatches, err := fs.Glob(c.importFS, relPattern)
		if err != nil {
			return nil, err
		}
		absMatches := make([]string, len(relMatches))
		for i, m := range relMatches {
			absMatches[i] = filepath.Join(c.importBase, filepath.FromSlash(m))
		}
		return absMatches, nil
	}
	return filepath.Glob(absPattern)
}

func (c *Compiler) compileAssign(
	node parser.Node,
	lhs, rhs []parser.Expr,
	op token.Token,
) error {
	numLHS, numRHS := len(lhs), len(rhs)
	if numLHS > 1 || numRHS > 1 {
		return c.errorf(node, "tuple assignment not allowed")
	}

	// resolve and compile left-hand side
	ident, selectors := resolveAssignLHS(lhs[0])
	numSel := len(selectors)

	if op == token.Define && numSel > 0 {
		// using selector on new variable does not make sense
		return c.errorf(node, "operator ':=' not allowed with selector")
	}

	symbol, depth, exists := c.symbolTable.Resolve(ident, false)
	if op == token.Define {
		if depth == 0 && exists {
			return c.errorf(node, "'%s' redeclared in this block", ident)
		}
		symbol = c.symbolTable.Define(ident)
	} else {
		if !exists {
			return c.errorf(node, "unresolved reference '%s'", ident)
		}
	}

	// +=, -=, *=, /=
	if op != token.Assign && op != token.Define {
		if err := c.Compile(lhs[0]); err != nil {
			return err
		}
	}

	// compile RHSs
	for _, expr := range rhs {
		if err := c.Compile(expr); err != nil {
			return err
		}
	}

	switch op {
	case token.AddAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Add))
	case token.SubAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Sub))
	case token.MulAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Mul))
	case token.QuoAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Quo))
	case token.RemAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Rem))
	case token.AndAssign:
		c.emit(node, parser.OpBinaryOp, int(token.And))
	case token.OrAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Or))
	case token.AndNotAssign:
		c.emit(node, parser.OpBinaryOp, int(token.AndNot))
	case token.XorAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Xor))
	case token.ShlAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Shl))
	case token.ShrAssign:
		c.emit(node, parser.OpBinaryOp, int(token.Shr))
	}

	// compile selector expressions (right to left)
	for i := numSel - 1; i >= 0; i-- {
		if err := c.Compile(selectors[i]); err != nil {
			return err
		}
	}

	switch symbol.Scope {
	case ScopeGlobal:
		if numSel > 0 {
			c.emit(node, parser.OpSetSelGlobal, symbol.Index, numSel)
		} else {
			c.emit(node, parser.OpSetGlobal, symbol.Index)
		}
	case ScopeLocal:
		if numSel > 0 {
			c.emit(node, parser.OpSetSelLocal, symbol.Index, numSel)
		} else {
			if op == token.Define && !symbol.LocalAssigned {
				c.emit(node, parser.OpDefineLocal, symbol.Index)
			} else {
				c.emit(node, parser.OpSetLocal, symbol.Index)
			}
		}

		// mark the symbol as local-assigned
		symbol.LocalAssigned = true
	case ScopeFree:
		if numSel > 0 {
			c.emit(node, parser.OpSetSelFree, symbol.Index, numSel)
		} else {
			c.emit(node, parser.OpSetFree, symbol.Index)
		}
	default:
		panic(fmt.Errorf("invalid assignment variable scope: %s",
			symbol.Scope))
	}
	return nil
}

// compileEmbed handles //embed directive statements. It resolves glob patterns,
// reads file contents at compile time, and emits a constant for the result.
// Supported placeholder types on the RHS of the := statement:
//
//	""          → single file embedded as string
//	bytes("")   → single file embedded as bytes
//	{}          → multiple files embedded as map[string]string
//	bytes({})   → multiple files embedded as map[string]bytes
func (c *Compiler) compileEmbed(node *parser.EmbedStmt) error {
	if c.importDir == "" {
		return c.errorf(node, "embed: file embed is not available (no source directory)")
	}
	if len(node.Patterns) == 0 {
		return c.errorf(node, "embed: no patterns specified")
	}
	if len(node.Assign.LHS) != 1 {
		return c.errorf(node, "embed: exactly one variable must be on the left-hand side")
	}
	if len(node.Assign.RHS) != 1 {
		return c.errorf(node, "embed: exactly one expression must be on the right-hand side")
	}

	// Determine desired output type from the RHS placeholder expression.
	type embedKind int
	const (
		embedString      embedKind = iota // single file as string
		embedBytes                        // single file as bytes
		embedMapString                    // multi-file as map[string]string
		embedMapBytes                     // multi-file as map[string]bytes
	)

	kind := embedString
	rhs := node.Assign.RHS[0]
	switch rhs := rhs.(type) {
	case *parser.StringLit:
		// x := ""
		kind = embedString
	case *parser.MapLit:
		// x := {}
		if len(rhs.Elements) != 0 {
			return c.errorf(node, "embed: map placeholder must be empty ({})")
		}
		kind = embedMapString
	case *parser.CallExpr:
		// x := bytes("") or x := bytes({})
		fn, ok := rhs.Func.(*parser.Ident)
		if !ok || fn.Name != "bytes" {
			return c.errorf(node, "embed: unsupported placeholder expression; use \"\", bytes(\"\"), {}, or bytes({})")
		}
		if len(rhs.Args) != 1 {
			return c.errorf(node, "embed: bytes() placeholder must have exactly one argument")
		}
		switch rhs.Args[0].(type) {
		case *parser.StringLit:
			kind = embedBytes
		case *parser.MapLit:
			kind = embedMapBytes
		default:
			return c.errorf(node, "embed: bytes() placeholder argument must be \"\" or {}")
		}
	default:
		return c.errorf(node, "embed: unsupported placeholder expression; use \"\", bytes(\"\"), {}, or bytes({})")
	}

	// Resolve glob patterns.
	var matchedPaths []string
	for _, pattern := range node.Patterns {
		var globPattern string
		if filepath.IsAbs(pattern) {
			return c.errorf(node, "embed: absolute paths are not allowed: %s", pattern)
		}
		globPattern = filepath.Join(c.importDir, filepath.FromSlash(pattern))
		matches, err := c.fsGlob(globPattern)
		if err != nil {
			return c.errorf(node, "embed: invalid glob pattern %q: %s", pattern, err.Error())
		}
		if len(matches) == 0 {
			return c.errorf(node, "embed: no files matched pattern %q", pattern)
		}
		matchedPaths = append(matchedPaths, matches...)
	}

	// Security: verify every matched embed path is within importBase so that
	// patterns like "../../etc/passwd" cannot read files outside the root.
	if c.importBase != "" {
		for _, path := range matchedPaths {
			allowed, err := isPathWithinBase(c.importBase, path)
			if err != nil {
				return c.errorf(node, "embed: file path error: %s", err.Error())
			}
			if !allowed {
				return c.errorf(node, "embed: file path escapes import root: %s", path)
			}
		}
	}

	// For single-file embeds require exactly one match; for multi-file embeds
	// build a Map. In both cases record each file in c.embeds.
	if kind == embedString || kind == embedBytes {
		if len(matchedPaths) != 1 {
			return c.errorf(node, "embed: single-file embed matched %d files (expected 1)", len(matchedPaths))
		}
		data, err := c.fsReadFile(matchedPaths[0])
		if err != nil {
			return c.errorf(node, "embed: failed to read file %q: %s", matchedPaths[0], err.Error())
		}
		var obj Object
		if kind == embedString {
			if len(data) > DefaultConfig.MaxStringLen {
				return c.error(node, ErrStringLimit)
			}
			obj = &String{Value: string(data)}
		} else {
			if len(data) > DefaultConfig.MaxBytesLen {
				return c.error(node, ErrBytesLimit)
			}
			obj = &Bytes{Value: data}
		}
		// Record embed metadata.
		rel, _ := filepath.Rel(c.importDir, matchedPaths[0])
		rel = filepath.ToSlash(rel)
		c.embeds = append(c.embeds, EmbedFile{Name: rel, Size: len(data)})
		c.emit(node, parser.OpConstant, c.addConstant(obj))
	} else {
		// Multi-file embed: build a Map constant.
		mapObj := &Map{Value: make(map[string]Object, len(matchedPaths))}
		for _, path := range matchedPaths {
			data, err := c.fsReadFile(path)
			if err != nil {
				return c.errorf(node, "embed: failed to read file %q: %s", path, err.Error())
			}
			// Use the path relative to importDir as the map key.
			rel, err := filepath.Rel(c.importDir, path)
			if err != nil {
				rel = path
			}
			rel = filepath.ToSlash(rel) // normalize to forward slashes
			var valObj Object
			if kind == embedMapString {
				if len(data) > DefaultConfig.MaxStringLen {
					return c.error(node, ErrStringLimit)
				}
				valObj = &String{Value: string(data)}
			} else {
				if len(data) > DefaultConfig.MaxBytesLen {
					return c.error(node, ErrBytesLimit)
				}
				valObj = &Bytes{Value: data}
			}
			mapObj.Value[rel] = valObj
			// Record embed metadata.
			c.embeds = append(c.embeds, EmbedFile{Name: rel, Size: len(data)})
		}
		c.emit(node, parser.OpConstant, c.addConstant(mapObj))
	}

	// Now define and assign the LHS variable (reuse the logic from compileAssign).
	ident, _ := resolveAssignLHS(node.Assign.LHS[0])
	symbol, depth, exists := c.symbolTable.Resolve(ident, false)
	if depth == 0 && exists {
		return c.errorf(node, "'%s' redeclared in this block", ident)
	}
	symbol = c.symbolTable.Define(ident)
	switch symbol.Scope {
	case ScopeGlobal:
		c.emit(node, parser.OpSetGlobal, symbol.Index)
	case ScopeLocal:
		c.emit(node, parser.OpDefineLocal, symbol.Index)
		symbol.LocalAssigned = true
	default:
		panic(fmt.Errorf("embed: unexpected symbol scope: %s", symbol.Scope))
	}
	return nil
}


func (c *Compiler) compileLogical(node *parser.BinaryExpr) error {
	// left side term
	if err := c.Compile(node.LHS); err != nil {
		return err
	}

	// jump position
	var jumpPos int
	if node.Token == token.LAnd {
		jumpPos = c.emit(node, parser.OpAndJump, 0)
	} else {
		jumpPos = c.emit(node, parser.OpOrJump, 0)
	}

	// right side term
	if err := c.Compile(node.RHS); err != nil {
		return err
	}

	c.changeOperand(jumpPos, len(c.currentInstructions()))
	return nil
}

func (c *Compiler) compileForStmt(stmt *parser.ForStmt) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// init statement
	if stmt.Init != nil {
		if err := c.Compile(stmt.Init); err != nil {
			return err
		}
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition expression
	postCondPos := -1
	if stmt.Cond != nil {
		if err := c.Compile(stmt.Cond); err != nil {
			return err
		}
		// condition jump position
		postCondPos = c.emit(stmt, parser.OpJumpFalsy, 0)
	}

	// enter loop
	loop := c.enterLoop()

	// body statement
	if err := c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())

	// post statement
	if stmt.Post != nil {
		if err := c.Compile(stmt.Post); err != nil {
			return err
		}
	}

	// back to condition
	c.emit(stmt, parser.OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	if postCondPos >= 0 {
		c.changeOperand(postCondPos, postStmtPos)
	}

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		c.changeOperand(pos, postStmtPos)
	}
	for _, pos := range loop.Continues {
		c.changeOperand(pos, postBodyPos)
	}
	return nil
}

func (c *Compiler) compileForInStmt(stmt *parser.ForInStmt) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// for-in statement is compiled like following:
	//
	//   for :it := iterator(iterable); :it.next();  {
	//     k, v := :it.get()  // DEFINE operator
	//
	//     ... body ...
	//   }
	//
	// ":it" is a local variable but it will not conflict with other user variables
	// because character ":" is not allowed in the variable names.

	// init
	//   :it = iterator(iterable)
	itSymbol := c.symbolTable.Define(":it")
	if err := c.Compile(stmt.Iterable); err != nil {
		return err
	}
	c.emit(stmt, parser.OpIteratorInit)
	if itSymbol.Scope == ScopeGlobal {
		c.emit(stmt, parser.OpSetGlobal, itSymbol.Index)
	} else {
		c.emit(stmt, parser.OpDefineLocal, itSymbol.Index)
	}

	// pre-condition position
	preCondPos := len(c.currentInstructions())

	// condition
	//  :it.HasMore()
	if itSymbol.Scope == ScopeGlobal {
		c.emit(stmt, parser.OpGetGlobal, itSymbol.Index)
	} else {
		c.emit(stmt, parser.OpGetLocal, itSymbol.Index)
	}
	c.emit(stmt, parser.OpIteratorNext)

	// condition jump position
	postCondPos := c.emit(stmt, parser.OpJumpFalsy, 0)

	// enter loop
	loop := c.enterLoop()

	// assign key variable
	if stmt.Key.Name != "_" {
		keySymbol := c.symbolTable.Define(stmt.Key.Name)
		if itSymbol.Scope == ScopeGlobal {
			c.emit(stmt, parser.OpGetGlobal, itSymbol.Index)
		} else {
			c.emit(stmt, parser.OpGetLocal, itSymbol.Index)
		}
		c.emit(stmt, parser.OpIteratorKey)
		if keySymbol.Scope == ScopeGlobal {
			c.emit(stmt, parser.OpSetGlobal, keySymbol.Index)
		} else {
			keySymbol.LocalAssigned = true
			c.emit(stmt, parser.OpDefineLocal, keySymbol.Index)
		}
	}

	// assign value variable
	if stmt.Value.Name != "_" {
		valueSymbol := c.symbolTable.Define(stmt.Value.Name)
		if itSymbol.Scope == ScopeGlobal {
			c.emit(stmt, parser.OpGetGlobal, itSymbol.Index)
		} else {
			c.emit(stmt, parser.OpGetLocal, itSymbol.Index)
		}
		c.emit(stmt, parser.OpIteratorValue)
		if valueSymbol.Scope == ScopeGlobal {
			c.emit(stmt, parser.OpSetGlobal, valueSymbol.Index)
		} else {
			valueSymbol.LocalAssigned = true
			c.emit(stmt, parser.OpDefineLocal, valueSymbol.Index)
		}
	}

	// body statement
	if err := c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.currentInstructions())

	// back to condition
	c.emit(stmt, parser.OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.currentInstructions())
	c.changeOperand(postCondPos, postStmtPos)

	// update all break/continue jump positions
	for _, pos := range loop.Breaks {
		c.changeOperand(pos, postStmtPos)
	}
	for _, pos := range loop.Continues {
		c.changeOperand(pos, postBodyPos)
	}
	return nil
}

func (c *Compiler) checkCyclicImports(node parser.Node, modulePath string) error {
	if c.modulePath == modulePath {
		return c.errorf(node, "cyclic module import: %s", modulePath)
	} else if c.parent != nil {
		return c.parent.checkCyclicImports(node, modulePath)
	}
	return nil
}

func (c *Compiler) compileModule(
	node parser.Node,
	modulePath string,
	src []byte,
	isFile bool,
) (*CompiledFunction, error) {
	if err := c.checkCyclicImports(node, modulePath); err != nil {
		return nil, err
	}

	compiledModule, exists := c.loadCompiledModule(modulePath)
	if exists {
		return compiledModule, nil
	}

	// Use a display name relative to the entrypoint directory (importEntryDir) for
	// portable, user-facing paths in the FileSet.  Fall back to importBase when
	// importEntryDir has not been set (e.g. standalone Compiler usage without Script).
	// We allow ".." in display paths (e.g. "../cli/one.rumo") because the security
	// containment check above already verified the path is within importBase; the
	// display name only needs to avoid leaking absolute OS paths.
	displayPath := modulePath
	displayBase := c.importEntryDir
	if displayBase == "" {
		displayBase = c.importBase
	}
	if displayBase != "" {
		if rel, err := filepath.Rel(displayBase, modulePath); err == nil {
			displayPath = rel
		}
	}

	modFile := c.file.Set().AddFile(displayPath, -1, len(src))
	p := parser.NewParser(modFile, src, nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	// inherit builtin functions
	symbolTable := NewSymbolTable()
	for _, sym := range c.symbolTable.BuiltinSymbols() {
		symbolTable.DefineBuiltin(sym.Index, sym.Name)
	}

	// no global scope for the module
	symbolTable = symbolTable.Fork(false)

	// compile module
	moduleCompiler := c.fork(modFile, modulePath, symbolTable, isFile)
	if err := moduleCompiler.Compile(file); err != nil {
		return nil, err
	}

	// code optimization
	moduleCompiler.optimizeFunc(node)
	compiledFunc := moduleCompiler.Bytecode().MainFunction
	compiledFunc.NumLocals = symbolTable.MaxSymbols()
	c.storeCompiledModule(modulePath, compiledFunc)
	return compiledFunc, nil
}

func (c *Compiler) loadCompiledModule(modulePath string) (mod *CompiledFunction, ok bool) {
	if c.parent != nil {
		return c.parent.loadCompiledModule(modulePath)
	}
	mod, ok = c.compiledModules[modulePath]
	return
}

func (c *Compiler) storeCompiledModule(modulePath string, module *CompiledFunction) {
	if c.parent != nil {
		c.parent.storeCompiledModule(modulePath, module)
	}
	c.compiledModules[modulePath] = module
}

func (c *Compiler) enterLoop() *loop {
	loop := &loop{}
	c.loops = append(c.loops, loop)
	c.loopIndex++
	if c.trace != nil {
		c.printTrace("LOOPE", c.loopIndex)
	}
	return loop
}

func (c *Compiler) leaveLoop() {
	if c.trace != nil {
		c.printTrace("LOOPL", c.loopIndex)
	}
	c.loops = c.loops[:len(c.loops)-1]
	c.loopIndex--
}

func (c *Compiler) currentLoop() *loop {
	if c.loopIndex >= 0 {
		return c.loops[c.loopIndex]
	}
	return nil
}

func (c *Compiler) enterSwitch() *switchCtx {
	sw := &switchCtx{}
	c.switches = append(c.switches, sw)
	return sw
}

func (c *Compiler) leaveSwitch() {
	c.switches = c.switches[:len(c.switches)-1]
}

func (c *Compiler) currentSwitch() *switchCtx {
	if n := len(c.switches); n > 0 {
		return c.switches[n-1]
	}
	return nil
}

// compileSwitchStmt compiles a switch statement using only existing opcodes
// (OpEqual, OpJumpFalsy, OpJump). Each case is auto-break (no implicit
// fallthrough); explicit `fallthrough` jumps to the next clause's body, and
// `break` exits the switch (handled via the loop break stack).
func (c *Compiler) compileSwitchStmt(node *parser.SwitchStmt) error {
	// Outer scope for Init.
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	if node.Init != nil {
		if err := c.Compile(node.Init); err != nil {
			return err
		}
	}

	hasTag := node.Tag != nil

	// Separate clauses into cases and an optional default.
	clauses := node.Body.Stmts
	var defaultClause *parser.CaseClause
	defaultIdx := -1
	for i, st := range clauses {
		cc, ok := st.(*parser.CaseClause)
		if !ok {
			return c.errorf(st, "non-case statement in switch body")
		}
		if cc.List == nil {
			if defaultClause != nil {
				return c.errorf(cc, "multiple default clauses in switch")
			}
			defaultClause = cc
			defaultIdx = i
		}
	}

	// Use the existing loop stack so `break` inside switch works without any
	// VM/compiler changes. We reject `continue` collected on this frame.
	br := c.enterLoop()
	sw := c.enterSwitch()

	// Build an ordered list of non-default cases (preserving textual order)
	// plus the default at the end.
	type caseInfo struct {
		clause       *parser.CaseClause
		matchJumps   []int // OpJumpFalsy slots that jump to next case match
		bodyStart    int
		fallthroughs []int // jump slots emitted via `fallthrough` in this body
	}
	var cases []*caseInfo
	for i, st := range clauses {
		if i == defaultIdx {
			continue
		}
		cases = append(cases, &caseInfo{clause: st.(*parser.CaseClause)})
	}

	// caseEndJumps collects unconditional jumps to the switch end emitted
	// after each (non-fallthrough) case body finishes.
	var caseEndJumps []int

	// Compile case-match preambles + bodies sequentially. The match preamble
	// for case i ends with an OpJumpFalsy whose target is patched to the
	// match-start of case i+1 (or default-start, or switch end).
	for idx, ci := range cases {
		// Patch previous case's "no match" jumps to point here.
		if idx > 0 {
			prev := cases[idx-1]
			here := len(c.currentInstructions())
			for _, jp := range prev.matchJumps {
				c.changeOperand(jp, here)
			}
		}

		// Match preamble: emit comparisons.
		// Each comparison pushes a bool. We use OpJumpFalsy (which pops the
		// value on both paths) to skip to the next comparison on a miss, and
		// an unconditional OpJump to the body on a hit. The final
		// comparison's OpJumpFalsy targets the next case's match start
		// (recorded in matchJumps).
		nExprs := len(ci.clause.List)
		var trueJumps []int // unconditional jumps to body when match succeeds
		for k, expr := range ci.clause.List {
			if hasTag {
				if err := c.Compile(node.Tag); err != nil {
					c.leaveSwitch()
					c.leaveLoop()
					return err
				}
				if err := c.Compile(expr); err != nil {
					c.leaveSwitch()
					c.leaveLoop()
					return err
				}
				c.emit(expr, parser.OpEqual)
			} else {
				if err := c.Compile(expr); err != nil {
					c.leaveSwitch()
					c.leaveLoop()
					return err
				}
			}
			if k < nExprs-1 {
				// On miss, skip to next comparison; on hit, jump to body.
				nextCmpJ := c.emit(expr, parser.OpJumpFalsy, 0)
				bodyJ := c.emit(expr, parser.OpJump, 0)
				trueJumps = append(trueJumps, bodyJ)
				// Patch the OpJumpFalsy to land on the next iteration (i.e.
				// after the OpJump we just emitted).
				c.changeOperand(nextCmpJ, len(c.currentInstructions()))
			}
		}
		// Final comparison: if false, jump to next case match start.
		mj := c.emit(ci.clause, parser.OpJumpFalsy, 0)
		ci.matchJumps = append(ci.matchJumps, mj)

		// Patch all "match-true" short-circuit jumps to land here (body start).
		bodyStart := len(c.currentInstructions())
		ci.bodyStart = bodyStart
		for _, jp := range trueJumps {
			c.changeOperand(jp, bodyStart)
		}

		// Compile body in a fresh lexical scope.
		c.symbolTable = c.symbolTable.Fork(true)
		sw.InLastClause = (idx == len(cases)-1) && defaultClause == nil
		// Save and reset per-clause fallthroughs so they only collect from
		// statements compiled in *this* clause's body.
		sw.Fallthroughs = nil
		for _, st := range ci.clause.Body {
			if err := c.Compile(st); err != nil {
				c.symbolTable = c.symbolTable.Parent(false)
				c.leaveSwitch()
				c.leaveLoop()
				return err
			}
		}
		ci.fallthroughs = sw.Fallthroughs
		sw.Fallthroughs = nil
		c.symbolTable = c.symbolTable.Parent(false)

		// After body: emit jump-to-end (unless last stmt was fallthrough,
		// indicated by ci.fallthroughs being non-empty AND being the very
		// last instruction; we conservatively always emit a jump-to-end and
		// also patch fallthroughs separately to next clause body start).
		endJ := c.emit(ci.clause, parser.OpJump, 0)
		caseEndJumps = append(caseEndJumps, endJ)
	}

	// After all non-default cases, patch the *last* case's matchJumps so
	// that on no-match we either fall to the default body or to switch end.
	noMatchPos := len(c.currentInstructions())
	if len(cases) > 0 {
		last := cases[len(cases)-1]
		for _, jp := range last.matchJumps {
			c.changeOperand(jp, noMatchPos)
		}
	}

	// Default clause body, if any.
	var defaultBodyStart int = -1
	if defaultClause != nil {
		defaultBodyStart = len(c.currentInstructions())
		c.symbolTable = c.symbolTable.Fork(true)
		sw.InLastClause = true
		sw.Fallthroughs = nil
		for _, st := range defaultClause.Body {
			if err := c.Compile(st); err != nil {
				c.symbolTable = c.symbolTable.Parent(false)
				c.leaveSwitch()
				c.leaveLoop()
				return err
			}
		}
		// fallthrough from default is illegal; sw.Fallthroughs should be
		// empty (validated when the statement is encountered).
		c.symbolTable = c.symbolTable.Parent(false)
	}

	// Switch end position.
	switchEnd := len(c.currentInstructions())

	// Patch end-of-body jumps to switchEnd.
	for _, jp := range caseEndJumps {
		c.changeOperand(jp, switchEnd)
	}

	// Patch break statements to switchEnd.
	if len(br.Continues) > 0 {
		c.leaveSwitch()
		c.leaveLoop()
		return c.errorf(node, "continue not allowed inside switch")
	}
	for _, jp := range br.Breaks {
		c.changeOperand(jp, switchEnd)
	}

	// Patch fallthrough jumps: each fallthrough goes to the next clause's
	// body start (or default body start when emitted from the last
	// non-default case).
	for idx, ci := range cases {
		if len(ci.fallthroughs) == 0 {
			continue
		}
		var target int
		if idx+1 < len(cases) {
			target = cases[idx+1].bodyStart
		} else if defaultBodyStart >= 0 {
			target = defaultBodyStart
		} else {
			// Should have been caught at compile time (InLastClause).
			c.leaveSwitch()
			c.leaveLoop()
			return c.errorf(ci.clause, "cannot fallthrough final case in switch")
		}
		for _, jp := range ci.fallthroughs {
			c.changeOperand(jp, target)
		}
	}

	c.leaveSwitch()
	c.leaveLoop()
	return nil
}



func (c *Compiler) currentInstructions() []byte {
	return c.scopes[c.scopeIndex].Instructions
}

func (c *Compiler) currentSourceMap() map[int]parser.Pos {
	return c.scopes[c.scopeIndex].SourceMap
}

func (c *Compiler) enterScope() {
	scope := compilationScope{
		SymbolInit: make(map[string]bool),
		SourceMap:  make(map[int]parser.Pos),
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = c.symbolTable.Fork(false)
	if c.trace != nil {
		c.printTrace("SCOPE", c.scopeIndex)
	}
}

func (c *Compiler) leaveScope() (instructions []byte, sourceMap map[int]parser.Pos) {
	instructions = c.currentInstructions()
	sourceMap = c.currentSourceMap()
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Parent(true)
	if c.trace != nil {
		c.printTrace("SCOPL", c.scopeIndex)
	}
	return
}

func (c *Compiler) fork(file *parser.SourceFile, modulePath string, symbolTable *SymbolTable, isFile bool) *Compiler {
	child := NewCompiler(file, symbolTable, nil, c.modules, c.trace)
	child.modulePath = modulePath // module file path
	child.parent = c              // parent to set to current compiler
	child.allowFileImport = c.allowFileImport
	child.importDir = c.importDir
	child.importBase = c.importBase
	child.importEntryDir = c.importEntryDir  // propagate entrypoint dir for consistent display paths
	child.importFS = c.importFS               // propagate virtualised filesystem to child
	if isFile && c.importDir != "" {
		child.importDir = filepath.Dir(modulePath)
	}
	return child
}

func isPathWithinBase(base, target string) (bool, error) {
	// Resolve symlinks where possible; fall back to lexical clean paths for
	// virtual FS targets that do not exist on the real filesystem.
	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		realBase = filepath.Clean(base)
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		realTarget = filepath.Clean(target)
	}
	rel, err := filepath.Rel(realBase, realTarget)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return true, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

func (c *Compiler) error(node parser.Node, err error) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    node,
		Err:     err,
	}
}

func (c *Compiler) errorf(node parser.Node, format string, args ...interface{}) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    node,
		Err:     fmt.Errorf(format, args...),
	}
}

func (c *Compiler) addConstant(o Object) int {
	if c.parent != nil {
		// module compilers will use their parent's constants array
		return c.parent.addConstant(o)
	}
	c.constants = append(c.constants, o)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("CONST %04d %s", len(c.constants)-1, o))
	}
	return len(c.constants) - 1
}

func (c *Compiler) addInstruction(b []byte) int {
	posNewIns := len(c.currentInstructions())
	c.scopes[c.scopeIndex].Instructions = append(
		c.currentInstructions(), b...)
	return posNewIns
}

func (c *Compiler) replaceInstruction(pos int, inst []byte) {
	copy(c.currentInstructions()[pos:], inst)
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("REPLC %s",
			FormatInstructions(
				c.scopes[c.scopeIndex].Instructions[pos:], pos)[0]))
	}
}

func (c *Compiler) changeOperand(opPos int, operand ...int) {
	op := c.currentInstructions()[opPos]
	inst := MakeInstruction(op, operand...)
	c.replaceInstruction(opPos, inst)
}

// optimizeFunc performs some code-level optimization for the current function
// instructions. It also removes unreachable (dead code) instructions and adds
// "returns" instruction if needed.
func (c *Compiler) optimizeFunc(node parser.Node) {
	// any instructions between RETURN and the function end
	// or instructions between RETURN and jump target position
	// are considered as unreachable.

	// pass 1. identify all jump destinations
	dsts := make(map[int]bool)
	iterateInstructions(c.scopes[c.scopeIndex].Instructions,
		func(pos int, opcode parser.Opcode, operands []int) bool {
			switch opcode {
			case parser.OpJump, parser.OpJumpFalsy,
				parser.OpAndJump, parser.OpOrJump:
				dsts[operands[0]] = true
			}
			return true
		})

	// pass 2. eliminate dead code
	var newInsts []byte
	posMap := make(map[int]int) // old position to new position
	var dstIdx int
	var deadCode bool
	iterateInstructions(c.scopes[c.scopeIndex].Instructions,
		func(pos int, opcode parser.Opcode, operands []int) bool {
			switch {
			case opcode == parser.OpReturn:
				if deadCode {
					return true
				}
				deadCode = true
			case dsts[pos]:
				dstIdx++
				deadCode = false
			case deadCode:
				return true
			}
			posMap[pos] = len(newInsts)
			newInsts = append(newInsts,
				MakeInstruction(opcode, operands...)...)
			return true
		})

	// pass 3. update jump positions
	var lastOp parser.Opcode
	var appendReturn bool
	endPos := len(c.scopes[c.scopeIndex].Instructions)
	newEndPost := len(newInsts)
	iterateInstructions(newInsts,
		func(pos int, opcode parser.Opcode, operands []int) bool {
			switch opcode {
			case parser.OpJump, parser.OpJumpFalsy, parser.OpAndJump,
				parser.OpOrJump:
				newDst, ok := posMap[operands[0]]
				if ok {
					copy(newInsts[pos:],
						MakeInstruction(opcode, newDst))
				} else if endPos == operands[0] {
					// there's a jump instruction that jumps to the end of
					// function compiler should append "return".
					copy(newInsts[pos:],
						MakeInstruction(opcode, newEndPost))
					appendReturn = true
				} else {
					panic(fmt.Errorf("invalid jump position: %d", newDst))
				}
			}
			lastOp = opcode
			return true
		})
	if lastOp != parser.OpReturn {
		appendReturn = true
	}

	// pass 4. update source map
	newSourceMap := make(map[int]parser.Pos)
	for pos, srcPos := range c.scopes[c.scopeIndex].SourceMap {
		newPos, ok := posMap[pos]
		if ok {
			newSourceMap[newPos] = srcPos
		}
	}
	c.scopes[c.scopeIndex].Instructions = newInsts
	c.scopes[c.scopeIndex].SourceMap = newSourceMap

	// append "return"
	if appendReturn {
		c.emit(node, parser.OpReturn, 0)
	}
}

func (c *Compiler) emit(node parser.Node, opcode parser.Opcode, operands ...int) int {
	filePos := parser.NoPos
	if node != nil {
		filePos = node.Pos()
	}

	inst := MakeInstruction(opcode, operands...)
	pos := c.addInstruction(inst)
	c.scopes[c.scopeIndex].SourceMap[pos] = filePos
	if c.trace != nil {
		c.printTrace(fmt.Sprintf("EMIT  %s",
			FormatInstructions(
				c.scopes[c.scopeIndex].Instructions[pos:], pos)[0]))
	}
	return pos
}

func (c *Compiler) printTrace(a ...interface{}) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	i := 2 * c.indent
	for i > n {
		_, _ = fmt.Fprint(c.trace, dots)
		i -= n
	}
	_, _ = fmt.Fprint(c.trace, dots[0:i])
	_, _ = fmt.Fprintln(c.trace, a...)
}

func resolveAssignLHS(expr parser.Expr) (name string, selectors []parser.Expr) {
	switch term := expr.(type) {
	case *parser.SelectorExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Sel)
		return
	case *parser.IndexExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Index)
	case *parser.Ident:
		name = term.Name
	}
	return
}

func iterateInstructions(b []byte, fn func(pos int, opcode parser.Opcode, operands []int) bool) {
	for i := 0; i < len(b); i++ {
		numOperands := parser.OpcodeOperands[b[i]]
		operands, read := parser.ReadOperands(numOperands, b[i+1:])
		if !fn(i, b[i], operands) {
			break
		}
		i += read
	}
}

// scopeHasTailCallPattern reports whether the instruction slice contains any
// tail-call shaped sequence:
//
//	OpCall … OpReturn
//	OpCall … OpPop OpReturn
//
// This mirrors the exact pattern checked by the VM at runtime when deciding
// whether to apply tail-call optimisation.
func scopeHasTailCallPattern(insts []byte) bool {
	found := false
	iterateInstructions(insts,
		func(pos int, opcode parser.Opcode, operands []int) bool {
			if opcode != parser.OpCall {
				return true
			}
			// Size of OpCall instruction: 1 (opcode) + sum of operand widths
			callSize := 1
			for _, w := range parser.OpcodeOperands[parser.OpCall] {
				callSize += w
			}
			next := pos + callSize
			if next >= len(insts) {
				return true
			}
			nextOp := parser.Opcode(insts[next])
			if nextOp == parser.OpReturn {
				found = true
				return false // stop early
			}
			if nextOp == parser.OpPop {
				popSize := 1
				for _, w := range parser.OpcodeOperands[parser.OpPop] {
					popSize += w
				}
				afterPop := next + popSize
				if afterPop < len(insts) && parser.Opcode(insts[afterPop]) == parser.OpReturn {
					found = true
					return false
				}
			}
			return true
		})
	return found
}

func tracec(c *Compiler, msg string) *Compiler {
	c.printTrace(msg, "{")
	c.indent++
	return c
}

func untracec(c *Compiler) {
	c.indent--
	c.printTrace("}")
}
