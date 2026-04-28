package vm

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/malivvan/rumo/vm/parser"
	"github.com/malivvan/rumo/vm/token"
)

var (
	// TrueValue represents a true value.
	TrueValue Object = &Bool{value: true}

	// FalseValue represents a false value.
	FalseValue Object = &Bool{value: false}

	// UndefinedValue represents an undefined value.
	UndefinedValue Object = &Undefined{}
)

// boolObject returns TrueValue if b is true, FalseValue otherwise.
func boolObject(b bool) Object {
	if b {
		return TrueValue
	}
	return FalseValue
}

// Small integer cache for interning [-128, 127] to avoid per-op heap
// allocations in tight arithmetic loops (issue 4.8).
const (
	intCacheMin = -128
	intCacheMax = 127
)

var intCache [intCacheMax - intCacheMin + 1]*Int

func init() {
	for i := range intCache {
		intCache[i] = &Int{Value: int64(i + intCacheMin)}
	}
}

// NewInt returns an *Int for v. For v in [-128, 127] a pre-allocated cached
// instance is returned so that arithmetic-heavy scripts do not churn the GC.
func NewInt(v int64) *Int {
	if v >= intCacheMin && v <= intCacheMax {
		return intCache[v-intCacheMin]
	}
	return &Int{Value: v}
}

// Object represents an object in the VM.
type Object interface {
	// TypeName should return the name of the type.
	TypeName() string

	// String should return a string representation of the type's value.
	String() string

	// BinaryOp should return another object that is the result of a given
	// binary operator and a right-hand side object. If BinaryOp returns an
	// error, the VM will treat it as a run-time error.
	BinaryOp(op token.Token, rhs Object) (Object, error)

	// IsFalsy should return true if the value of the type should be considered
	// as falsy.
	IsFalsy() bool

	// Equals should return true if the value of the type should be considered
	// as equal to the value of another object.
	Equals(another Object) bool

	// Copy should return a copy of the type (and its value). Copy function
	// will be used for copy() builtin function which is expected to deep-copy
	// the values generally.
	Copy() Object

	// IndexGet should take an index Object and return a result Object or an
	// error for indexable objects. Indexable is an object that can take an
	// index and return an object. If error is returned, the runtime will treat
	// it as a run-time error and ignore returned value. If Object is not
	// indexable, ErrNotIndexable should be returned as error. If nil is
	// returned as value, it will be converted to UndefinedToken value by the
	// runtime.
	IndexGet(index Object) (value Object, err error)

	// IndexSet should take an index Object and a value Object for index
	// assignable objects. Index assignable is an object that can take an index
	// and a value on the left-hand side of the assignment statement. If Object
	// is not index assignable, ErrNotIndexAssignable should be returned as
	// error. If an error is returned, it will be treated as a run-time error.
	IndexSet(index, value Object) error

	// Iterate should return an Iterator for the type.
	Iterate() Iterator

	// CanIterate should return whether the Object can be Iterated.
	CanIterate() bool

	// Call should take an arbitrary number of arguments and returns a return
	// value and/or an error, which the VM will consider as a run-time error.
	Call(ctx context.Context, args ...Object) (ret Object, err error)

	// CanCall should return whether the Object can be Called.
	CanCall() bool
}

// ObjectImpl represents a default Object Implementation. To defined a new
// value type, one can embed ObjectImpl in their type declarations to avoid
// implementing all non-significant methods. TypeName() and String() methods
// still need to be implemented.
type ObjectImpl struct {
}

// TypeName returns the name of the type.
func (o *ObjectImpl) TypeName() string {
	return "unknown"
}

func (o *ObjectImpl) String() string {
	return ""
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *ObjectImpl) BinaryOp(_ token.Token, _ Object) (Object, error) {
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *ObjectImpl) Copy() Object {
	return nil
}

// IsFalsy returns true if the value of the type is falsy.
func (o *ObjectImpl) IsFalsy() bool {
	return false
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *ObjectImpl) Equals(x Object) bool {
	return o == x
}

// IndexGet returns an element at a given index.
func (o *ObjectImpl) IndexGet(_ Object) (res Object, err error) {
	return nil, ErrNotIndexable
}

// IndexSet sets an element at a given index.
func (o *ObjectImpl) IndexSet(_, _ Object) (err error) {
	return ErrNotIndexAssignable
}

// Iterate returns an iterator.
func (o *ObjectImpl) Iterate() Iterator {
	return nil
}

// CanIterate returns whether the Object can be Iterated.
func (o *ObjectImpl) CanIterate() bool {
	return false
}

// Call takes an arbitrary number of arguments and returns a return value
// and/or an error.
func (o *ObjectImpl) Call(_ context.Context, _ ...Object) (ret Object, err error) {
	return nil, nil
}

// CanCall returns whether the Object can be Called.
func (o *ObjectImpl) CanCall() bool {
	return false
}

// Array represents an array of objects.
type Array struct {
	ObjectImpl
	mu    sync.RWMutex
	Value []Object
}

// TypeName returns the name of the type.
func (o *Array) TypeName() string {
	return "array"
}

func (o *Array) String() string {
	return o.stringWithVisited(make(map[uintptr]bool))
}

