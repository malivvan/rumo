package std

//go:generate go run gensrcmods.go

import (
	"context"
	"fmt"
	"github.com/malivvan/rumo/vm"
)

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

// Func returns a BuiltinFunction from the given function value.
func Func(function any) *vm.BuiltinFunction {
	if f, ok := Prop(function).get().(*vm.BuiltinFunction); ok {
		return f
	}
	return nil
}

// Prop returns a Property for the given property value.
func Prop(property any) *Property {
	switch v := property.(type) {
	case string:
		return &Property{
			get: func() vm.Object {
				return &vm.String{Value: v}
			},
		}
	case *string:
		return &Property{
			get: func() vm.Object { return &vm.String{Value: *v} },
			set: func(o vm.Object) error {
				if str, ok := o.(*vm.String); ok {
					*v = str.Value
					return nil
				}
				return &vm.ErrInvalidArgumentType{Name: "property", Expected: "string", Found: o.TypeName()}
			},
		}

	case func() error:
		return &Property{
			get: func() vm.Object {
				return &vm.BuiltinFunction{
					Value: FuncARE(v),
				}
			},
		}
	case func([]byte) (int, error):
		return &Property{
			get: func() vm.Object {
				return &vm.BuiltinFunction{
					Value: FuncAYRIE(v),
				}
			},
		}
	}
	return nil
}

// Property represents a property with getter and optional setter functions.
type Property struct {
	get func() vm.Object
	set func(vm.Object) error
}

// Get returns the value of the property.
func (prop *Property) Get() vm.Object {
	return prop.get()
}

// Set sets the value of the property if it is writable.
func (prop *Property) Set(value vm.Object) error {
	if prop.set == nil {
		return fmt.Errorf("property is not writable")
	}
	return prop.set(value)
}

// CanCall checks if the property is callable.
func (prop *Property) CanCall() bool {
	_, ok := prop.get().(*vm.BuiltinFunction)
	return ok
}

// Call invokes the property as a function if it is callable.
func (prop *Property) Call(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if f, ok := prop.get().(*vm.BuiltinFunction); ok {
		return f.Value(ctx, args...)
	}
	return nil, fmt.Errorf("property is not callable")
}

// Object is a generic object type that can hold any value of type T.
type Object[T any] struct {
	vm.ObjectImpl
	Name  string
	Value T
	Props map[string]*Property
	ToStr func(T) string
}

// NewObject creates a new Object with the given value, name, string conversion function, and properties.
func NewObject[T any](value T, name string, toStr func(T) string, bind map[string]any) *Object[T] {
	w := new(Object[T])
	w.Name = name
	w.Value = value
	w.Props = make(map[string]*Property)
	w.ToStr = toStr
	for k, v := range bind {
		w.Props[k] = Prop(v)
	}
	return w
}

// IndexGet retrieves a property value by its key.
func (obj *Object[T]) IndexGet(key vm.Object) (vm.Object, error) {
	if k, ok := vm.ToString(key); ok {
		if v, ok := obj.Props[k]; ok {
			return v.Get(), nil
		}
	}
	return nil, fmt.Errorf("property not found")
}

// IndexSet sets a property value by its key.
func (obj *Object[T]) IndexSet(key vm.Object, val vm.Object) error {
	if k, ok := vm.ToString(key); ok {
		if v, ok := obj.Props[k]; ok {
			return v.Set(val)
		}
	}
	return fmt.Errorf("property not found")
}

// String returns the string representation of the object value using the ToStr function.
func (obj *Object[T]) String() string {
	return obj.ToStr(obj.Value)
}

// TypeName returns the name of the object type.
func (obj *Object[T]) TypeName() string {
	return obj.Name
}
