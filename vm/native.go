//go:build native

package vm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// NativeSupported reports whether the current build includes FFI (native)
// support.  It returns true when compiled with -tags native.
func NativeSupported() bool { return true }

// resolveLibPath returns the absolute, symlink-resolved path for a shared
// library name.  For names that contain a path separator the path is made
// absolute and all symlinks are resolved.  For bare names (e.g. "libm.so.6")
// the function searches the dynamic-linker search path in the same order the
// OS linker would: LD_LIBRARY_PATH (Linux) / DYLD_LIBRARY_PATH (macOS),
// followed by a set of well-known standard directories.  If no file is found
// the original name is returned unchanged.
func resolveLibPath(name string) string {
	if strings.ContainsRune(name, '/') {
		abs, err := filepath.Abs(name)
		if err != nil {
			return name
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return abs
		}
		return real
	}

	// Bare library name: build a candidate search list.
	var dirs []string
	if v := os.Getenv("LD_LIBRARY_PATH"); v != "" { // Linux
		dirs = append(dirs, filepath.SplitList(v)...)
	}
	if v := os.Getenv("DYLD_LIBRARY_PATH"); v != "" { // macOS
		dirs = append(dirs, filepath.SplitList(v)...)
	}
	if v := os.Getenv("DYLD_FALLBACK_LIBRARY_PATH"); v != "" { // macOS fallback
		dirs = append(dirs, filepath.SplitList(v)...)
	}
	// Standard system paths.
	dirs = append(dirs, "/usr/local/lib", "/usr/lib", "/lib")
	// Architecture-specific Linux multilib paths.
	switch runtime.GOARCH {
	case "amd64":
		dirs = append(dirs, "/usr/lib/x86_64-linux-gnu", "/lib/x86_64-linux-gnu")
	case "arm64":
		dirs = append(dirs, "/usr/lib/aarch64-linux-gnu", "/lib/aarch64-linux-gnu")
	case "386":
		dirs = append(dirs, "/usr/lib/i386-linux-gnu", "/lib/i386-linux-gnu")
	case "arm":
		dirs = append(dirs, "/usr/lib/arm-linux-gnueabihf", "/lib/arm-linux-gnueabihf")
	}
	for _, dir := range dirs {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			if real, err := filepath.EvalSymlinks(candidate); err == nil {
				return real
			}
			return candidate
		}
	}
	return name
}

// ResolveNativePath checks whether the shared library identified by name is
// loadable via the same dlopen mechanism the vm package uses at runtime.
// If the library can be opened it returns the resolved absolute path;
// otherwise it returns an empty string.  This lets callers distinguish
// "path stored in bytecode but not present on this system" from "path present
// and ready", and also provides the concrete on-disk path for display.
func ResolveNativePath(name string) string {
	if name == "" {
		return ""
	}
	handle, err := purego.Dlopen(name, purego.RTLD_LAZY)
	if err != nil {
		return ""
	}
	_ = purego.Dlclose(handle)
	return resolveLibPath(name)
}

// ----------------------------------------------------------------------------
// Native allow-list
// ----------------------------------------------------------------------------

// nativeAllowMu guards nativeAllowPaths.
var nativeAllowMu sync.RWMutex

// nativeAllowPaths is the set of shared-library paths that the host
// application has explicitly approved for dlopen.  When nil every path is
// blocked (zero-trust default).
var nativeAllowPaths map[string]struct{}

// AllowNativePath registers path as a trusted native library path that
// Native.Call is permitted to dlopen.  Call this during program initialisation,
// before executing any scripts, for every shared library you intentionally
// expose through the native FFI.
//
// An empty allow-list (the default) blocks all native loads.
func AllowNativePath(path string) {
	nativeAllowMu.Lock()
	defer nativeAllowMu.Unlock()
	if nativeAllowPaths == nil {
		nativeAllowPaths = make(map[string]struct{})
	}
	nativeAllowPaths[path] = struct{}{}
}

