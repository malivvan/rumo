package vm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/malivvan/rumo/vm/token"
)

// UserTypeKind identifies the form of a user-defined type.
type UserTypeKind int

const (
	// UserTypeStruct is declared with `type X struct { ... }`.
	UserTypeStruct UserTypeKind = iota + 1
	// UserTypeFunc is declared with `type X func(...)`.
	UserTypeFunc
	// UserTypeValue is declared with `type X Name` (alias / named value type).
	UserTypeValue
)

// UserType is the runtime representation of a user-defined type introduced by
// a `type` statement.
//
// A UserType is a callable value; invoking it constructs a value of the
// declared type:
//
//   - struct: returns a *StructInstance whose fields are initialised from
//     positional or keyword arguments, each value validated against the
//     declared field type.
//   - func:   wraps the given callable so subsequent invocations validate
//     their arguments (and return value) against the declared signature.
//   - value:  delegates to the underlying built-in converter (int, float,
//     string, bool, bytes, array, map, ...).
type UserType struct {
	ObjectImpl

	Name string
	Kind UserTypeKind

	// Struct metadata (UserTypeStruct). FieldTypes is parallel to Fields.
	Fields     []string
	FieldTypes []string

	// Func metadata (UserTypeFunc). ParamTypes is parallel to Params.
	Params     []string // parameter names (may be empty strings for unnamed)
	ParamTypes []string
	VarArgs    bool
	NumParams  int    // number of fixed parameters (excluding the varargs slot)
	Result     string // return type name; "" = no return

	// Value metadata (UserTypeValue). Underlying is the name of the base
	// type (e.g. "int", "string"). The converter is resolved lazily.
	Underlying string
}

// TypeName returns the name of the type.
func (t *UserType) TypeName() string {
	switch t.Kind {
	case UserTypeStruct:
		return "type:struct:" + t.Name
	case UserTypeFunc:
		return "type:func:" + t.Name
	case UserTypeValue:
		return "type:value:" + t.Name
	}
	return "type:" + t.Name
}

func (t *UserType) String() string {
	switch t.Kind {
	case UserTypeStruct:
		var parts []string
		for i, f := range t.Fields {
			parts = append(parts, f+" "+t.FieldTypes[i])
		}
		return "type " + t.Name + " struct { " + strings.Join(parts, "; ") + " }"
	case UserTypeFunc:
		return "type " + t.Name + " " + t.funcSignature()
	case UserTypeValue:
		return "type " + t.Name + " " + t.Underlying
	}
	return "type " + t.Name
}

// funcSignature returns the `func(...) R` form for UserTypeFunc.
func (t *UserType) funcSignature() string {
	parts := make([]string, len(t.Params))
	for i, p := range t.Params {
		typ := t.ParamTypes[i]
		if t.VarArgs && i == len(t.Params)-1 {
			typ = "..." + typ
		}
		if p == "" {
			parts[i] = typ
		} else {
			parts[i] = p + " " + typ
		}
	}
	sig := "func(" + strings.Join(parts, ", ") + ")"
	if t.Result != "" {
		sig += " " + t.Result
	}
	return sig
}

// Copy returns the same UserType. Type definitions are immutable, so sharing
// is safe.
func (t *UserType) Copy() Object { return t }

// Equals returns true if both receivers refer to the same UserType.
func (t *UserType) Equals(x Object) bool {
	other, ok := x.(*UserType)
	return ok && other == t
}

// IsFalsy — a type is never falsy.
func (t *UserType) IsFalsy() bool { return false }

// CanCall allows the VM to dispatch to this object via OpCall.
func (t *UserType) CanCall() bool { return true }

// Call constructs a value of this type from the given arguments.
func (t *UserType) Call(ctx context.Context, args ...Object) (Object, error) {
	switch t.Kind {
	case UserTypeStruct:
		return t.callStruct(args)
	case UserTypeFunc:
		return t.callFunc(args)
	case UserTypeValue:
		return t.callValue(ctx, args)
	}
	return nil, fmt.Errorf("type %q: invalid kind", t.Name)
}

