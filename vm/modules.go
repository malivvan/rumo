package vm

import "sync"

// Importable interface represents importable module instance.
type Importable interface {
	// Import should return either an Object or module source code ([]byte).
	Import(moduleName string) (interface{}, error)
}

// ModuleMap represents a set of named modules. Use NewModuleMap to create a
// new module map. It is safe for concurrent use: reads and writes are
// guarded by an internal RWMutex, so an embedder may keep registering
// modules while compiled programs are being loaded.
type ModuleMap struct {
	mu sync.RWMutex
	m  map[string]Importable
}

// NewModuleMap creates a new module map.
func NewModuleMap() *ModuleMap {
	return &ModuleMap{
		m: make(map[string]Importable),
	}
}

// Add adds an import module.
func (m *ModuleMap) Add(name string, module Importable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[name] = module
}

// AddBuiltinModule adds a builtin module.
func (m *ModuleMap) AddBuiltinModule(name string, attrs map[string]Object) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[name] = &BuiltinModule{Attrs: attrs}
}

// AddSourceModule adds a source module.
func (m *ModuleMap) AddSourceModule(name string, src []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[name] = &SourceModule{Src: src}
}

// Remove removes a named module.
func (m *ModuleMap) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, name)
}

// Get returns an import module identified by name. It returns if the name is
// not found.
func (m *ModuleMap) Get(name string) Importable {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.m[name]
}

// GetBuiltinModule returns a builtin module identified by name. It returns
// if the name is not found or the module is not a builtin module.
func (m *ModuleMap) GetBuiltinModule(name string) *BuiltinModule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mod, _ := m.m[name].(*BuiltinModule)
	return mod
}

// GetSourceModule returns a source module identified by name. It returns if
// the name is not found or the module is not a source module.
func (m *ModuleMap) GetSourceModule(name string) *SourceModule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mod, _ := m.m[name].(*SourceModule)
	return mod
}

// Copy creates a copy of the module map.
func (m *ModuleMap) Copy() *ModuleMap {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c := &ModuleMap{
		m: make(map[string]Importable),
	}
	for name, mod := range m.m {
		c.m[name] = mod
	}
	return c
}

// Each iterates over all named modules and applies the given function to each of them.
func (m *ModuleMap) Each(f func(name string, mod Importable)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, mod := range m.m {
		f(name, mod)
	}
}

// Len returns the number of named modules.
func (m *ModuleMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

// AddMap adds named modules from another module map.
func (m *ModuleMap) AddMap(o *ModuleMap) {
	o.mu.RLock()
	m.mu.Lock()
	for name, mod := range o.m {
		m.m[name] = mod
	}
	m.mu.Unlock()
	o.mu.RUnlock()
}

// SourceModule is an importable module that's written in vm.
type SourceModule struct {
	Src []byte
}

// Import returns a module source code.
func (m *SourceModule) Import(_ string) (interface{}, error) {
	return m.Src, nil
}
