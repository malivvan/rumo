package vm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/malivvan/rumo/vm/purego"
)

// ----------------------------------------------------------------------------
// Native library type system
// ----------------------------------------------------------------------------

// NativeKind identifies a rumo <-> C type mapping used by the native runtime.
type NativeKind int

const (
	NativeInvalid NativeKind = iota
	NativeVoid               // no value (return only)
	NativeInt                // int64  <=> intptr_t / long
	NativeUInt               // uint64 <=> uintptr_t / unsigned long
	NativeBool               // bool
	NativeFloat              // float64 <=> double
	NativeString             // string  <=> const char* (null-terminated)
	NativePtr                // uintptr <=> void*
	NativeBytes              // []byte  <=> void* (pointer to slice data)
)

// nativeKindByName resolves a user-facing type keyword such as "int" or
// "float" to its internal NativeKind.  Names are case sensitive.
func nativeKindByName(name string) (NativeKind, bool) {
	switch name {
	case "void":
		return NativeVoid, true
	case "int":
		return NativeInt, true
	case "uint":
		return NativeUInt, true
	case "bool":
		return NativeBool, true
	case "float":
		return NativeFloat, true
	case "string":
		return NativeString, true
	case "ptr":
		return NativePtr, true
	case "bytes":
		return NativeBytes, true
	}
	return NativeInvalid, false
}

func (k NativeKind) String() string {
	switch k {
	case NativeVoid:
		return "void"
	case NativeInt:
		return "int"
	case NativeUInt:
		return "uint"
	case NativeBool:
		return "bool"
	case NativeFloat:
		return "float"
	case NativeString:
		return "string"
	case NativePtr:
		return "ptr"
	case NativeBytes:
		return "bytes"
	}
	return "invalid"
}

// goType returns the reflect.Type used to register the corresponding C
// function signature with purego.
func (k NativeKind) goType() reflect.Type {
	switch k {
	case NativeInt:
		return reflect.TypeOf(int64(0))
	case NativeUInt:
		return reflect.TypeOf(uint64(0))
	case NativeBool:
		return reflect.TypeOf(false)
	case NativeFloat:
		return reflect.TypeOf(float64(0))
	case NativeString:
		return reflect.TypeOf("")
	case NativePtr, NativeBytes:
		return reflect.TypeOf(uintptr(0))
	}
	return nil
}

// NativeFuncSpec is the compile-time description of a single native function
// binding captured from a `native ... { ... }` statement.
type NativeFuncSpec struct {
	Name   string
	Params []NativeKind
	Return NativeKind // NativeVoid = no return
}

// ----------------------------------------------------------------------------
// Native loader (constant placed in the bytecode)
// ----------------------------------------------------------------------------

// Native is the constant object produced by the compiler for every `native`
// statement. It embeds the full loader spec and implements Call so the VM
// can materialize the library at runtime by emitting a regular OpCall.
// Calling it loads the shared library (once) and returns a Map of callable
// functions keyed by their declared names.
type Native struct {
	ObjectImpl
	Path  string
	Funcs []NativeFuncSpec

	mu     sync.Mutex
	loaded *Map
	handle uintptr
}

// TypeName returns the name of the type.
func (o *Native) TypeName() string { return "native-loader" }

func (o *Native) String() string {
	return fmt.Sprintf("<native-loader %q>", o.Path)
}

// Copy returns a copy of the loader.  The cached handle is intentionally not
// copied so the clone will dlopen on first use, just like the original.
func (o *Native) Copy() Object {
	funcs := make([]NativeFuncSpec, len(o.Funcs))
	copy(funcs, o.Funcs)
	return &Native{Path: o.Path, Funcs: funcs}
}

// Equals returns true if another loader points at the same library and
// declares the same bindings in the same order.
func (o *Native) Equals(x Object) bool {
	t, ok := x.(*Native)
	if !ok {
		return false
	}
	if o.Path != t.Path || len(o.Funcs) != len(t.Funcs) {
		return false
	}
	for i := range o.Funcs {
		a, b := o.Funcs[i], t.Funcs[i]
		if a.Name != b.Name || a.Return != b.Return || len(a.Params) != len(b.Params) {
			return false
		}
		for j := range a.Params {
			if a.Params[j] != b.Params[j] {
				return false
			}
		}
	}
	return true
}