func (t *UserType) callStruct(args []Object) (Object, error) {
	inst := &StructInstance{
		Type:   t,
		Values: make(map[string]Object, len(t.Fields)),
	}
	// Keyword-style construction: a single map maps field -> value.
	if len(args) == 1 {
		switch m := args[0].(type) {
		case *Map:
			m.mu.RLock()
			defer m.mu.RUnlock()
			return t.fillFromMap(inst, m.Value)
		case *ImmutableMap:
			return t.fillFromMap(inst, m.Value)
		}
	}

	if len(args) > len(t.Fields) {
		return nil, fmt.Errorf("type %s: too many arguments (want %d, got %d)",
			t.Name, len(t.Fields), len(args))
	}

	// Positional initialisation with zero-filling.
	for i, name := range t.Fields {
		declared := t.FieldTypes[i]
		if i < len(args) {
			v, err := coerceToDeclared(t.Name+"."+name, declared, args[i])
			if err != nil {
				return nil, err
			}
			inst.Values[name] = v
		} else {
			inst.Values[name] = zeroForType(declared)
		}
	}
	return inst, nil
}

func (t *UserType) fillFromMap(inst *StructInstance, src map[string]Object) (Object, error) {
	// Fields not mentioned in src take their zero value.
	for i, name := range t.Fields {
		if v, ok := src[name]; ok {
			vv, err := coerceToDeclared(t.Name+"."+name, t.FieldTypes[i], v)
			if err != nil {
				return nil, err
			}
			inst.Values[name] = vv
		} else {
			inst.Values[name] = zeroForType(t.FieldTypes[i])
		}
	}
	// Reject keys that do not correspond to any field.
	known := make(map[string]struct{}, len(t.Fields))
	for _, f := range t.Fields {
		known[f] = struct{}{}
	}
	for k := range src {
		if _, ok := known[k]; !ok {
			return nil, fmt.Errorf("type %s: unknown field %q", t.Name, k)
		}
	}
	return inst, nil
}

func (t *UserType) callFunc(args []Object) (Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("type %s: expected exactly 1 argument (a callable), got %d",
			t.Name, len(args))
	}
	fn := args[0]
	if !fn.CanCall() {
		return nil, fmt.Errorf("type %s: argument is not callable (%s)", t.Name, fn.TypeName())
	}
	// Validate declared arity against known callables.
	if cf, ok := fn.(*CompiledFunction); ok {
		if t.VarArgs {
			if cf.NumParameters < t.NumParams {
				return nil, fmt.Errorf("type %s: callable takes %d parameter(s), want at least %d",
					t.Name, cf.NumParameters, t.NumParams)
			}
		} else {
			want := t.NumParams
			if cf.VarArgs {
				if cf.NumParameters-1 > want {
					return nil, fmt.Errorf("type %s: callable fixed arity %d exceeds declared %d",
						t.Name, cf.NumParameters-1, want)
				}
			} else if cf.NumParameters != want {
				return nil, fmt.Errorf("type %s: callable takes %d parameter(s), want %d",
					t.Name, cf.NumParameters, want)
			}
		}
	}
	// Return a wrapper that validates argument (and return) types on every
	// invocation. The wrapper is a regular BuiltinFunction so it integrates
	// with the VM's existing dispatch machinery.
	typ := t
	return &BuiltinFunction{
		Name: t.Name,
		Value: func(ctx context.Context, callArgs ...Object) (Object, error) {
			if err := typ.validateCallArgs(callArgs); err != nil {
				return nil, err
			}
			res, err := CallFunc(ctx, fn, callArgs...)
			if err != nil {
				return nil, err
			}
			if res == nil {
				res = UndefinedValue
			}
			if typ.Result != "" {
				if !isOfType(typ.Result, res) {
					return nil, fmt.Errorf("type %s: return value type mismatch: want %s, got %s",
						typ.Name, typ.Result, res.TypeName())
				}
			}
			return res, nil
		},
	}, nil
}