// ClearNativeAllowList removes every entry from the allow-list.  It is
// intended for use in tests to reset global state between test cases.
func ClearNativeAllowList() {
	nativeAllowMu.Lock()
	defer nativeAllowMu.Unlock()
	nativeAllowPaths = nil
}

// isNativePathAllowed reports whether path has been registered with
// AllowNativePath.
func isNativePathAllowed(path string) bool {
	nativeAllowMu.RLock()
	defer nativeAllowMu.RUnlock()
	_, ok := nativeAllowPaths[path]
	return ok
}

// ----------------------------------------------------------------------------
// Native library type system
// ----------------------------------------------------------------------------
// NativeKind, NativeInt/UInt/Float/Double aliases, and NativeFuncSpec are
// defined in native_types.go (no build tag) so they are available in both
// native and non-native builds for encoding/decoding bytecode.

// nativeKindByName resolves a user-facing type keyword such as "int" or
// "float" to its internal NativeKind.  Names are case sensitive.
func nativeKindByName(name string) (NativeKind, bool) {
	switch name {
	case "void":
		return NativeVoid, true
	case "byte":
		return NativeByte, true
	case "int8":
		return NativeInt8, true
	case "uint8":
		return NativeUint8, true
	case "int16":
		return NativeInt16, true
	case "uint16":
		return NativeUint16, true
	case "int":
		return NativeInt, true
	case "uint":
		return NativeUInt, true
	case "int64":
		return NativeInt64, true
	case "uint64":
		return NativeUint64, true
	case "bool":
		return NativeBool, true
	case "float":
		return NativeFloat32, true
	case "double", "float64":
		return NativeFloat64, true
	case "float32":
		return NativeFloat32, true
	case "rune":
		return NativeRune, true
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
	case NativeByte:
		return "byte"
	case NativeInt8:
		return "int8"
	case NativeUint8:
		return "uint8"
	case NativeInt16:
		return "int16"
	case NativeUint16:
		return "uint16"
	case NativeInt64:
		return "int"
	case NativeUint64:
		return "uint"
	case NativeBool:
		return "bool"
	case NativeFloat64:
		return "double"
	case NativeFloat32:
		return "float"
	case NativeRune:
		return "rune"
	case NativeString:
		return "string"
	case NativePtr:
		return "ptr"
	case NativeBytes:
		return "bytes"
	case NativeStruct:
		return "struct"
	}
	return "invalid"
}

// goType returns the reflect.Type used to register the corresponding C
// function signature with purego.
func (k NativeKind) goType() reflect.Type {
	switch k {
	case NativeByte:
		return reflect.TypeOf(byte(0))
	case NativeInt8:
		return reflect.TypeOf(int8(0))
	case NativeUint8:
		return reflect.TypeOf(uint8(0))
	case NativeInt16:
		return reflect.TypeOf(int16(0))
	case NativeUint16:
		return reflect.TypeOf(uint16(0))
	case NativeInt64:
		return reflect.TypeOf(int64(0))
	case NativeUint64:
		return reflect.TypeOf(uint64(0))
	case NativeBool:
		return reflect.TypeOf(false)
	case NativeFloat64:
		return reflect.TypeOf(float64(0))
	case NativeFloat32:
		return reflect.TypeOf(float32(0))
	case NativeRune:
		return reflect.TypeOf(int32(0))
	case NativeString:
		return reflect.TypeOf("")
	case NativePtr, NativeBytes:
		return reflect.TypeOf(uintptr(0))
	}
	return nil
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
	Path    string
	Funcs   []NativeFuncSpec
	Structs []NativeStructSpec

	mu          sync.Mutex
	loaded      *Map
	handle      uintptr
	structTypes []reflect.Type // lazily built reflect.Types per Structs entry
}

// TypeName returns the name of the type.
func (o *Native) TypeName() string { return "native-loader" }

func (o *Native) String() string {
	return fmt.Sprintf("<native-loader %q>", o.Path)
}

// NativePath returns the shared library path embedded in this loader constant.
func (o *Native) NativePath() string { return o.Path }