// IsFalsy — a loader is falsy only if it binds no symbols.
func (o *Native) IsFalsy() bool { return len(o.Funcs) == 0 }

// CanCall allows OpCall to dispatch to us.
func (o *Native) CanCall() bool { return true }

// Call materializes the library. The result is cached: subsequent calls
// return the same *Map.
func (o *Native) Call(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("native loader takes no arguments")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.loaded != nil {
		return o.loaded, nil
	}

	handle, err := purego.Dlopen(o.Path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("native: failed to open %q: %w", o.Path, err)
	}

	entries := make(map[string]Object, len(o.Funcs)+2)

	for _, spec := range o.Funcs {
		bound, bindErr := buildNativeBinding(handle, spec)
		if bindErr != nil {
			_ = purego.Dlclose(handle)
			return nil, fmt.Errorf("native %s.%s: %w", o.Path, spec.Name, bindErr)
		}
		entries[spec.Name] = bound
	}

	// Introspection helpers ------------------------------------------------
	entries["__path__"] = &String{Value: o.Path}

	closedFlag := false
	entries["close"] = &BuiltinFunction{Name: "close", Value: func(_ context.Context, args ...Object) (Object, error) {
		if len(args) != 0 {
			return nil, ErrWrongNumArguments
		}
		o.mu.Lock()
		defer o.mu.Unlock()
		if closedFlag {
			return UndefinedValue, nil
		}
		closedFlag = true
		if err := purego.Dlclose(handle); err != nil {
			return nil, err
		}
		return UndefinedValue, nil
	}}

	m := &Map{Value: entries}
	o.handle = handle
	o.loaded = m
	return m, nil
}

// ----------------------------------------------------------------------------
// Per-symbol binding
// ----------------------------------------------------------------------------

// buildNativeBinding resolves the symbol using dlsym and wraps it in a
// *BuiltinFunction that converts rumo arguments <-> Go values, invokes the
// purego-registered function and converts the result back.
func buildNativeBinding(handle uintptr, spec NativeFuncSpec) (*BuiltinFunction, error) {
	// sanity check types
	for i, p := range spec.Params {
		if p == NativeVoid {
			return nil, fmt.Errorf("parameter %d has void type", i)
		}
		if p == NativeInvalid {
			return nil, fmt.Errorf("parameter %d has invalid type", i)
		}
	}
	if spec.Return == NativeInvalid {
		return nil, errors.New("invalid return type")
	}

	// build the dynamic Go function signature
	in := make([]reflect.Type, len(spec.Params))
	for i, p := range spec.Params {
		in[i] = p.goType()
	}
	var out []reflect.Type
	if spec.Return != NativeVoid {
		out = []reflect.Type{spec.Return.goType()}
	}
	fnType := reflect.FuncOf(in, out, false)

	// allocate a pointer to a nil function value and let purego populate it
	fnPtrValue := reflect.New(fnType)
	purego.RegisterLibFunc(fnPtrValue.Interface(), handle, spec.Name)
	fnValue := fnPtrValue.Elem()

	params := append([]NativeKind(nil), spec.Params...)
	ret := spec.Return
	name := spec.Name

	return &BuiltinFunction{
		Name: name,
		Value: func(_ context.Context, args ...Object) (Object, error) {
			if len(args) != len(params) {
				return nil, fmt.Errorf("native %s: wrong number of arguments: want=%d got=%d",
					name, len(params), len(args))
			}

			// Rumo values frequently outlive a single call.  Go values built
			// out of them (e.g. CString backing memory) need to survive the
			// C call itself; keepAlive captures those references.  purego
			// also manages memory for arguments it converts, but we keep
			// user-provided slices reachable for the duration of the call.
			var keepAlive []any

			callArgs := make([]reflect.Value, len(args))
			for i, arg := range args {
				v, pin, err := rumoToNativeArg(arg, params[i])
				if err != nil {
					return nil, fmt.Errorf("native %s: argument %d: %w", name, i, err)
				}
				callArgs[i] = v
				if pin != nil {
					keepAlive = append(keepAlive, pin)
				}
			}

			results := fnValue.Call(callArgs)
			// keep arguments alive until after the call has returned
			_ = keepAlive

			if ret == NativeVoid {
				return UndefinedValue, nil
			}
			return nativeResultToRumo(results[0], ret)
		},
	}, nil
}