func (t *UserType) validateCallArgs(args []Object) error {
	if t.VarArgs {
		if len(args) < t.NumParams {
			return fmt.Errorf("type %s: not enough arguments: want at least %d, got %d",
				t.Name, t.NumParams, len(args))
		}
		for i := 0; i < t.NumParams; i++ {
			if !isOfType(t.ParamTypes[i], args[i]) {
				return fmt.Errorf("type %s: arg %d (%s) type mismatch: want %s, got %s",
					t.Name, i, paramLabel(t.Params, i), t.ParamTypes[i], args[i].TypeName())
			}
		}
		tailType := t.ParamTypes[t.NumParams]
		for i := t.NumParams; i < len(args); i++ {
			if !isOfType(tailType, args[i]) {
				return fmt.Errorf("type %s: varargs element %d type mismatch: want %s, got %s",
					t.Name, i-t.NumParams, tailType, args[i].TypeName())
			}
		}
		return nil
	}

	if len(args) != t.NumParams {
		return fmt.Errorf("type %s: want %d argument(s), got %d",
			t.Name, t.NumParams, len(args))
	}
	for i := 0; i < t.NumParams; i++ {
		if !isOfType(t.ParamTypes[i], args[i]) {
			return fmt.Errorf("type %s: arg %d (%s) type mismatch: want %s, got %s",
				t.Name, i, paramLabel(t.Params, i), t.ParamTypes[i], args[i].TypeName())
		}
	}
	return nil
}

func paramLabel(names []string, i int) string {
	if i < len(names) && names[i] != "" {
		return names[i]
	}
	return fmt.Sprintf("#%d", i)
}

func (t *UserType) callValue(ctx context.Context, args []Object) (Object, error) {
	conv, ok := valueTypeConverter(t.Underlying)
	if !ok {
		return nil, fmt.Errorf("type %s: unknown underlying type %q", t.Name, t.Underlying)
	}
	return conv(ctx, args...)
}

// valueTypeConverter resolves a named underlying type to its built-in
// conversion function. The returned callable has the same semantics as the
// homonymous builtin — `int`, `float`, `string`, etc.
func valueTypeConverter(name string) (CallableFunc, bool) {
	switch name {
	case "int":
		return builtinInt, true
	case "int8":
		return builtinInt8, true
	case "int16":
		return builtinInt16, true
	case "int64":
		return builtinInt64, true
	case "uint":
		return builtinUint, true
	case "uint8":
		return builtinUint8, true
	case "uint16":
		return builtinUint16, true
	case "uint64":
		return builtinUint64, true
	case "byte":
		return builtinByte, true
	case "bool":
		return builtinBool, true
	case "float", "float32":
		return builtinFloat32, true
	case "float64", "double":
		return builtinFloat64, true
	case "char", "rune":
		return builtinChar, true
	case "string":
		return builtinString, true
	case "bytes":
		return builtinBytes, true
	case "error":
		return builtinError, true
	case "ptr":
		return builtinPtr, true
	case "array":
		return builtinArray, true
	case "immutable_array":
		return builtinImmutableArray, true
	case "map":
		return builtinMap, true
	case "immutable_map":
		return builtinImmutableMap, true
	case "time":
		return builtinTime, true
	}
	return nil, false
}

// coerceToDeclared ensures that v is acceptable for a field/parameter of the
// given declared type. Values already of the declared type pass through.
// The special sentinel `undefined` is replaced with the declared zero value
// so that `Point({y: 2})` leaves `x` as `0` (for `x int`). Anything else is
// rejected with a precise error.
func coerceToDeclared(label, typeName string, v Object) (Object, error) {
	if typeName == "" {
		return v, nil
	}
	if _, isUndef := v.(*Undefined); isUndef {
		return zeroForType(typeName), nil
	}
	if isOfType(typeName, v) {
		return v, nil
	}
	return nil, fmt.Errorf("field %s: type mismatch: want %s, got %s",
		label, typeName, v.TypeName())
}

