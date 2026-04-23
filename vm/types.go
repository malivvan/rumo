package vm

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
//     positional or keyword arguments.
//   - func:   checks arity (with optional varargs) and returns the passed
//     callable, effectively acting as a runtime type assertion.
//   - value:  delegates to the underlying built-in converter (int, float,
//     string, bool, bytes, array, map, ...).
type UserType struct {
	ObjectImpl

	Name string
	Kind UserTypeKind

	// Struct metadata (UserTypeStruct).
	Fields []string

	// Func metadata (UserTypeFunc).
	Params     []string
	VarArgs    bool
	NumParams  int // number of fixed parameters (excluding varargs slot)

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
		return "type " + t.Name + " struct { " + strings.Join(t.Fields, "; ") + " }"
	case UserTypeFunc:
		params := append([]string(nil), t.Params...)
		if t.VarArgs && len(params) > 0 {
			params[len(params)-1] = "..." + params[len(params)-1]
		}
		return "type " + t.Name + " func(" + strings.Join(params, ", ") + ")"
	case UserTypeValue:
		return "type " + t.Name + " " + t.Underlying
	}
	return "type " + t.Name
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
	// Default all fields to undefined.
	for _, f := range t.Fields {
		inst.Values[f] = UndefinedValue
	}
	switch len(args) {
	case 0:
		// zero value
		return inst, nil
	case 1:
		// Single argument may be a map of field -> value for keyword-style
		// initialisation, otherwise fall through to positional handling.
		if m, ok := args[0].(*Map); ok {
			m.mu.RLock()
			defer m.mu.RUnlock()
			for k, v := range m.Value {
				if _, known := inst.Values[k]; !known {
					return nil, fmt.Errorf("type %s: unknown field %q", t.Name, k)
				}
				inst.Values[k] = v
			}
			return inst, nil
		}
		if m, ok := args[0].(*ImmutableMap); ok {
			for k, v := range m.Value {
				if _, known := inst.Values[k]; !known {
					return nil, fmt.Errorf("type %s: unknown field %q", t.Name, k)
				}
				inst.Values[k] = v
			}
			return inst, nil
		}
	}
	if len(args) > len(t.Fields) {
		return nil, fmt.Errorf("type %s: too many arguments (want %d, got %d)",
			t.Name, len(t.Fields), len(args))
	}
	for i, v := range args {
		inst.Values[t.Fields[i]] = v
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
	// For CompiledFunction we can eagerly validate the arity.
	if cf, ok := fn.(*CompiledFunction); ok {
		if t.VarArgs {
			if cf.NumParameters < t.NumParams {
				return nil, fmt.Errorf("type %s: callable takes %d parameter(s), want at least %d",
					t.Name, cf.NumParameters, t.NumParams)
			}
		} else {
			want := t.NumParams
			if cf.VarArgs {
				// A varargs callable with fewer fixed params can still accept
				// our fixed-arity call; don't reject it.
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
	return fn, nil
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	var parts []string
	if s.Type != nil {
		// Preserve declared field order.
		for _, f := range s.Type.Fields {
			parts = append(parts, fmt.Sprintf("%s: %s", f, s.Values[f].String()))
		}
	} else {
		for k, v := range s.Values {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := make(map[string]Object, len(s.Values))
	for k, v := range s.Values {
		c[k] = v.Copy()
	}
	return &StructInstance{Type: s.Type, Values: c}
}

// Equals returns true if s and x are instances of the same user type and
// every field compares equal.
func (s *StructInstance) Equals(x Object) bool {
	other, ok := x.(*StructInstance)
	if !ok {
		return false
	}
	if s.Type != other.Type {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	if len(s.Values) != len(other.Values) {
		return false
	}
	for k, v := range s.Values {
		ov, ok := other.Values[k]
		if !ok || !v.Equals(ov) {
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

// IndexSet assigns a value to an existing field; unknown fields are rejected.
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
