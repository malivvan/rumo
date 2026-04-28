package module

import (
	"context"
	"fmt"

	"github.com/malivvan/rumo/vm"
)

// TypeRegistration is the type-erased view of a *Type[T] that allows it to
// be registered into a BuiltinModule without exposing its type parameter.
//
// It is implemented exclusively by *Type[T]; user code should not implement
// it directly.
type TypeRegistration interface {
	// typeName returns the exported name of the type (e.g. "Mutex").
	typeName() string
	// typeExport returns the parsed Export descriptor for the type itself.
	typeExport() *Export
	// methodExports returns the parsed Export descriptors for each method.
	methodExports() map[string]*Export
	// constructor builds the *vm.BuiltinFunction that the VM will invoke
	// when user code calls the type, e.g. `m := sync.Mutex()`.
	constructor() *vm.BuiltinFunction
}

// Constructor is the Go-side factory invoked when user code calls the type
// (e.g. `m := sync.Mutex()`). It receives the context and the raw VM-level
// arguments and returns the per-instance state of Go type T.
//
// A non-nil error is converted to a Rumo error via WrapError and returned
// to the caller as the value of the constructor expression.
type Constructor[T any] func(ctx context.Context, args []vm.Object) (T, error)

// MethodBinder receives the per-instance state and returns the Go-level
// implementation of a single method. The returned value is fed through
// Func() so any signature accepted by Func() (e.g. `func() error`,
// `func(string) string`) works.
//
// Typical usage simply returns a method value bound to the receiver:
//
//	t.Method("lock()", func(m *sync.Mutex) any { return m.Lock })
type MethodBinder[T any] func(self T) any

// Type is a module-level Type with a fixed set of methods bound to a
// per-instance Go state of type T.
//
// Instances are returned by the constructor as a *vm.ImmutableMap of
// per-instance *vm.BuiltinFunction methods. Both the constructor and the
// returned instance map use only Object kinds that are already part of the
// VM's bytecode encoding (BuiltinFunction, ImmutableMap), so a Type defined
// this way participates transparently in compile-and-load round-trips:
//
//   - The Type itself is registered in its host BuiltinModule as a
//     *vm.BuiltinFunction constructor; it is encoded by name (just like any
//     other builtin) and re-bound to the in-process module on decode (see
//     bytecode.fixDecodedObject).
//   - Method values returned by an instance are also *vm.BuiltinFunction
//     and therefore encodable; their .Value closure is process-local and
//     intentionally not serialised, matching the pre-existing behaviour for
//     all other BuiltinFunction values.
type Type[T any] struct {
	name    string
	export  *Export
	ctor    Constructor[T]
	methods []typeMethod[T]
	exports map[string]*Export
}

type typeMethod[T any] struct {
	name   string
	export *Export
	bind   MethodBinder[T]
}

// NewType creates a new Type with the given export definition and
// constructor.
//
// The export definition follows the same syntax as Func() / Const(), e.g.
// "Mutex() (m *Mutex)". Only the Name part is required; the rest is used
// for documentation.
func NewType[T any](def string, ctor Constructor[T]) *Type[T] {
	if len(def) == 0 {
		panic("type def cannot be empty")
	}
	if ctor == nil {
		panic("type constructor cannot be nil")
	}
	e := ParseExport(def)
	if e == nil {
		panic("invalid type definition: " + def)
	}
	return &Type[T]{
		name:    e.Name,
		export:  e,
		ctor:    ctor,
		exports: make(map[string]*Export),
	}
}

// Method registers a method on the Type. The bind callback is invoked once
// per instance to produce the Go-side implementation, which is then wrapped
// via Func() into a vm.CallableFunc.
//
// Method names must be unique within a Type; duplicate registrations panic.
func (t *Type[T]) Method(def string, bind MethodBinder[T]) *Type[T] {
	if len(def) == 0 {
		panic("method def cannot be empty")
	}
	if bind == nil {
		panic("method binder cannot be nil")
	}
	e := ParseExport(def)
	if e == nil {
		panic("invalid method definition: " + def)
	}
	if _, exists := t.exports[e.Name]; exists {
		panic(fmt.Errorf("type %s: method already registered: %s", t.name, e.Name))
	}
	t.methods = append(t.methods, typeMethod[T]{name: e.Name, export: e, bind: bind})
	t.exports[e.Name] = e
	return t
}

// Methods returns the parsed Export descriptors for the methods registered
// on this Type, keyed by method name. The returned map is a snapshot; the
// caller may mutate it freely.
func (t *Type[T]) Methods() map[string]*Export {
	out := make(map[string]*Export, len(t.exports))
	for k, v := range t.exports {
		out[k] = v
	}
	return out
}

// --- TypeRegistration implementation -----------------------------------

func (t *Type[T]) typeName() string    { return t.name }
func (t *Type[T]) typeExport() *Export { return t.export }
func (t *Type[T]) methodExports() map[string]*Export {
	out := make(map[string]*Export, len(t.exports))
	for k, v := range t.exports {
		out[k] = v
	}
	return out
}

func (t *Type[T]) constructor() *vm.BuiltinFunction {
	// Snapshot the registration data so the closure does not alias the
	// receiver — types are conceptually immutable once registered.
	methods := make([]typeMethod[T], len(t.methods))
	copy(methods, t.methods)
	name := t.name
	ctor := t.ctor
	return &vm.BuiltinFunction{
		Name: name,
		Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			self, err := ctor(ctx, args)
			if err != nil {
				return WrapError(err), nil
			}
			attrs := make(map[string]vm.Object, len(methods))
			for _, m := range methods {
				impl := m.bind(self)
				if impl == nil {
					return nil, fmt.Errorf("type %s: method %s: nil implementation", name, m.name)
				}
				attrs[m.name] = &vm.BuiltinFunction{
					Name:  name + "." + m.name,
					Value: Func(impl),
				}
			}
			return &vm.ImmutableMap{Value: attrs}, nil
		},
	}
}