// Copy returns a copy of the loader.  The cached handle is intentionally not
// copied so the clone will dlopen on first use, just like the original.
func (o *Native) Copy() Object {
	funcs := make([]NativeFuncSpec, len(o.Funcs))
	copy(funcs, o.Funcs)
	structs := make([]NativeStructSpec, len(o.Structs))
	copy(structs, o.Structs)
	return &Native{Path: o.Path, Funcs: funcs, Structs: structs}
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

	// Enforce the embedder's allow-list before attempting any OS-level open.
	// An empty allow-list (the default) blocks every path so that scripts
	// cannot load arbitrary shared libraries without the host application
	// explicitly opting in.
	if !isNativePathAllowed(o.Path) {
		return nil, fmt.Errorf(
			"native: path %q is not in the allow-list; "+
				"register it with vm.AllowNativePath before running scripts",
			o.Path,
		)
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
		bound, bindErr := o.buildNativeBinding(handle, spec)
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

// nativeArgPin captures objects that must remain alive for the duration of a
// native call as well as an optional post-call hook (used by struct-pointer
// out-parameters to copy mutated fields back into the user-supplied Map).
type nativeArgPin struct {
	keepAlive any
	postCall  func() error
}

// structReflectType returns the cached reflect.Type for the struct at idx,
// building it on demand. Mutual cycles between struct types are not
// supported; recursive references will return an error.
func (o *Native) structReflectType(idx int) (reflect.Type, error) {
	if idx < 0 || idx >= len(o.Structs) {
		return nil, fmt.Errorf("invalid struct index %d", idx)
	}
	if o.structTypes == nil {
		o.structTypes = make([]reflect.Type, len(o.Structs))
	}
	if t := o.structTypes[idx]; t != nil {
		return t, nil
	}
	spec := o.Structs[idx]
	fields := make([]reflect.StructField, 0, len(spec.Fields))
	for _, f := range spec.Fields {
		var ft reflect.Type
		if f.Kind == NativeStruct {
			if f.StructIdx == idx {
				return nil, fmt.Errorf("struct %q references itself by value", spec.Name)
			}
			nt, err := o.structReflectType(f.StructIdx)
			if err != nil {
				return nil, err
			}
			ft = nt
		} else {
			ft = f.Kind.goType()
			if ft == nil {
				return nil, fmt.Errorf("struct %q field %q has invalid type", spec.Name, f.Name)
			}
		}
		fields = append(fields, reflect.StructField{
			Name: nativeFieldGoName(f.Name),
			Type: ft,
		})
	}
	t := reflect.StructOf(fields)
	o.structTypes[idx] = t
	return t, nil
}

// nativeFieldGoName converts a (possibly lowercase) rumo field name into a
// valid exported Go identifier so reflect.StructOf will accept it. Names
// already starting with an uppercase letter are returned unchanged; the rest
// are prefixed with `F_` so collisions remain unlikely.
func nativeFieldGoName(name string) string {
	if name == "" {
		return "F_"
	}
	r := name[0]
	if r >= 'A' && r <= 'Z' {
		return name
	}
	return "F_" + name
}

// mapToStruct populates a settable struct reflect.Value from a rumo Map,
// converting each declared field via rumoToNativeArg (no pinning is needed
// because struct fields are bare scalars copied into the struct memory).
func (o *Native) mapToStruct(m *Map, structIdx int, dst reflect.Value) error {
	spec := o.Structs[structIdx]
	for i, f := range spec.Fields {
		v, ok := m.Value[f.Name]
		if !ok {
			v = UndefinedValue
		}
		fv := dst.Field(i)
		if f.Kind == NativeStruct {
			child, ok := v.(*Map)
			if !ok {
				return fmt.Errorf("field %q: expected map for struct %q, got %s",
					f.Name, o.Structs[f.StructIdx].Name, v.TypeName())
			}
			if err := o.mapToStruct(child, f.StructIdx, fv); err != nil {
				return fmt.Errorf("field %q: %w", f.Name, err)
			}
			continue
		}
		rv, _, err := rumoToNativeArg(v, f.Kind)
		if err != nil {
			return fmt.Errorf("field %q: %w", f.Name, err)
		}
		fv.Set(rv.Convert(fv.Type()))
	}
	return nil
}

// structToMap converts a struct reflect.Value (or pointer to struct) into a
// fresh rumo *Map.
func (o *Native) structToMap(src reflect.Value, structIdx int) (*Map, error) {
	if src.Kind() == reflect.Pointer {
		if src.IsNil() {
			return &Map{Value: map[string]Object{}}, nil
		}
		src = src.Elem()
	}
	spec := o.Structs[structIdx]
	out := make(map[string]Object, len(spec.Fields))
	for i, f := range spec.Fields {
		fv := src.Field(i)
		if f.Kind == NativeStruct {
			nested, err := o.structToMap(fv, f.StructIdx)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", f.Name, err)
			}
			out[f.Name] = nested
			continue
		}
		obj, err := nativeResultToRumo(fv, f.Kind)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", f.Name, err)
		}
		out[f.Name] = obj
	}
	return &Map{Value: out}, nil
}