// rumoToNativeArg converts a rumo Object to a reflect.Value typed exactly as
// purego expects for a single parameter.  The second return value (if
// non-nil) is an object that must be kept alive for the duration of the C
// call — callers are responsible for doing so.
func rumoToNativeArg(o Object, kind NativeKind) (reflect.Value, any, error) {
	switch kind {
	case NativeInt:
		n, ok := ToInt64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected int, got %s", o.TypeName())
		}
		return reflect.ValueOf(n), nil, nil
	case NativeUInt:
		n, ok := ToInt64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected uint/int, got %s", o.TypeName())
		}
		return reflect.ValueOf(uint64(n)), nil, nil
	case NativeBool:
		b, ok := ToBool(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected bool, got %s", o.TypeName())
		}
		return reflect.ValueOf(b), nil, nil
	case NativeFloat:
		f, ok := ToFloat64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected float, got %s", o.TypeName())
		}
		return reflect.ValueOf(f), nil, nil
	case NativeString:
		// purego copies non null-terminated strings into arena memory that
		// is only valid for the call, so we don't need to keep them alive
		// ourselves, but we do need to hand the pointer-backed string to
		// reflect as a string value.
		s, ok := ToString(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected string, got %s", o.TypeName())
		}
		return reflect.ValueOf(s), nil, nil
	case NativePtr:
		switch v := o.(type) {
		case *Int:
			return reflect.ValueOf(uintptr(v.Value)), nil, nil
		case *Undefined:
			return reflect.ValueOf(uintptr(0)), nil, nil
		case *Bytes:
			if len(v.Value) == 0 {
				return reflect.ValueOf(uintptr(0)), v, nil
			}
			return reflect.ValueOf(uintptr(unsafe.Pointer(&v.Value[0]))), v, nil
		case *String:
			if v.Value == "" {
				return reflect.ValueOf(uintptr(0)), nil, nil
			}
			// unsafe.StringData is the canonical way to expose the backing
			// bytes of a Go string without copying.
			data := unsafe.StringData(v.Value)
			return reflect.ValueOf(uintptr(unsafe.Pointer(data))), v, nil
		}
		return reflect.Value{}, nil, fmt.Errorf("expected ptr-convertible, got %s", o.TypeName())
	case NativeBytes:
		b, ok := o.(*Bytes)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected bytes, got %s", o.TypeName())
		}
		if len(b.Value) == 0 {
			return reflect.ValueOf(uintptr(0)), b, nil
		}
		return reflect.ValueOf(uintptr(unsafe.Pointer(&b.Value[0]))), b, nil
	}
	return reflect.Value{}, nil, fmt.Errorf("unsupported native kind %s", kind)
}

// nativeResultToRumo boxes a Go value produced by a purego call into the
// matching rumo Object.
func nativeResultToRumo(v reflect.Value, kind NativeKind) (Object, error) {
	switch kind {
	case NativeInt:
		return &Int{Value: v.Int()}, nil
	case NativeUInt:
		// rumo does not ship a dedicated unsigned integer type by default;
		// fall back to signed Int.  Callers can still round-trip 64-bit
		// patterns but sign bit semantics apply.
		return &Int{Value: int64(v.Uint())}, nil
	case NativeBool:
		if v.Bool() {
			return TrueValue, nil
		}
		return FalseValue, nil
	case NativeFloat:
		return &Float{Value: v.Float()}, nil
	case NativeString:
		// purego already copied the C string into Go memory.
		return &String{Value: v.String()}, nil
	case NativePtr:
		return &Int{Value: int64(v.Uint())}, nil
	case NativeBytes:
		// Bytes return is ambiguous (length unknown) - expose as a raw ptr.
		return &Int{Value: int64(v.Uint())}, nil
	}
	return nil, fmt.Errorf("unsupported native return kind %s", kind)
}