// isOfType returns true if v's runtime type matches the named declared type.
// Unknown type names (including user-defined types not yet resolvable) are
// treated permissively — anything goes.
func isOfType(typeName string, v Object) bool {
	switch typeName {
	case "":
		return true
	case "int", "int64":
		_, ok := v.(*Int)
		return ok
	case "int8":
		_, ok := v.(*Int8)
		return ok
	case "int16":
		_, ok := v.(*Int16)
		return ok
	case "uint":
		_, ok := v.(*Uint)
		return ok
	case "uint8":
		_, ok := v.(*Uint8)
		return ok
	case "uint16":
		_, ok := v.(*Uint16)
		return ok
	case "uint64":
		_, ok := v.(*Uint64)
		return ok
	case "byte":
		_, ok := v.(*Byte)
		return ok
	case "bool":
		_, ok := v.(*Bool)
		return ok
	case "float", "float64", "double":
		_, ok := v.(*Float64)
		return ok
	case "float32":
		_, ok := v.(*Float32)
		return ok
	case "char", "rune":
		_, ok := v.(*Char)
		return ok
	case "string":
		_, ok := v.(*String)
		return ok
	case "bytes":
		_, ok := v.(*Bytes)
		return ok
	case "error":
		_, ok := v.(*Error)
		return ok
	case "ptr":
		_, ok := v.(*Ptr)
		return ok
	case "array":
		_, ok := v.(*Array)
		return ok
	case "immutable_array":
		_, ok := v.(*ImmutableArray)
		return ok
	case "map":
		_, ok := v.(*Map)
		return ok
	case "immutable_map":
		_, ok := v.(*ImmutableMap)
		return ok
	case "time":
		_, ok := v.(*Time)
		return ok
	case "undefined":
		_, ok := v.(*Undefined)
		return ok
	case "any", "object":
		return true
	}
	// Unknown type (user-defined or unsupported): if it's a struct
	// instance carrying that type's name, accept it.
	if si, ok := v.(*StructInstance); ok {
		if si.Type != nil && si.Type.Name == typeName {
			return true
		}
	}
	// Otherwise be permissive — we don't enforce unknown type names at runtime.
	return true
}

// zeroForType returns the Go-like zero value for a named declared type.
// Unknown types default to undefined.
func zeroForType(typeName string) Object {
	switch typeName {
	case "int", "int64":
		return &Int{}
	case "int8":
		return &Int8{}
	case "int16":
		return &Int16{}
	case "uint":
		return &Uint{}
	case "uint8":
		return &Uint8{}
	case "uint16":
		return &Uint16{}
	case "uint64":
		return &Uint64{}
	case "byte":
		return &Byte{}
	case "bool":
		return FalseValue
	case "float", "float64", "double":
		return &Float64{}
	case "float32":
		return &Float32{}
	case "char", "rune":
		return &Char{}
	case "string":
		return &String{}
	case "bytes":
		return &Bytes{}
	case "array":
		return &Array{}
	case "immutable_array":
		return &ImmutableArray{}
	case "map":
		return &Map{Value: make(map[string]Object)}
	case "immutable_map":
		return &ImmutableMap{Value: make(map[string]Object)}
	}
	return UndefinedValue
}

// StructInstance is a single value of a user-defined struct type. It supports
// indexed access by field name (`.x` and `["x"]`), iteration over fields, and
// equality against other instances of the same type.
type StructInstance struct {
	ObjectImpl

	Type   *UserType
	mu     sync.RWMutex
	Values map[string]Object
}

// TypeName returns the name of the type.
func (s *StructInstance) TypeName() string {
	if s.Type != nil {
		return s.Type.Name
	}
	return "struct"
}

func (s *StructInstance) String() string {
	return s.stringWithVisited(make(map[uintptr]bool))
}

