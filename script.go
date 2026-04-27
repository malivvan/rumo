package rumo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/fs"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/codec"

	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/malivvan/rumo/vm/parser"
)

// Magic is a magic number every encoded Program starts with.
// format: [4]MAGIC [2]VERSION [4]SIZE [N]DATA [32]SHA256(DATA)
const Magic = "RUMO"

// FormatVersion is the current bytecode format version.
// It is stored as a little-endian uint16 in bytes [4:6] of the header.
// Increment this constant whenever the on-disk format changes in an
// incompatible way so that old compiled files produce a clear error
// ("incompatible bytecode version") instead of cryptic decode failures.
//
// Version history:
//
//	1: initial format; trailer was an 8-byte CRC64/ECMA checksum.
//	2: trailer replaced with a 32-byte SHA-256 digest.
//	3: ImmutableMap encoding prepends an out-of-band module-name string so
//	   the __module_name__ namespace cannot be spoofed by user script data.
//	4: Bytecode encoding prepends a builtin name table so that OpGetBuiltin
//	   indices are resolved by name at load time.  New builtins may now be
//	   inserted anywhere in the registration list without corrupting compiled
//	   bytecode (fixes issue 5.10: Builtin index baked into bytecode).
//
// FormatVersion history:
//
//	1 – original CRC64/ECMA trailer
//	2 – SHA-256 trailer
//	3 – Ptr serialisation removed
//	4 – various encoding hardening
//	5 – Time encoding now includes timezone name (fixes silent UTC coercion)
const FormatVersion uint16 = 5

// Script can simplify compilation and execution of embedded scripts.
type Script struct {
	variables        map[string]*Variable
	modules          *vm.ModuleMap
	name             string
	input            []byte
	maxAllocs        int64
	maxConstObjects  int
	maxStringLen     int
	enableFileImport bool
	importDir        string
	importFS         fs.FS // optional virtualised FS; nil → default os.DirFS(importDir)
	permissions      vm.Permissions
}

// NewScript creates a Script instance with an input script.
//
// By default, the script runs with:
//   - deny-all Permissions (no file I/O, exec, env write, or chdir allowed);
//     call SetPermissions(vm.UnrestrictedPermissions()) to opt in.
//   - bounded resource limits (MaxAllocs, MaxStringLen, MaxBytesLen from
//     vm.DefaultConfig); call SetMaxAllocs(-1) etc. or pass vm.UnlimitedConfig()
//     to disable limits for trusted scripts.
func NewScript(input []byte) *Script {
	return &Script{
		variables:       make(map[string]*Variable),
		name:            "(main)",
		input:           input,
		maxAllocs:       0, // 0 = use DefaultConfig.MaxAllocs (bounded safe default)
		maxConstObjects: -1,
	}
}

// Add adds a new variable or updates an existing variable to the script.
func (s *Script) Add(name string, value interface{}) error {
	obj, err := vm.FromInterface(value)
	if err != nil {
		return err
	}
	s.variables[name] = &Variable{
		name:  name,
		value: obj,
	}
	return nil
}

// Remove removes (undefines) an existing variable for the script. It returns
// false if the variable name is not defined.
func (s *Script) Remove(name string) bool {
	if _, ok := s.variables[name]; !ok {
		return false
	}
	delete(s.variables, name)
	return true
}

// SetName sets the name of the script.
func (s *Script) SetName(name string) {
	s.name = name
}

// SetImports sets import modules.
func (s *Script) SetImports(modules *vm.ModuleMap) {
	s.modules = modules
}

// SetImportDir sets the initial import directory for script files.
func (s *Script) SetImportDir(dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	s.importDir = dir
	return nil
}

// SetMaxAllocs sets the maximum number of objects allocations during the run
// time. Compiled script will return ErrObjectAllocLimit error if it
// exceeds this limit.
func (s *Script) SetMaxAllocs(n int64) {
	s.maxAllocs = n
}