// updateMapFromStruct mutates an existing *Map in place to mirror the field
// values of a struct reflect.Value. Used by `*Struct` pointer parameters to
// implement out-parameter semantics after the C call returns.
func (o *Native) updateMapFromStruct(m *Map, src reflect.Value, structIdx int) error {
	if src.Kind() == reflect.Pointer {
		src = src.Elem()
	}
	spec := o.Structs[structIdx]
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Value == nil {
		m.Value = make(map[string]Object, len(spec.Fields))
	}
	for i, f := range spec.Fields {
		fv := src.Field(i)
		if f.Kind == NativeStruct {
			child, _ := m.Value[f.Name].(*Map)
			if child == nil {
				child = &Map{Value: map[string]Object{}}
				m.Value[f.Name] = child
			}
			if err := o.updateMapFromStruct(child, fv, f.StructIdx); err != nil {
				return fmt.Errorf("field %q: %w", f.Name, err)
			}
			continue
		}
		obj, err := nativeResultToRumo(fv, f.Kind)
		if err != nil {
			return fmt.Errorf("field %q: %w", f.Name, err)
		}
		m.Value[f.Name] = obj
	}
	return nil
}

// buildNativeBinding resolves the symbol using dlsym and wraps it in a
// *BuiltinFunction that converts rumo arguments <-> Go values, invokes the
// purego-registered function and converts the result back.
func (o *Native) buildNativeBinding(handle uintptr, spec NativeFuncSpec) (*BuiltinFunction, error) {
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
		if p == NativeStruct {
			st, err := o.structReflectType(spec.ParamStructIdx[i])
			if err != nil {
				return nil, fmt.Errorf("parameter %d: %w", i, err)
			}
			if spec.ParamPointer[i] {
				// purego treats *Struct as a uintptr at the ABI level, but
				// registering the actual pointer type is more portable; we
				// still hand it a uintptr converted from unsafe.Pointer.
				in[i] = reflect.TypeOf(uintptr(0))
			} else {
				in[i] = st
			}
			continue
		}
		in[i] = p.goType()
	}
	var out []reflect.Type
	if spec.Return != NativeVoid {
		switch spec.Return {
		case NativeStruct:
			st, err := o.structReflectType(spec.ReturnStructIdx)
			if err != nil {
				return nil, fmt.Errorf("return: %w", err)
			}
			if spec.ReturnPointer {
				out = []reflect.Type{reflect.TypeOf(uintptr(0))}
			} else {
				out = []reflect.Type{st}
			}
		default:
			out = []reflect.Type{spec.Return.goType()}
		}
	}
	fnType := reflect.FuncOf(in, out, false)

	// allocate a pointer to a nil function value and let purego populate it
	fnPtrValue := reflect.New(fnType)
	purego.RegisterLibFunc(fnPtrValue.Interface(), handle, spec.Name)
	fnValue := fnPtrValue.Elem()

	params := append([]NativeKind(nil), spec.Params...)
	paramStructIdx := append([]int(nil), spec.ParamStructIdx...)
	paramPointer := append([]bool(nil), spec.ParamPointer...)
	ret := spec.Return
	retStructIdx := spec.ReturnStructIdx
	retPointer := spec.ReturnPointer
	name := spec.Name

	return &BuiltinFunction{
		Name: name,
		Value: func(_ context.Context, args ...Object) (Object, error) {
			if len(args) != len(params) {
				return nil, fmt.Errorf("native %s: wrong number of arguments: want=%d got=%d",
					name, len(params), len(args))
			}

			var keepAlive []any
			var postCalls []func() error

			callArgs := make([]reflect.Value, len(args))
			for i, arg := range args {
				if params[i] == NativeStruct {
					rv, pin, err := o.rumoToNativeStructArg(arg, paramStructIdx[i], paramPointer[i])
					if err != nil {
						return nil, fmt.Errorf("native %s: argument %d: %w", name, i, err)
					}
					callArgs[i] = rv
					if pin != nil {
						if pin.keepAlive != nil {
							keepAlive = append(keepAlive, pin.keepAlive)
						}
						if pin.postCall != nil {
							postCalls = append(postCalls, pin.postCall)
						}
					}
					continue
				}
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
			runtime.KeepAlive(keepAlive)

			// out-parameter copy-back for *Struct args
			for _, fn := range postCalls {
				if err := fn(); err != nil {
					return nil, fmt.Errorf("native %s: %w", name, err)
				}
			}

			if ret == NativeVoid {
				return UndefinedValue, nil
			}
			if ret == NativeStruct {
				return o.nativeStructResultToRumo(results[0], retStructIdx, retPointer)
			}
			return nativeResultToRumo(results[0], ret)
		},
	}, nil
}

