package module

// SourceModule represents a module defined by source code.
type SourceModule struct {
	name string
	data []byte
}

// NewSource creates a new source module with the given name and data, and registers it in the module map.
func NewSource(name string, data []byte) *SourceModule {
	if len(name) == 0 {
		panic("module name cannot be empty")
	}
	m := &SourceModule{name: name, data: data}
	return m
}