// SetMaxConstObjects sets the maximum number of objects in the compiled
// constants.
func (s *Script) SetMaxConstObjects(n int) {
	s.maxConstObjects = n
}

// EnableFileImport enables or disables module loading from local files. Local
// file modules are disabled by default.
func (s *Script) EnableFileImport(enable bool) {
	s.enableFileImport = enable
}

// SetImportFS sets a custom fs.FS used to resolve import and embed paths
// during compilation.  The FS should be rooted at the same directory that was
// (or will be) passed to SetImportDir.  This allows sandboxed or virtualised
// compilation environments (e.g. in-memory filesystems, WASM, wasip1) where
// direct os.ReadFile calls are unavailable or restricted.
// When nil (the default), the compiler creates an os.DirFS(importDir)
// automatically when SetImportDir is called.
func (s *Script) SetImportFS(fsys fs.FS) {
	s.importFS = fsys
}

// SetPermissions configures which privileged os-module operations the script is
// allowed to perform. The zero value of Permissions (default) denies all
// operations; use vm.UnrestrictedPermissions() to allow everything, or set
// individual Allow* fields to grant only the capabilities your script needs.
func (s *Script) SetPermissions(p vm.Permissions) {
	s.permissions = p
}

// SetMaxStringLen sets the maximum byte-length for string values produced
// during script execution. Zero (the default) defers to DefaultConfig.
func (s *Script) SetMaxStringLen(n int) {
	s.maxStringLen = n
}

// Compile compiles the script with all the defined variables and returns Program object.
func (s *Script) Compile() (*Program, error) {
	symbolTable, globals, err := s.prepCompile()
	if err != nil {
		return nil, err
	}

	fileSet := parser.NewFileSet()
	input := normalizeSource(s.input)
	srcFile := fileSet.AddFile(s.name, -1, len(input))
	p := parser.NewParser(srcFile, input, nil)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	c := vm.NewCompiler(srcFile, symbolTable, nil, s.modules, nil)
	c.EnableFileImport(s.enableFileImport)
	c.SetImportDir(s.importDir)
	if s.importFS != nil {
		c.SetImportFS(s.importFS)
	}
	if err := c.Compile(file); err != nil {
		return nil, err
	}

	// reduce globals size
	globals = globals[:symbolTable.MaxSymbols()+1]

	// global symbol names to indexes
	indices := make(map[string]int, len(globals))
	for _, name := range symbolTable.Names() {
		symbol, _, _ := symbolTable.Resolve(name, false)
		if symbol.Scope == vm.ScopeGlobal {
			indices[name] = symbol.Index
		}
	}

	// remove duplicates from constants
	bytecode := c.Bytecode()
	bytecode.RemoveDuplicates()

	// check the constant objects limit
	if s.maxConstObjects >= 0 {
		cnt := bytecode.CountObjects()
		if cnt > s.maxConstObjects {
			return nil, fmt.Errorf("exceeding constant objects limit: %d", cnt)
		}
	}
	return &Program{
		id:            programIDCounter.Add(1),
		globalIndices: indices,
		bytecode:      bytecode,
		globals:       globals,
		maxAllocs:     s.maxAllocs,
		maxStringLen:  s.maxStringLen,
		permissions:   s.permissions,
	}, nil
}

func normalizeSource(input []byte) []byte {
	input = bytes.TrimPrefix(input, []byte{0xEF, 0xBB, 0xBF})
	if bytes.HasPrefix(input, []byte("#!")) {
		if idx := bytes.IndexByte(input, '\n'); idx >= 0 {
			return input[idx+1:]
		}
		return []byte{}
	}
	return input
}

// Run compiles and runs the scripts. Use returned compiled object to access
// global variables.
func (s *Script) Run() (program *Program, err error) {
	program, err = s.Compile()
	if err != nil {
		return
	}
	err = program.Run()
	return
}