// rumoToNativeStructArg converts a rumo Map into either a struct value or a
// pointer-sized integer suitable for passing to a C function. When pointer is
// true the returned pin's postCall hook is responsible for copying mutated
// struct fields back into the original Map (out-parameter semantics).
func (o *Native) rumoToNativeStructArg(arg Object, structIdx int, pointer bool) (reflect.Value, *nativeArgPin, error) {
	st, err := o.structReflectType(structIdx)
	if err != nil {
		return reflect.Value{}, nil, err
	}

	if pointer {
		if _, isUndef := arg.(*Undefined); isUndef {
			return reflect.ValueOf(uintptr(0)), nil, nil
		}
		m, ok := arg.(*Map)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf(
				"expected map for *%s, got %s", o.Structs[structIdx].Name, arg.TypeName())
		}
		ptr := reflect.New(st) // *T
		if err := o.mapToStruct(m, structIdx, ptr.Elem()); err != nil {
			return reflect.Value{}, nil, err
		}
		raw := ptr.Pointer() // returns uintptr; pin via keepAlive below
		pin := &nativeArgPin{
			keepAlive: ptr.Interface(),
			postCall: func() error {
				return o.updateMapFromStruct(m, ptr.Elem(), structIdx)
			},
		}
		return reflect.ValueOf(raw), pin, nil
	}

	// by-value
	m, ok := arg.(*Map)
	if !ok {
		return reflect.Value{}, nil, fmt.Errorf(
			"expected map for struct %s, got %s", o.Structs[structIdx].Name, arg.TypeName())
	}
	dst := reflect.New(st).Elem()
	if err := o.mapToStruct(m, structIdx, dst); err != nil {
		return reflect.Value{}, nil, err
	}
	return dst, nil, nil
}

// nativeStructResultToRumo converts a struct return value into a rumo *Map.
// For pointer returns the underlying memory is owned by the C side; rumo
// dereferences it once into a fresh Map and never retains the pointer.
func (o *Native) nativeStructResultToRumo(v reflect.Value, structIdx int, pointer bool) (Object, error) {
	if pointer {
		raw := uintptr(v.Uint())
		if raw == 0 {
			return UndefinedValue, nil
		}
		st, err := o.structReflectType(structIdx)
		if err != nil {
			return nil, err
		}
		ptr := reflect.NewAt(st, unsafe.Pointer(raw))
		return o.structToMap(ptr.Elem(), structIdx)
	}
	return o.structToMap(v, structIdx)
}