func (o *Array) stringWithVisited(vis map[uintptr]bool) string {
	key := uintptr(unsafe.Pointer(o))
	if vis[key] {
		return "[...]"
	}
	vis[key] = true
	defer delete(vis, key)
	o.mu.RLock()
	snap := make([]Object, len(o.Value))
	copy(snap, o.Value)
	o.mu.RUnlock()
	var elements []string
	for _, e := range snap {
		elements = append(elements, elemString(e, vis))
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Array) BinaryOp(op token.Token, rhs Object) (Object, error) {
	if rhs, ok := rhs.(*Array); ok {
		switch op {
		case token.Add:
			o.mu.RLock()
			defer o.mu.RUnlock()
			rhs.mu.RLock()
			defer rhs.mu.RUnlock()
			if len(rhs.Value) == 0 {
				return o, nil
			}
			combined := make([]Object, len(o.Value)+len(rhs.Value))
			copy(combined, o.Value)
			copy(combined[len(o.Value):], rhs.Value)
			return &Array{Value: combined}, nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Array) Copy() Object {
	return o.copyWithMemo(make(map[uintptr]Object))
}

func (o *Array) copyWithMemo(memo map[uintptr]Object) Object {
	key := uintptr(unsafe.Pointer(o))
	if existing, ok := memo[key]; ok {
		return existing
	}
	o.mu.RLock()
	snap := make([]Object, len(o.Value))
	copy(snap, o.Value)
	o.mu.RUnlock()
	c := &Array{Value: make([]Object, len(snap))}
	memo[key] = c // register before recursing to break cycles
	for i, elem := range snap {
		c.Value[i] = copyElem(elem, memo)
	}
	return c
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Array) IsFalsy() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.Value) == 0
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Array) Equals(x Object) bool {
	return o.equalsWithVisited(x, make(map[[2]uintptr]bool))
}

func (o *Array) equalsWithVisited(xObj Object, vis map[[2]uintptr]bool) bool {
	oPtr := uintptr(unsafe.Pointer(o))
	var xArr *Array
	var xIArr *ImmutableArray
	var xPtr uintptr
	switch x := xObj.(type) {
	case *Array:
		xArr = x
		xPtr = uintptr(unsafe.Pointer(x))
	case *ImmutableArray:
		xIArr = x
		xPtr = uintptr(unsafe.Pointer(x))
	default:
		return false
	}
	lo, hi := oPtr, xPtr
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := [2]uintptr{lo, hi}
	if vis[pairKey] {
		return true // assume equal on cycle
	}
	vis[pairKey] = true
	o.mu.RLock()
	defer o.mu.RUnlock()
	var xVal []Object
	if xArr != nil {
		if xArr != o {
			xArr.mu.RLock()
			defer xArr.mu.RUnlock()
		}
		xVal = xArr.Value
	} else {
		xVal = xIArr.Value
	}
	if len(o.Value) != len(xVal) {
		return false
	}
	for i, e := range o.Value {
		if !equalsElem(e, xVal[i], vis) {
			return false
		}
	}
	return true
}

// IndexGet returns an element at a given index.
func (o *Array) IndexGet(index Object) (res Object, err error) {
	intIdx, ok := index.(*Int)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	idxVal := int(intIdx.Value)
	if idxVal < 0 || idxVal >= len(o.Value) {
		res = UndefinedValue
		return
	}
	res = o.Value[idxVal]
	return
}

// IndexSet sets an element at a given index.
func (o *Array) IndexSet(index, value Object) (err error) {
	intIdx, ok := ToInt(index)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if intIdx < 0 || intIdx >= len(o.Value) {
		err = ErrIndexOutOfBounds
		return
	}
	o.Value[intIdx] = value
	return nil
}

// Iterate creates an array iterator. It snapshots the current slice so the
// iterator is not affected by concurrent mutations.
func (o *Array) Iterate() Iterator {
	o.mu.RLock()
	snapshot := make([]Object, len(o.Value))
	copy(snapshot, o.Value)
	o.mu.RUnlock()
	return &ArrayIterator{
		v: snapshot,
		l: len(snapshot),
	}
}

// CanIterate returns whether the Object can be Iterated.
func (o *Array) CanIterate() bool {
	return true
}

// Bool represents a boolean value.
type Bool struct {
	ObjectImpl

	// this is intentionally non-public to force using objects.TrueValue and
	// FalseValue always
	value bool
}

func (o *Bool) String() string {
	if o.value {
		return "true"
	}

	return "false"
}

// TypeName returns the name of the type.
func (o *Bool) TypeName() string {
	return "bool"
}

// Copy returns a copy of the type.
func (o *Bool) Copy() Object {
	return o
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Bool) IsFalsy() bool {
	return !o.value
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Bool) Equals(x Object) bool {
	return o == x
}

// BuiltinFunction represents a builtin function.
type BuiltinFunction struct {
	ObjectImpl
	Name  string
	Value CallableFunc
}

// TypeName returns the name of the type.
func (o *BuiltinFunction) TypeName() string {
	return "builtin-function:" + o.Name
}

func (o *BuiltinFunction) String() string {
	return "<builtin-function>"
}

// Copy returns a copy of the type.
func (o *BuiltinFunction) Copy() Object {
	return &BuiltinFunction{Name: o.Name, Value: o.Value}
}

// Equals returns true if the value of the type is equal to the value of
// another object. Two BuiltinFunction instances are equal when they share
// the same non-empty name, which uniquely identifies a builtin in the registry.
func (o *BuiltinFunction) Equals(x Object) bool {
	t, ok := x.(*BuiltinFunction)
	if !ok {
		return false
	}
	return o.Name != "" && o.Name == t.Name
}

// Call executes a builtin function.
func (o *BuiltinFunction) Call(ctx context.Context, args ...Object) (Object, error) {
	return o.Value(ctx, args...)
}

// CanCall returns whether the Object can be Called.
func (o *BuiltinFunction) CanCall() bool {
	return true
}

// BuiltinModule is an importable module that's written in Go.
type BuiltinModule struct {
	Attrs map[string]Object
}

// Import returns an immutable map for the module.
func (m *BuiltinModule) Import(moduleName string) (interface{}, error) {
	return m.AsImmutableMap(moduleName), nil
}

// AsImmutableMap converts builtin module into an immutable map.
func (m *BuiltinModule) AsImmutableMap(moduleName string) *ImmutableMap {
	attrs := make(map[string]Object, len(m.Attrs))
	for k, v := range m.Attrs {
		attrs[k] = v.Copy()
	}
	return &ImmutableMap{Value: attrs, moduleName: moduleName}
}

// Bytes represents a byte array.
type Bytes struct {
	ObjectImpl
	Value []byte
}

func (o *Bytes) String() string {
	return string(o.Value)
}

// TypeName returns the name of the type.
func (o *Bytes) TypeName() string {
	return "bytes"
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Bytes) BinaryOp(op token.Token, rhs Object) (Object, error) {
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *Bytes:
			if len(o.Value)+len(rhs.Value) > DefaultConfig.MaxBytesLen {
				return nil, ErrBytesLimit
			}
			return &Bytes{Value: append(o.Value, rhs.Value...)}, nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Bytes) Copy() Object {
	return &Bytes{Value: append([]byte{}, o.Value...)}
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Bytes) IsFalsy() bool {
	return len(o.Value) == 0
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Bytes) Equals(x Object) bool {
	t, ok := x.(*Bytes)
	if !ok {
		return false
	}
	return bytes.Equal(o.Value, t.Value)
}

// IndexGet returns an element (as Int) at a given index.
func (o *Bytes) IndexGet(index Object) (res Object, err error) {
	intIdx, ok := index.(*Int)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	idxVal := int(intIdx.Value)
	if idxVal < 0 || idxVal >= len(o.Value) {
		res = UndefinedValue
		return
	}
	res = NewInt(int64(o.Value[idxVal]))
	return
}

// Iterate creates a bytes iterator over a snapshot of the current value.
// The snapshot prevents concurrent mutations of Value from affecting the
// in-flight iterator.
func (o *Bytes) Iterate() Iterator {
	cp := append([]byte(nil), o.Value...)
	return &BytesIterator{
		v: cp,
		l: len(cp),
	}
}

// CanIterate returns whether the Object can be Iterated.
func (o *Bytes) CanIterate() bool {
	return true
}

// Char represents a character value.
type Char struct {
	ObjectImpl
	Value rune
}

func (o *Char) String() string {
	return string(o.Value)
}

// TypeName returns the name of the type.
func (o *Char) TypeName() string {
	return "char"
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Char) BinaryOp(op token.Token, rhs Object) (Object, error) {
	switch rhs := rhs.(type) {
	case *Char:
		switch op {
		case token.Add:
			r := o.Value + rhs.Value
			if r == o.Value {
				return o, nil
			}
			return &Char{Value: r}, nil
		case token.Sub:
			r := o.Value - rhs.Value
			if r == o.Value {
				return o, nil
			}
			return &Char{Value: r}, nil
		case token.Less:
			if o.Value < rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.Value > rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.Value <= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.Value >= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case *Int:
		switch op {
		case token.Add:
			r := o.Value + rune(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return &Char{Value: r}, nil
		case token.Sub:
			r := o.Value - rune(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return &Char{Value: r}, nil
		case token.Less:
			if int64(o.Value) < rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if int64(o.Value) > rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if int64(o.Value) <= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if int64(o.Value) >= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Char) Copy() Object {
	return &Char{Value: o.Value}
}

// IsFalsy returns true if the value of the type is falsy.
// Char is always truthy: the NUL character ('\x00') is still a valid character
// value, so the mere presence of a Char object means "I have a character".
func (o *Char) IsFalsy() bool {
	return false
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Char) Equals(x Object) bool {
	t, ok := x.(*Char)
	if !ok {
		return false
	}
	return o.Value == t.Value
}

// CompiledFunction represents a compiled function.
type CompiledFunction struct {
	ObjectImpl
	Instructions  []byte
	NumLocals     int // number of local variables (including function parameters)
	NumParameters int
	VarArgs       bool
	SourceMap     map[int]parser.Pos
	Free          []*ObjectPtr

	// spawnOnce guards the one-time computation of spawnAlias.
	// spawnAlias[i] = j means Free[i] and Free[j] reference the same
	// *ObjectPtr cell (j ≤ i); j == i means the slot is canonical (unique).
	// Cached so that isolateClosureFree avoids rebuilding the dedup map on
	// every goroutine spawn (issue 4.3).
	spawnOnce  sync.Once
	spawnAlias []int
}

// TypeName returns the name of the type.
func (o *CompiledFunction) TypeName() string {
	return "compiled-function"
}

func (o *CompiledFunction) String() string {
	return "<compiled-function>"
}

// Copy returns a copy of the type.
func (o *CompiledFunction) Copy() Object {
	return &CompiledFunction{
		Instructions:  append([]byte{}, o.Instructions...),
		NumLocals:     o.NumLocals,
		NumParameters: o.NumParameters,
		VarArgs:       o.VarArgs,
		SourceMap:     o.SourceMap,
		Free:          append([]*ObjectPtr{}, o.Free...), // DO NOT Copy() of elements; these are variable pointers
	}
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *CompiledFunction) Equals(_ Object) bool {
	return false
}

// SourcePos returns the source position of the instruction at ip.
func (o *CompiledFunction) SourcePos(ip int) parser.Pos {
	for ip >= 0 {
		if p, ok := o.SourceMap[ip]; ok {
			return p
		}
		ip--
	}
	return parser.NoPos
}

// CanCall returns whether the Object can be Called.
func (o *CompiledFunction) CanCall() bool {
	return true
}

// Error represents an error value.
type Error struct {
	ObjectImpl
	Value Object
}

// TypeName returns the name of the type.
func (o *Error) TypeName() string {
	return "error"
}

func (o *Error) String() string {
	if o.Value != nil {
		return fmt.Sprintf("error: %s", o.Value.String())
	}
	return "error"
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Error) IsFalsy() bool {
	return true // error is always false.
}

// Copy returns a copy of the type.
func (o *Error) Copy() Object {
	return &Error{Value: o.Value.Copy()}
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Error) Equals(x Object) bool {
	return o == x // pointer equality
}

// IndexGet returns an element at a given index.
func (o *Error) IndexGet(index Object) (res Object, err error) {
	if strIdx, _ := ToString(index); strIdx != "value" {
		err = ErrInvalidIndexOnError
		return
	}
	res = o.Value
	return
}

// Float32 represents a 32-bit floating point number value (maps to C float).
type Float32 struct {
	ObjectImpl
	Value float32
}

func (o *Float32) String() string {
	return strconv.FormatFloat(float64(o.Value), 'f', -1, 32)
}

// TypeName returns the name of the type.
func (o *Float32) TypeName() string {
	return "float32"
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Float32) BinaryOp(op token.Token, rhs Object) (Object, error) {
	lv := float64(o.Value)
	switch rhs := rhs.(type) {
	case *Float32:
		rv := float64(rhs.Value)
		switch op {
		case token.Add:
			r := lv + rv
			if r == lv {
				return o, nil
			}
			return &Float32{Value: float32(r)}, nil
		case token.Sub:
			r := lv - rv
			if r == lv {
				return o, nil
			}
			return &Float32{Value: float32(r)}, nil
		case token.Mul:
			r := lv * rv
			if r == lv {
				return o, nil
			}
			return &Float32{Value: float32(r)}, nil
		case token.Quo:
			if rv == 0 {
				return nil, ErrDivisionByZero
			}
			r := lv / rv
			if r == lv {
				return o, nil
			}
			return &Float32{Value: float32(r)}, nil
		case token.Less:
			return boolObject(lv < rv), nil
		case token.Greater:
			return boolObject(lv > rv), nil
		case token.LessEq:
			return boolObject(lv <= rv), nil
		case token.GreaterEq:
			return boolObject(lv >= rv), nil
		}
	case *Float64:
		switch op {
		case token.Add:
			return &Float64{Value: lv + rhs.Value}, nil
		case token.Sub:
			return &Float64{Value: lv - rhs.Value}, nil
		case token.Mul:
			return &Float64{Value: lv * rhs.Value}, nil
		case token.Quo:
			if rhs.Value == 0 {
				return nil, ErrDivisionByZero
			}
			return &Float64{Value: lv / rhs.Value}, nil
		case token.Less:
			return boolObject(lv < rhs.Value), nil
		case token.Greater:
			return boolObject(lv > rhs.Value), nil
		case token.LessEq:
			return boolObject(lv <= rhs.Value), nil
		case token.GreaterEq:
			return boolObject(lv >= rhs.Value), nil
		}
	case *Int:
		rv := float64(rhs.Value)
		switch op {
		case token.Add:
			return &Float32{Value: float32(lv + rv)}, nil
		case token.Sub:
			return &Float32{Value: float32(lv - rv)}, nil
		case token.Mul:
			return &Float32{Value: float32(lv * rv)}, nil
		case token.Quo:
			if rv == 0 {
				return nil, ErrDivisionByZero
			}
			return &Float32{Value: float32(lv / rv)}, nil
		case token.Less:
			return boolObject(lv < rv), nil
		case token.Greater:
			return boolObject(lv > rv), nil
		case token.LessEq:
			return boolObject(lv <= rv), nil
		case token.GreaterEq:
			return boolObject(lv >= rv), nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Float32) Copy() Object { return &Float32{Value: o.Value} }

// IsFalsy returns true if the value of the type is falsy.
// Both zero and NaN are considered falsy, consistent with Int where 0 is falsy.
func (o *Float32) IsFalsy() bool { return o.Value == 0 || math.IsNaN(float64(o.Value)) }

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Float32) Equals(x Object) bool {
	t, ok := x.(*Float32)
	if !ok {
		return false
	}
	return o.Value == t.Value
}

// Float64 represents a 64-bit floating point number value (maps to C double).
type Float64 struct {
	ObjectImpl
	Value float64
}

// Float is a backward-compatible alias for Float64.
type Float = Float64

func (o *Float64) String() string {
	return strconv.FormatFloat(o.Value, 'f', -1, 64)
}

// TypeName returns the name of the type.
func (o *Float64) TypeName() string {
	return "float64"
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Float64) BinaryOp(op token.Token, rhs Object) (Object, error) {
	switch rhs := rhs.(type) {
	case *Float64:
		switch op {
		case token.Add:
			r := o.Value + rhs.Value
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Sub:
			r := o.Value - rhs.Value
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Mul:
			r := o.Value * rhs.Value
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Quo:
			if rhs.Value == 0 {
				return nil, ErrDivisionByZero
			}
			r := o.Value / rhs.Value
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Less:
			return boolObject(o.Value < rhs.Value), nil
		case token.Greater:
			return boolObject(o.Value > rhs.Value), nil
		case token.LessEq:
			return boolObject(o.Value <= rhs.Value), nil
		case token.GreaterEq:
			return boolObject(o.Value >= rhs.Value), nil
		}
	case *Float32:
		rv := float64(rhs.Value)
		switch op {
		case token.Add:
			return &Float64{Value: o.Value + rv}, nil
		case token.Sub:
			return &Float64{Value: o.Value - rv}, nil
		case token.Mul:
			return &Float64{Value: o.Value * rv}, nil
		case token.Quo:
			if rv == 0 {
				return nil, ErrDivisionByZero
			}
			return &Float64{Value: o.Value / rv}, nil
		case token.Less:
			return boolObject(o.Value < rv), nil
		case token.Greater:
			return boolObject(o.Value > rv), nil
		case token.LessEq:
			return boolObject(o.Value <= rv), nil
		case token.GreaterEq:
			return boolObject(o.Value >= rv), nil
		}
	case *Int:
		switch op {
		case token.Add:
			r := o.Value + float64(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Sub:
			r := o.Value - float64(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Mul:
			r := o.Value * float64(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Quo:
			if rhs.Value == 0 {
				return nil, ErrDivisionByZero
			}
			r := o.Value / float64(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return &Float64{Value: r}, nil
		case token.Less:
			return boolObject(o.Value < float64(rhs.Value)), nil
		case token.Greater:
			return boolObject(o.Value > float64(rhs.Value)), nil
		case token.LessEq:
			return boolObject(o.Value <= float64(rhs.Value)), nil
		case token.GreaterEq:
			return boolObject(o.Value >= float64(rhs.Value)), nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Float64) Copy() Object {
	return &Float64{Value: o.Value}
}

// IsFalsy returns true if the value of the type is falsy.
// Both zero and NaN are considered falsy, consistent with Int where 0 is falsy.
func (o *Float64) IsFalsy() bool {
	return o.Value == 0 || math.IsNaN(o.Value)
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Float64) Equals(x Object) bool {
	t, ok := x.(*Float64)
	if !ok {
		return false
	}
	return o.Value == t.Value
}

// ImmutableArray represents an immutable array of objects.
type ImmutableArray struct {
	ObjectImpl
	Value []Object
}

// TypeName returns the name of the type.
func (o *ImmutableArray) TypeName() string {
	return "immutable-array"
}

func (o *ImmutableArray) String() string {
	return o.stringWithVisited(make(map[uintptr]bool))
}

func (o *ImmutableArray) stringWithVisited(vis map[uintptr]bool) string {
	key := uintptr(unsafe.Pointer(o))
	if vis[key] {
		return "[...]"
	}
	vis[key] = true
	defer delete(vis, key)
	var elements []string
	for _, e := range o.Value {
		elements = append(elements, elemString(e, vis))
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *ImmutableArray) BinaryOp(op token.Token, rhs Object) (Object, error) {
	if rhs, ok := rhs.(*ImmutableArray); ok {
		switch op {
		case token.Add:
			arr := make([]Object, 0, len(o.Value)+len(rhs.Value))
			arr = append(arr, o.Value...)
			arr = append(arr, rhs.Value...)
			return &Array{Value: arr}, nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *ImmutableArray) Copy() Object {
	return o.copyWithMemo(make(map[uintptr]Object))
}

func (o *ImmutableArray) copyWithMemo(memo map[uintptr]Object) Object {
	key := uintptr(unsafe.Pointer(o))
	if existing, ok := memo[key]; ok {
		return existing
	}
	c := &Array{Value: make([]Object, len(o.Value))}
	memo[key] = c
	for i, elem := range o.Value {
		c.Value[i] = copyElem(elem, memo)
	}
	return c
}

// IsFalsy returns true if the value of the type is falsy.
func (o *ImmutableArray) IsFalsy() bool {
	return len(o.Value) == 0
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *ImmutableArray) Equals(x Object) bool {
	return o.equalsWithVisited(x, make(map[[2]uintptr]bool))
}

func (o *ImmutableArray) equalsWithVisited(xObj Object, vis map[[2]uintptr]bool) bool {
	oPtr := uintptr(unsafe.Pointer(o))
	var xArr *Array
	var xIArr *ImmutableArray
	var xPtr uintptr
	switch x := xObj.(type) {
	case *Array:
		xArr = x
		xPtr = uintptr(unsafe.Pointer(x))
	case *ImmutableArray:
		xIArr = x
		xPtr = uintptr(unsafe.Pointer(x))
	default:
		return false
	}
	lo, hi := oPtr, xPtr
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := [2]uintptr{lo, hi}
	if vis[pairKey] {
		return true
	}
	vis[pairKey] = true
	var xVal []Object
	if xArr != nil {
		xArr.mu.RLock()
		defer xArr.mu.RUnlock()
		xVal = xArr.Value
	} else {
		xVal = xIArr.Value
	}
	if len(o.Value) != len(xVal) {
		return false
	}
	for i, e := range o.Value {
		if !equalsElem(e, xVal[i], vis) {
			return false
		}
	}
	return true
}

// IndexGet returns an element at a given index.
func (o *ImmutableArray) IndexGet(index Object) (res Object, err error) {
	intIdx, ok := index.(*Int)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	idxVal := int(intIdx.Value)
	if idxVal < 0 || idxVal >= len(o.Value) {
		res = UndefinedValue
		return
	}
	res = o.Value[idxVal]
	return
}

// Iterate creates an array iterator.
func (o *ImmutableArray) Iterate() Iterator {
	return &ArrayIterator{
		v: o.Value,
		l: len(o.Value),
	}
}

// CanIterate returns whether the Object can be Iterated.
func (o *ImmutableArray) CanIterate() bool {
	return true
}

// ImmutableMap represents an immutable map object.
type ImmutableMap struct {
	ObjectImpl
	Value      map[string]Object
	moduleName string // set only by AsImmutableMap; not present in user-constructed maps
}

// TypeName returns the name of the type.
func (o *ImmutableMap) TypeName() string {
	return "immutable-map"
}

// ModuleName returns the module name associated with this map, or an empty
// string for user-constructed immutable maps that are not module objects.
func (o *ImmutableMap) ModuleName() string {
	return o.moduleName
}

func (o *ImmutableMap) String() string {
	return o.stringWithVisited(make(map[uintptr]bool))
}

func (o *ImmutableMap) stringWithVisited(vis map[uintptr]bool) string {
	key := uintptr(unsafe.Pointer(o))
	if vis[key] {
		return "{...}"
	}
	vis[key] = true
	defer delete(vis, key)
	var pairs []string
	for k, v := range o.Value {
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, elemString(v, vis)))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

// Copy returns a copy of the type.
func (o *ImmutableMap) Copy() Object {
	return o.copyWithMemo(make(map[uintptr]Object))
}

func (o *ImmutableMap) copyWithMemo(memo map[uintptr]Object) Object {
	key := uintptr(unsafe.Pointer(o))
	if existing, ok := memo[key]; ok {
		return existing
	}
	c := &Map{Value: make(map[string]Object, len(o.Value))}
	memo[key] = c
	for k, v := range o.Value {
		c.Value[k] = copyElem(v, memo)
	}
	return c
}

// IsFalsy returns true if the value of the type is falsy.
func (o *ImmutableMap) IsFalsy() bool {
	return len(o.Value) == 0
}

// IndexGet returns the value for the given key.
func (o *ImmutableMap) IndexGet(index Object) (res Object, err error) {
	strIdx, ok := ToString(index)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	res, ok = o.Value[strIdx]
	if !ok {
		res = UndefinedValue
	}
	return
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *ImmutableMap) Equals(x Object) bool {
	return o.equalsWithVisited(x, make(map[[2]uintptr]bool))
}

func (o *ImmutableMap) equalsWithVisited(xObj Object, vis map[[2]uintptr]bool) bool {
	oPtr := uintptr(unsafe.Pointer(o))
	var xMap *Map
	var xIMap *ImmutableMap
	var xPtr uintptr
	switch x := xObj.(type) {
	case *Map:
		xMap = x
		xPtr = uintptr(unsafe.Pointer(x))
	case *ImmutableMap:
		xIMap = x
		xPtr = uintptr(unsafe.Pointer(x))
	default:
		return false
	}
	lo, hi := oPtr, xPtr
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := [2]uintptr{lo, hi}
	if vis[pairKey] {
		return true
	}
	vis[pairKey] = true
	var xVal map[string]Object
	if xMap != nil {
		xMap.mu.RLock()
		defer xMap.mu.RUnlock()
		xVal = xMap.Value
	} else {
		xVal = xIMap.Value
	}
	if len(o.Value) != len(xVal) {
		return false
	}
	for k, v := range o.Value {
		if !equalsElem(v, xVal[k], vis) {
			return false
		}
	}
	return true
}

// Iterate creates an immutable map iterator.
func (o *ImmutableMap) Iterate() Iterator {
	var keys []string
	for k := range o.Value {
		keys = append(keys, k)
	}
	return &MapIterator{
		v: o.Value,
		k: keys,
		l: len(keys),
	}
}

// CanIterate returns whether the Object can be Iterated.
func (o *ImmutableMap) CanIterate() bool {
	return true
}

// Int represents an integer value.
type Int struct {
	ObjectImpl
	Value int64
}

func (o *Int) String() string {
	return strconv.FormatInt(o.Value, 10)
}

// TypeName returns the name of the type.
func (o *Int) TypeName() string {
	return "int"
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Int) BinaryOp(op token.Token, rhs Object) (Object, error) {
	switch rhs := rhs.(type) {
	case *Int:
		switch op {
		case token.Add:
			r := o.Value + rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Sub:
			r := o.Value - rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Mul:
			r := o.Value * rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Quo:
			if rhs.Value == 0 {
				return nil, ErrDivisionByZero
			}
			r := o.Value / rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Rem:
			if rhs.Value == 0 {
				return nil, ErrDivisionByZero
			}
			r := o.Value % rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.And:
			r := o.Value & rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Or:
			r := o.Value | rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Xor:
			r := o.Value ^ rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.AndNot:
			r := o.Value &^ rhs.Value
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Shl:
			r := o.Value << uint64(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Shr:
			r := o.Value >> uint64(rhs.Value)
			if r == o.Value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Less:
			if o.Value < rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.Value > rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.Value <= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.Value >= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case *Float32:
		lv := float64(o.Value)
		rv := float64(rhs.Value)
		switch op {
		case token.Add:
			return &Float32{Value: float32(lv + rv)}, nil
		case token.Sub:
			return &Float32{Value: float32(lv - rv)}, nil
		case token.Mul:
			return &Float32{Value: float32(lv * rv)}, nil
		case token.Quo:
			return &Float32{Value: float32(lv / rv)}, nil
		case token.Less:
			return boolObject(lv < rv), nil
		case token.Greater:
			return boolObject(lv > rv), nil
		case token.LessEq:
			return boolObject(lv <= rv), nil
		case token.GreaterEq:
			return boolObject(lv >= rv), nil
		}
	case *Float64:
		switch op {
		case token.Add:
			return &Float64{Value: float64(o.Value) + rhs.Value}, nil
		case token.Sub:
			return &Float64{Value: float64(o.Value) - rhs.Value}, nil
		case token.Mul:
			return &Float64{Value: float64(o.Value) * rhs.Value}, nil
		case token.Quo:
			return &Float64{Value: float64(o.Value) / rhs.Value}, nil
		case token.Less:
			return boolObject(float64(o.Value) < rhs.Value), nil
		case token.Greater:
			return boolObject(float64(o.Value) > rhs.Value), nil
		case token.LessEq:
			return boolObject(float64(o.Value) <= rhs.Value), nil
		case token.GreaterEq:
			return boolObject(float64(o.Value) >= rhs.Value), nil
		}
	case *Char:
		switch op {
		case token.Add:
			return &Char{Value: rune(o.Value) + rhs.Value}, nil
		case token.Sub:
			return &Char{Value: rune(o.Value) - rhs.Value}, nil
		case token.Less:
			if o.Value < int64(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.Value > int64(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.Value <= int64(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.Value >= int64(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Int) Copy() Object {
	return NewInt(o.Value)
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Int) IsFalsy() bool {
	return o.Value == 0
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Int) Equals(x Object) bool {
	t, ok := x.(*Int)
	if !ok {
		return false
	}
	return o.Value == t.Value
}

// Map represents a map of objects.
type Map struct {
	ObjectImpl
	mu    sync.RWMutex
	Value map[string]Object
}

// TypeName returns the name of the type.
func (o *Map) TypeName() string {
	return "map"
}

func (o *Map) String() string {
	return o.stringWithVisited(make(map[uintptr]bool))
}

func (o *Map) stringWithVisited(vis map[uintptr]bool) string {
	key := uintptr(unsafe.Pointer(o))
	if vis[key] {
		return "{...}"
	}
	vis[key] = true
	defer delete(vis, key)
	o.mu.RLock()
	snap := make(map[string]Object, len(o.Value))
	for k, v := range o.Value {
		snap[k] = v
	}
	o.mu.RUnlock()
	var pairs []string
	for k, v := range snap {
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, elemString(v, vis)))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

// Copy returns a copy of the type.
func (o *Map) Copy() Object {
	return o.copyWithMemo(make(map[uintptr]Object))
}

func (o *Map) copyWithMemo(memo map[uintptr]Object) Object {
	key := uintptr(unsafe.Pointer(o))
	if existing, ok := memo[key]; ok {
		return existing
	}
	o.mu.RLock()
	snap := make(map[string]Object, len(o.Value))
	for k, v := range o.Value {
		snap[k] = v
	}
	o.mu.RUnlock()
	c := &Map{Value: make(map[string]Object, len(snap))}
	memo[key] = c // register before recursing to break cycles
	for k, v := range snap {
		c.Value[k] = copyElem(v, memo)
	}
	return c
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Map) IsFalsy() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.Value) == 0
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Map) Equals(x Object) bool {
	return o.equalsWithVisited(x, make(map[[2]uintptr]bool))
}

func (o *Map) equalsWithVisited(xObj Object, vis map[[2]uintptr]bool) bool {
	oPtr := uintptr(unsafe.Pointer(o))
	var xMap *Map
	var xIMap *ImmutableMap
	var xPtr uintptr
	switch x := xObj.(type) {
	case *Map:
		xMap = x
		xPtr = uintptr(unsafe.Pointer(x))
	case *ImmutableMap:
		xIMap = x
		xPtr = uintptr(unsafe.Pointer(x))
	default:
		return false
	}
	lo, hi := oPtr, xPtr
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := [2]uintptr{lo, hi}
	if vis[pairKey] {
		return true // assume equal on cycle
	}
	vis[pairKey] = true
	o.mu.RLock()
	defer o.mu.RUnlock()
	var xVal map[string]Object
	if xMap != nil {
		if xMap != o {
			xMap.mu.RLock()
			defer xMap.mu.RUnlock()
		}
		xVal = xMap.Value
	} else {
		xVal = xIMap.Value
	}
	if len(o.Value) != len(xVal) {
		return false
	}
	for k, v := range o.Value {
		if !equalsElem(v, xVal[k], vis) {
			return false
		}
	}
	return true
}

// IndexGet returns the value for the given key.
func (o *Map) IndexGet(index Object) (res Object, err error) {
	strIdx, ok := ToString(index)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	res, ok = o.Value[strIdx]
	if !ok {
		res = UndefinedValue
	}
	return
}

// IndexSet sets the value for the given key.
func (o *Map) IndexSet(index, value Object) (err error) {
	strIdx, ok := ToString(index)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Value[strIdx] = value
	return nil
}

// Iterate creates a map iterator. It snapshots the current keys and values so
// the iterator is not affected by concurrent mutations.
func (o *Map) Iterate() Iterator {
	o.mu.RLock()
	keys := make([]string, 0, len(o.Value))
	vals := make([]Object, 0, len(o.Value))
	for k, v := range o.Value {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	o.mu.RUnlock()
	return &MapIterator{
		v: o.Value,
		k: keys,
		s: vals,
		l: len(keys),
	}
}

// CanIterate returns whether the Object can be Iterated.
func (o *Map) CanIterate() bool {
	return true
}

// ObjectPtr represents a free variable.
type ObjectPtr struct {
	ObjectImpl
	Value *Object
}

func (o *ObjectPtr) String() string {
	return "free-var"
}

// TypeName returns the name of the type.
func (o *ObjectPtr) TypeName() string {
	return "<free-var>"
}

// Copy returns a new ObjectPtr cell whose contents are a deep copy of the
// original. Returning o (the same pointer) would cause two callers — e.g.
// two VMs importing the same BuiltinModule — to share a single mutable cell,
// making a write through one caller visible in the other.
func (o *ObjectPtr) Copy() Object {
	if o.Value == nil {
		return &ObjectPtr{}
	}
	val := (*o.Value).Copy()
	return &ObjectPtr{Value: &val}
}

// IsFalsy returns true if the value of the type is falsy.
func (o *ObjectPtr) IsFalsy() bool {
	return o.Value == nil
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *ObjectPtr) Equals(x Object) bool {
	return o == x
}

// String represents a string value.
type String struct {
	ObjectImpl
	Value   string
	runeStr atomic.Pointer[[]rune]
}

// TypeName returns the name of the type.
func (o *String) TypeName() string {
	return "string"
}

func (o *String) String() string {
	return strconv.Quote(o.Value)
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *String) BinaryOp(op token.Token, rhs Object) (Object, error) {
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *String:
			if len(o.Value)+len(rhs.Value) > DefaultConfig.MaxStringLen {
				return nil, ErrStringLimit
			}
			return &String{Value: o.Value + rhs.Value}, nil
		default:
			rhsStr := rhs.String()
			if len(o.Value)+len(rhsStr) > DefaultConfig.MaxStringLen {
				return nil, ErrStringLimit
			}
			return &String{Value: o.Value + rhsStr}, nil
		}
	case token.Less:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value < rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.LessEq:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value <= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.Greater:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value > rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.GreaterEq:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value >= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, ErrInvalidOperator
}

// IsFalsy returns true if the value of the type is falsy.
func (o *String) IsFalsy() bool {
	return len(o.Value) == 0
}

// Copy returns a copy of the type.
func (o *String) Copy() Object {
	c := &String{Value: o.Value}
	// Propagate the rune cache if it has already been computed so that
	// the work is not needlessly repeated (issue 4.9).
	if p := o.runeStr.Load(); p != nil {
		c.runeStr.Store(p)
	}
	return c
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *String) Equals(x Object) bool {
	t, ok := x.(*String)
	if !ok {
		return false
	}
	return o.Value == t.Value
}

// IndexGet returns a character at a given index.
// It scans the UTF-8 string one rune at a time so that a single index
// access does not need to decode (and allocate) the full rune slice.
func (o *String) IndexGet(index Object) (res Object, err error) {
	intIdx, ok := index.(*Int)
	if !ok {
		err = ErrInvalidIndexType
		return
	}
	idxVal := int(intIdx.Value)
	if idxVal < 0 {
		res = UndefinedValue
		return
	}
	i := 0
	for _, r := range o.Value {
		if i == idxVal {
			res = &Char{Value: r}
			return
		}
		i++
	}
	res = UndefinedValue
	return
}

// Iterate creates a string iterator.
func (o *String) Iterate() Iterator {
	rs := o.runeSlice()
	return &StringIterator{
		v: rs,
		l: len(rs),
	}
}

// runeSlice returns the rune slice for the string, computing and caching it
// on first access in a lock-free, race-free manner (issue 4.9).
func (o *String) runeSlice() []rune {
	if p := o.runeStr.Load(); p != nil {
		return *p
	}
	rs := []rune(o.Value)
	if o.runeStr.CompareAndSwap(nil, &rs) {
		return rs
	}
	return *o.runeStr.Load()
}

// CanIterate returns whether the Object can be Iterated.
func (o *String) CanIterate() bool {
	return true
}

// Time represents a time value.
type Time struct {
	ObjectImpl
	Value time.Time
}

func (o *Time) String() string {
	return o.Value.String()
}

// TypeName returns the name of the type.
func (o *Time) TypeName() string {
	return "time"
}

// BinaryOp returns another object that is the result of a given binary
// operator and a right-hand side object.
func (o *Time) BinaryOp(op token.Token, rhs Object) (Object, error) {
	switch rhs := rhs.(type) {
	case *Int:
		switch op {
		case token.Add: // time + int => time
			if rhs.Value == 0 {
				return o, nil
			}
			return &Time{Value: o.Value.Add(time.Duration(rhs.Value))}, nil
		case token.Sub: // time - int => time
			if rhs.Value == 0 {
				return o, nil
			}
			return &Time{Value: o.Value.Add(time.Duration(-rhs.Value))}, nil
		}
	case *Time:
		switch op {
		case token.Sub: // time - time => int (duration)
			return NewInt(int64(o.Value.Sub(rhs.Value))), nil
		case token.Less: // time < time => bool
			if o.Value.Before(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.Value.After(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.Value.Equal(rhs.Value) || o.Value.Before(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.Value.Equal(rhs.Value) || o.Value.After(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, ErrInvalidOperator
}

// Copy returns a copy of the type.
func (o *Time) Copy() Object {
	return &Time{Value: o.Value}
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Time) IsFalsy() bool {
	return o.Value.IsZero()
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Time) Equals(x Object) bool {
	t, ok := x.(*Time)
	if !ok {
		return false
	}
	return o.Value.Equal(t.Value)
}

// IndexGet exposes time.Time accessor methods as bound *BuiltinFunction
// values, so that scripts can write `t.nanosecond()` / `t.format(layout)`
// even though the freestanding `times.now()` / `times.parse()` helpers
// return a *Time rather than the ImmutableMap produced by the
// `times.Time(x)` TypeRegistration constructor. Method names mirror the
// methods registered on times.Time so the two access styles are
// interchangeable. Unknown keys return UndefinedValue (not an error) to
// match Map / ImmutableMap convention.
func (o *Time) IndexGet(index Object) (Object, error) {
	name, ok := ToString(index)
	if !ok {
		return nil, ErrInvalidIndexType
	}
	t := o.Value
	mkInt := func(n int64) CallableFunc {
		return func(_ context.Context, args ...Object) (Object, error) {
			if len(args) != 0 {
				return nil, ErrWrongNumArguments
			}
			return &Int{Value: n}, nil
		}
	}
	mkStr := func(s string) CallableFunc {
		return func(_ context.Context, args ...Object) (Object, error) {
			if len(args) != 0 {
				return nil, ErrWrongNumArguments
			}
			return &String{Value: s}, nil
		}
	}
	mkBool := func(b bool) CallableFunc {
		return func(_ context.Context, args ...Object) (Object, error) {
			if len(args) != 0 {
				return nil, ErrWrongNumArguments
			}
			return boolObject(b), nil
		}
	}
	switch name {
	case "year":
		return &BuiltinFunction{Name: "time.year", Value: mkInt(int64(t.Year()))}, nil
	case "month":
		return &BuiltinFunction{Name: "time.month", Value: mkInt(int64(t.Month()))}, nil
	case "day":
		return &BuiltinFunction{Name: "time.day", Value: mkInt(int64(t.Day()))}, nil
	case "hour":
		return &BuiltinFunction{Name: "time.hour", Value: mkInt(int64(t.Hour()))}, nil
	case "minute":
		return &BuiltinFunction{Name: "time.minute", Value: mkInt(int64(t.Minute()))}, nil
	case "second":
		return &BuiltinFunction{Name: "time.second", Value: mkInt(int64(t.Second()))}, nil
	case "nanosecond":
		return &BuiltinFunction{Name: "time.nanosecond", Value: mkInt(int64(t.Nanosecond()))}, nil
	case "unix":
		return &BuiltinFunction{Name: "time.unix", Value: mkInt(t.Unix())}, nil
	case "unix_nano":
		return &BuiltinFunction{Name: "time.unix_nano", Value: mkInt(t.UnixNano())}, nil
	case "string":
		return &BuiltinFunction{Name: "time.string", Value: mkStr(t.String())}, nil
	case "location":
		return &BuiltinFunction{Name: "time.location", Value: mkStr(t.Location().String())}, nil
	case "is_zero":
		return &BuiltinFunction{Name: "time.is_zero", Value: mkBool(t.IsZero())}, nil
	case "format":
		return &BuiltinFunction{Name: "time.format", Value: func(_ context.Context, args ...Object) (Object, error) {
			if len(args) != 1 {
				return nil, ErrWrongNumArguments
			}
			layout, ok := ToString(args[0])
			if !ok {
				return nil, ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
			}
			return &String{Value: t.Format(layout)}, nil
		}}, nil
	}
	return UndefinedValue, nil
}

// Undefined represents an undefined value.
type Undefined struct {
	ObjectImpl
}

// TypeName returns the name of the type.
func (o *Undefined) TypeName() string {
	return "undefined"
}

func (o *Undefined) String() string {
	return "<undefined>"
}

// Copy returns a copy of the type.
func (o *Undefined) Copy() Object {
	return o
}

// IsFalsy returns true if the value of the type is falsy.
func (o *Undefined) IsFalsy() bool {
	return true
}

// Equals returns true if the value of the type is equal to the value of
// another object.
func (o *Undefined) Equals(x Object) bool {
	return o == x
}

// IndexGet returns an element at a given index.
func (o *Undefined) IndexGet(_ Object) (Object, error) {
	return UndefinedValue, nil
}

// Iterate creates a map iterator.
func (o *Undefined) Iterate() Iterator {
	return o
}

// CanIterate returns whether the Object can be Iterated.
func (o *Undefined) CanIterate() bool {
	return true
}

// Next returns true if there are more elements to iterate.
func (o *Undefined) Next() bool {
	return false
}

// Key returns the key or index value of the current element.
func (o *Undefined) Key() Object {
	return o
}

// Value returns the value of the current element.
func (o *Undefined) Value() Object {
	return o
}

// --- Cycle-detection helpers (issue 5.9) ---
//
// equalsElem, copyElem, and elemString propagate the per-call visited/memo
// maps into container types so that cyclic Array/Map structures do not cause
// infinite recursion in Equals, Copy, or String.

// equalsElem calls the cycle-safe equalsWithVisited for container types;
// for all other types it delegates to the normal Equals method.
func equalsElem(a, b Object, vis map[[2]uintptr]bool) bool {
	switch a := a.(type) {
	case *Array:
		return a.equalsWithVisited(b, vis)
	case *ImmutableArray:
		return a.equalsWithVisited(b, vis)
	case *Map:
		return a.equalsWithVisited(b, vis)
	case *ImmutableMap:
		return a.equalsWithVisited(b, vis)
	case *StructInstance:
		return a.equalsWithVisited(b, vis)
	}
	return a.Equals(b)
}

// copyElem calls the cycle-safe copyWithMemo for container types;
// for all other types it delegates to the normal Copy method.
func copyElem(o Object, memo map[uintptr]Object) Object {
	switch o := o.(type) {
	case *Array:
		return o.copyWithMemo(memo)
	case *ImmutableArray:
		return o.copyWithMemo(memo)
	case *Map:
		return o.copyWithMemo(memo)
	case *ImmutableMap:
		return o.copyWithMemo(memo)
	case *StructInstance:
		return o.copyWithMemo(memo)
	}
	return o.Copy()
}

// elemString calls the cycle-safe stringWithVisited for container types;
// for all other types it delegates to the normal String method.
func elemString(o Object, vis map[uintptr]bool) string {
	switch o := o.(type) {
	case *Array:
		return o.stringWithVisited(vis)
	case *ImmutableArray:
		return o.stringWithVisited(vis)
	case *Map:
		return o.stringWithVisited(vis)
	case *ImmutableMap:
		return o.stringWithVisited(vis)
	case *StructInstance:
		return o.stringWithVisited(vis)
	}
	return o.String()
}