// RunContext is like Run but includes a context.
func (s *Script) RunContext(ctx context.Context) (program *Program, err error) {
	program, err = s.Compile()
	if err != nil {
		return
	}
	err = program.RunContext(ctx)
	return
}

func (s *Script) prepCompile() (symbolTable *vm.SymbolTable, globals []vm.Object, err error) {
	var names []string
	for name := range s.variables {
		names = append(names, name)
	}

	symbolTable = vm.NewSymbolTable()
	for idx, fn := range vm.GetAllBuiltinFunctions() {
		symbolTable.DefineBuiltin(idx, fn.Name)
	}

	globals = make([]vm.Object, vm.DefaultConfig.GlobalsSize)

	for idx, name := range names {
		symbol := symbolTable.Define(name)
		if symbol.Index != idx {
			panic(fmt.Errorf("wrong symbol index: %d != %d",
				idx, symbol.Index))
		}
		globals[symbol.Index] = s.variables[name].value
	}
	return
}

// Program is a compiled instance of the user script. Use Script.Compile() to
// create Compiled object.
// programIDCounter is incremented for every new Program to give it a
// stable, portable unique identity used for consistent lock ordering in
// Program.Equals (replacing the old unsafe.Pointer comparison).
var programIDCounter atomic.Int64

type Program struct {
	id            int64
	globalIndices map[string]int
	bytecode      *vm.Bytecode
	globals       []vm.Object
	maxAllocs     int64
	maxStringLen  int
	args          []string
	permissions   vm.Permissions
	lock          sync.RWMutex
}

// SetArgs sets the argument list that will be visible to the script via args().
func (p *Program) SetArgs(args []string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.args = args
}

// SetPermissions updates the permission policy for future Run/RunContext calls.
func (p *Program) SetPermissions(perm vm.Permissions) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.permissions = perm
}

// SetMaxStringLen updates the maximum string length for future Run/RunContext calls.
func (p *Program) SetMaxStringLen(n int) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.maxStringLen = n
}

// Bytecode returns the compiled bytecode of the Program.
func (p *Program) Bytecode() *vm.Bytecode {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.bytecode
}

// Unmarshal deserializes the Program from a byte slice.
// The global Modules() map is used to resolve imported builtin modules.
// Use UnmarshalWithModules to supply a custom module map (e.g. when the
// compiled script imports modules not present in the standard library).
func (p *Program) Unmarshal(b []byte) error {
	return p.UnmarshalWithModules(b, Modules())
}

// UnmarshalWithModules deserializes the Program from a byte slice using the
// provided module map to resolve imported builtin modules.  Pass a ModuleMap
// that contains every builtin module referenced by the compiled bytecode;
// the global Modules() map is a sensible starting point for standard-library
// modules, and custom modules can be added via ModuleMap.AddBuiltinModule.
func (p *Program) UnmarshalWithModules(b []byte, modules *vm.ModuleMap) (err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// header: [4]MAGIC [2]VERSION [4]SIZE = 10 bytes; tail: [32]SHA256
	if len(b) < 42 {
		return fmt.Errorf("invalid byte slice length: %d", len(b))
	}
	head := b[:10]
	body := b[10 : len(b)-32]
	tail := b[len(b)-32:]

	if string(head[:4]) != Magic {
		return fmt.Errorf("invalid magic number: %q", string(head[:4]))
	}
	ver := binary.LittleEndian.Uint16(head[4:6])
	if ver != FormatVersion {
		return fmt.Errorf("incompatible bytecode version: got %d, want %d", ver, FormatVersion)
	}
	size := binary.LittleEndian.Uint32(head[6:10])
	if size != uint32(len(body)) {
		return fmt.Errorf("invalid size: %d != %d", size, len(body))
	}
	wantSum := sha256.Sum256(body)
	if string(tail) != string(wantSum[:]) {
		return fmt.Errorf("invalid sha256 checksum: file may be corrupted or tampered")
	}

	n := 0
	n, p.globalIndices, err = codec.UnmarshalMap[string, int](n, body, codec.UnmarshalString, codec.UnmarshalInt)
	if err != nil {
		return err
	}
	n, p.globals, err = codec.UnmarshalSlice[vm.Object](n, body, vm.UnmarshalObject)
	if err != nil {
		return err
	}
	n, p.maxAllocs, err = codec.UnmarshalInt64(n, body)
	if err != nil {
		return err
	}

	if modules == nil {
		modules = vm.NewModuleMap()
	}
	p.bytecode = &vm.Bytecode{}
	err = p.bytecode.Unmarshal(body[n:], modules)
	if err != nil {
		return err
	}

	return nil
}