func (s *StructInstance) stringWithVisited(vis map[uintptr]bool) string {
	key := uintptr(unsafe.Pointer(s))
	if vis[key] {
		return s.TypeName() + "{...}"
	}
	vis[key] = true
	defer delete(vis, key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var parts []string
	if s.Type != nil {
		for _, f := range s.Type.Fields {
			parts = append(parts, fmt.Sprintf("%s: %s", f, elemString(s.Values[f], vis)))
		}
	} else {
		for k, v := range s.Values {
			parts = append(parts, fmt.Sprintf("%s: %s", k, elemString(v, vis)))
		}
	}
	name := "struct"
	if s.Type != nil {
		name = s.Type.Name
	}
	return name + "{" + strings.Join(parts, ", ") + "}"
}

// Copy returns a deep copy of the instance.
func (s *StructInstance) Copy() Object {
	return s.copyWithMemo(make(map[uintptr]Object))
}

func (s *StructInstance) copyWithMemo(memo map[uintptr]Object) Object {
	key := uintptr(unsafe.Pointer(s))
	if existing, ok := memo[key]; ok {
		return existing
	}
	s.mu.RLock()
	snap := make(map[string]Object, len(s.Values))
	for k, v := range s.Values {
		snap[k] = v
	}
	s.mu.RUnlock()
	c := &StructInstance{Type: s.Type, Values: make(map[string]Object, len(snap))}
	memo[key] = c
	for k, v := range snap {
		c.Values[k] = copyElem(v, memo)
	}
	return c
}

// Equals returns true if s and x are instances of the same user type and
// every field compares equal.
func (s *StructInstance) Equals(x Object) bool {
	return s.equalsWithVisited(x, make(map[[2]uintptr]bool))
}

func (s *StructInstance) equalsWithVisited(xObj Object, vis map[[2]uintptr]bool) bool {
	other, ok := xObj.(*StructInstance)
	if !ok {
		return false
	}
	if s.Type != other.Type {
		return false
	}
	sPtr := uintptr(unsafe.Pointer(s))
	oPtr := uintptr(unsafe.Pointer(other))
	lo, hi := sPtr, oPtr
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := [2]uintptr{lo, hi}
	if vis[pairKey] {
		return true
	}
	vis[pairKey] = true
	s.mu.RLock()
	defer s.mu.RUnlock()
	if other != s {
		other.mu.RLock()
		defer other.mu.RUnlock()
	}
	if len(s.Values) != len(other.Values) {
		return false
	}
	for k, v := range s.Values {
		ov, ok := other.Values[k]
		if !ok || !equalsElem(v, ov, vis) {
			return false
		}
	}
	return true
}

// IsFalsy returns true if all field values are falsy.
func (s *StructInstance) IsFalsy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.Values {
		if !v.IsFalsy() {
			return false
		}
	}
	return true
}

// IndexGet returns the value of the given field.
func (s *StructInstance) IndexGet(index Object) (Object, error) {
	key, ok := ToString(index)
	if !ok {
		return nil, ErrInvalidIndexType
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, known := s.Values[key]
	if !known {
		return nil, fmt.Errorf("type %s: no such field %q", s.TypeName(), key)
	}
	return v, nil
}

// IndexSet assigns a value to an existing field, validating the declared
// type. Unknown fields are rejected.
func (s *StructInstance) IndexSet(index, value Object) error {
	key, ok := ToString(index)
	if !ok {
		return ErrInvalidIndexType
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, known := s.Values[key]; !known {
		return fmt.Errorf("type %s: no such field %q", s.TypeName(), key)
	}
	if s.Type != nil {
		for i, f := range s.Type.Fields {
			if f == key {
				coerced, err := coerceToDeclared(s.Type.Name+"."+key, s.Type.FieldTypes[i], value)
				if err != nil {
					return err
				}
				s.Values[key] = coerced
				return nil
			}
		}
	}
	s.Values[key] = value
	return nil
}

// Iterate produces (fieldName, value) pairs in declared order.
func (s *StructInstance) Iterate() Iterator {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	if s.Type != nil {
		keys = append(keys, s.Type.Fields...)
	} else {
		for k := range s.Values {
			keys = append(keys, k)
		}
	}
	vals := make([]Object, len(keys))
	snap := make(map[string]Object, len(s.Values))
	for i, k := range keys {
		v := s.Values[k]
		if v == nil {
			v = UndefinedValue
		}
		vals[i] = v
		snap[k] = v
	}
	return &MapIterator{v: snap, k: keys, s: vals, l: len(keys)}
}

// CanIterate returns true; struct instances iterate as (name, value) pairs.
func (s *StructInstance) CanIterate() bool { return true }

// BinaryOp supports only equality/inequality via Equals elsewhere. Structs
// have no arithmetic operators.
func (s *StructInstance) BinaryOp(op token.Token, rhs Object) (Object, error) {
	return nil, ErrInvalidOperator
}
