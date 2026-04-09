package module

// SourceModule represents a module defined by source code.
type SourceModule struct {
	export map[string]*Export
	module []byte
}

// Module returns the source code of the module.
func (m *SourceModule) Module() []byte { return m.module }

// Exports returns the export map of the module.
func (m *SourceModule) Exports() map[string]*Export {
	return m.export
}

// NewSource creates a new source module with the given name and data, and registers it in the module map.
func NewSource(module string) *SourceModule {
	m := &SourceModule{module: []byte(module), export: make(map[string]*Export)}
	exports, err := ParseExports(module)
	if err != nil {
		panic(err)
	}
	for _, export := range exports {
		m.export[export.Name] = export
	}
	return m
}