// Marshal serializes the Program into a byte slice.
func (p *Program) Marshal() ([]byte, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	code, err := p.bytecode.Marshal()
	if err != nil {
		return nil, err
	}

	n := 0
	data := make([]byte,
		codec.SizeMap[string, int](p.globalIndices, codec.SizeString, codec.SizeInt)+
			codec.SizeSlice[vm.Object](p.globals, vm.SizeOfObject)+
			codec.SizeInt64())
	n = codec.MarshalMap[string, int](n, data, p.globalIndices, codec.MarshalString, codec.MarshalInt)
	n = codec.MarshalSlice[vm.Object](n, data, p.globals, vm.MarshalObject)
	n = codec.MarshalInt64(n, data, p.maxAllocs)
	if n != len(data) {
		return nil, fmt.Errorf("encoded length mismatch: %d != %d", n, len(data))
	}

	body := append(data, code...)

	var head [10]byte
	copy(head[0:4], Magic)
	binary.LittleEndian.PutUint16(head[4:6], FormatVersion)
	binary.LittleEndian.PutUint32(head[6:10], uint32(len(body)))

	var tail [32]byte
	sum := sha256.Sum256(body)
	copy(tail[:], sum[:])

	return append(append(head[:], body...), tail[:]...), nil
}

// Run executes the compiled script in the virtual machine.
func (p *Program) Run() error {
	// Snapshot program state under a brief read lock.
	p.lock.RLock()
	globals := make([]vm.Object, len(p.globals))
	copy(globals, p.globals)
	bytecode := p.bytecode
	maxAllocs := p.maxAllocs
	maxStringLen := p.maxStringLen
	args := p.args
	permissions := p.permissions
	p.lock.RUnlock()

	v := vm.NewVM(context.Background(), bytecode, globals, &vm.Config{MaxAllocs: maxAllocs, MaxStringLen: maxStringLen, Permissions: permissions})
	// Always override Args so the script never inherits os.Args from the VM default.
	// Default to an empty slice when the caller did not call SetArgs.
	if args == nil {
		args = []string{}
	}
	v.Args = args
	err := v.Run()

	// Write back modified globals under a brief write lock.
	p.lock.Lock()
	copy(p.globals, globals)
	p.lock.Unlock()

	return err
}

// RunContext is like Run but includes a context.
func (p *Program) RunContext(ctx context.Context) (err error) {
	// Snapshot program state under a brief read lock.
	p.lock.RLock()
	globals := make([]vm.Object, len(p.globals))
	copy(globals, p.globals)
	bytecode := p.bytecode
	maxAllocs := p.maxAllocs
	maxStringLen := p.maxStringLen
	args := p.args
	permissions := p.permissions
	p.lock.RUnlock()

	v := vm.NewVM(ctx, bytecode, globals, &vm.Config{MaxAllocs: maxAllocs, MaxStringLen: maxStringLen, Permissions: permissions})
	// Always override Args so the script never inherits os.Args from the VM default.
	// Default to an empty slice when the caller did not call SetArgs.
	if args == nil {
		args = []string{}
	}
	v.Args = args
	ch := make(chan error, 1)
	go func() {
		ch <- v.Run()
	}()

	select {
	case <-ctx.Done():
		v.Abort()
		<-ch
		err = ctx.Err()
	case err = <-ch:
	}

	// Write back modified globals under a brief write lock.
	p.lock.Lock()
	copy(p.globals, globals)
	p.lock.Unlock()

	return
}