// rumoToNativeArg converts a rumo Object to a reflect.Value typed exactly as
// purego expects for a single parameter.  The second return value (if
// non-nil) is an object that must be kept alive for the duration of the C
// call — callers are responsible for doing so.
func rumoToNativeArg(o Object, kind NativeKind) (reflect.Value, any, error) {
	switch kind {
	case NativeByte:
		n, ok := ToInt64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected byte, got %s", o.TypeName())
		}
		return reflect.ValueOf(byte(n)), nil, nil
	case NativeInt8:
		n, ok := ToInt64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected int8, got %s", o.TypeName())
		}
		return reflect.ValueOf(int8(n)), nil, nil
	case NativeUint8:
		n, ok := ToUint64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected uint8, got %s", o.TypeName())
		}
		return reflect.ValueOf(uint8(n)), nil, nil
	case NativeInt16:
		n, ok := ToInt64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected int16, got %s", o.TypeName())
		}
		return reflect.ValueOf(int16(n)), nil, nil
	case NativeUint16:
		n, ok := ToUint64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected uint16, got %s", o.TypeName())
		}
		return reflect.ValueOf(uint16(n)), nil, nil
	case NativeInt64:
		n, ok := ToInt64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected int, got %s", o.TypeName())
		}
		return reflect.ValueOf(n), nil, nil
	case NativeUint64:
		n, ok := ToUint64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected uint/int, got %s", o.TypeName())
		}
		return reflect.ValueOf(n), nil, nil
	case NativeRune:
		r, ok := ToRune(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected rune, got %s", o.TypeName())
		}
		return reflect.ValueOf(r), nil, nil
	case NativeBool:
		b, ok := ToBool(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected bool, got %s", o.TypeName())
		}
		return reflect.ValueOf(b), nil, nil
	case NativeFloat64:
		f, ok := ToFloat64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected double, got %s", o.TypeName())
		}
		return reflect.ValueOf(f), nil, nil
	case NativeFloat32:
		f, ok := ToFloat64(o)
		if !ok {
			return reflect.Value{}, nil, fmt.Errorf("expected float, got %s", o.TypeName())
		}
		return reflect.ValueOf(float32(f)), nil, nil
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
	case NativeByte:
		return &Byte{Value: byte(v.Uint())}, nil
	case NativeInt8:
		return &Int8{Value: int8(v.Int())}, nil
	case NativeUint8:
		return &Uint8{Value: uint8(v.Uint())}, nil
	case NativeInt16:
		return &Int16{Value: int16(v.Int())}, nil
	case NativeUint16:
		return &Uint16{Value: uint16(v.Uint())}, nil
	case NativeInt64:
		return NewInt(v.Int()), nil
	case NativeUint64:
		return &Uint64{Value: v.Uint()}, nil
	case NativeRune:
		return &Char{Value: rune(v.Int())}, nil
	case NativeBool:
		if v.Bool() {
			return TrueValue, nil
		}
		return FalseValue, nil
	case NativeFloat64:
		return &Float64{Value: v.Float()}, nil
	case NativeFloat32:
		return &Float32{Value: float32(v.Float())}, nil
	case NativeString:
		// purego already copied the C string into Go memory.
		return &String{Value: v.String()}, nil
	case NativePtr:
		return &Ptr{Value: unsafe.Pointer(uintptr(v.Uint()))}, nil
	case NativeBytes:
		// Bytes return is ambiguous (length unknown) - expose as a raw ptr.
		return &Ptr{Value: unsafe.Pointer(uintptr(v.Uint()))}, nil
	}
	return nil, fmt.Errorf("unsupported native return kind %s", kind)
}