// Clone creates a new copy of Compiled. Cloned copies are safe for concurrent
// use by multiple goroutines.
func (p *Program) Clone() *Program {
	p.lock.RLock()
	defer p.lock.RUnlock()

	clone := &Program{
		id:            programIDCounter.Add(1),
		globalIndices: p.globalIndices,
		bytecode:      p.bytecode,
		globals:       make([]vm.Object, len(p.globals)),
		maxAllocs:     p.maxAllocs,
		maxStringLen:  p.maxStringLen,
		permissions:   p.permissions,
	}
	// deep-copy global objects so mutations in the clone do not affect
	// the original (or vice versa). (Issue #10)
	for idx, g := range p.globals {
		if g != nil {
			clone.globals[idx] = g.Copy()
		}
	}
	return clone
}

// IsDefined returns true if the variable name is defined (has value) before or
// after the execution.
func (p *Program) IsDefined(name string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	idx, ok := p.globalIndices[name]
	if !ok {
		return false
	}
	v := p.globals[idx]
	if v == nil {
		return false
	}
	return v != vm.UndefinedValue
}

// Get returns a variable identified by the name.
func (p *Program) Get(name string) *Variable {
	p.lock.RLock()
	defer p.lock.RUnlock()

	value := vm.UndefinedValue
	if idx, ok := p.globalIndices[name]; ok {
		value = p.globals[idx]
		if value == nil {
			value = vm.UndefinedValue
		}
	}
	return &Variable{
		name:  name,
		value: value,
	}
}

// GetAll returns all the variables that are defined by the compiled script.
func (p *Program) GetAll() []*Variable {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var vars []*Variable
	for name, idx := range p.globalIndices {
		value := p.globals[idx]
		if value == nil {
			value = vm.UndefinedValue
		}
		vars = append(vars, &Variable{
			name:  name,
			value: value,
		})
	}
	return vars
}

// Set replaces the value of a global variable identified by the name. An error
// will be returned if the name was not defined during compilation.
func (p *Program) Set(name string, value interface{}) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	obj, err := vm.FromInterface(value)
	if err != nil {
		return err
	}
	idx, ok := p.globalIndices[name]
	if !ok {
		return fmt.Errorf("'%s' is not defined", name)
	}
	p.globals[idx] = obj
	return nil
}

// Equals compares two Program objects for equality.
func (p *Program) Equals(other *Program) bool {
	// Short-circuit: a program always equals itself.
	if p == other {
		p.lock.RLock()
		defer p.lock.RUnlock()
		return true
	}

	// Acquire locks in a consistent order (by stable integer ID) to avoid
	// deadlock when two goroutines call a.Equals(b) and b.Equals(a)
	// concurrently.  Using integer IDs instead of pointer addresses avoids
	// undefined behaviour under moving GCs (issue 6.3).
	first, second := &p.lock, &other.lock
	if p.id > other.id {
		first, second = second, first
	}
	first.RLock()
	defer first.RUnlock()
	second.RLock()
	defer second.RUnlock()

	if len(p.globalIndices) != len(other.globalIndices) {
		return false
	}
	for k, v := range p.globalIndices {
		if ov, ok := other.globalIndices[k]; !ok || v != ov {
			return false
		}
	}
	if len(p.globals) != len(other.globals) {
		return false
	}
	for i, v := range p.globals {
		if ov := other.globals[i]; v != ov {
			return false
		}
	}
	if p.maxAllocs != other.maxAllocs {
		return false
	}
	if !p.bytecode.Equals(other.bytecode) {
		return false
	}
	return true
}
