package module

import (
	"context"
	"fmt"
	"time"

	"github.com/malivvan/rumo/vm"
)

// BuiltinModule represents a standard library module.
type BuiltinModule struct {
	export map[string]*Export
	object map[string]vm.Object
}

// Exports returns the export map of the module.
func (m *BuiltinModule) Exports() map[string]*Export {
	return m.export
}

// Objects returns the object map of the module.
func (m *BuiltinModule) Objects() map[string]vm.Object {
	return m.object
}

// NewBuiltin creates a new BuiltinModule with the given name and registers it in the module map. It panics if the name is empty or already exists.
func NewBuiltin() *BuiltinModule {
	m := &BuiltinModule{export: make(map[string]*Export), object: make(map[string]vm.Object)}
	return m
}

// Const x
func (m *BuiltinModule) Const(def string, val any) *BuiltinModule {
	if len(def) == 0 {
		panic("constant def cannot be empty")
	}
	export := ParseExport(def)
	if export == nil {
		panic("invalid export definition: " + def)
	}
	m.object[export.Name] = Const(val)
	m.export[export.Name] = export
	return m
}

// Func registers a function in the module with the given export definition and implementation. It panics if the definition is invalid or the implementation type is unsupported.
func (m *BuiltinModule) Func(def string, impl any) *BuiltinModule {
	if len(def) == 0 {
		panic("function def cannot be empty")
	}
	export := ParseExport(def)
	if export == nil {
		panic("invalid export definition: " + def)
	}
	m.object[export.Name] = &vm.BuiltinFunction{Name: export.Name, Value: Func(impl)}
	m.export[export.Name] = export
	return m
}

// Const x
func Const(val any) vm.Object {
	switch v := val.(type) {
	case int64:
		return &vm.Int{Value: v}
	case time.Month:
		return &vm.Int{Value: int64(v)}
	case time.Duration:
		return &vm.Int{Value: int64(v)}
	case int:
		return &vm.Int{Value: int64(v)}
	case float64:
		return &vm.Float{Value: float64(v)}
	case string:
		return &vm.String{Value: v}
	case bool:
		if v {
			return vm.TrueValue
		}
		return vm.FalseValue
	case []byte:
		return &vm.Bytes{Value: v}
	default:
		panic(fmt.Errorf("unsupported constant type: %T", val))
	}
}

// Func transforms a Go function into a vm.CallableFunc. It panics if the function type is unsupported.
func Func(impl any) vm.CallableFunc {
	switch f := impl.(type) {
	case vm.CallableFunc:
		return f
	case func():
		return funcAR(f)
	case func() int:
		return funcARI(f)
	case func() int64:
		return funcARI(func() int { return int(f()) })
	case func(int64) int64:
		return funcAI64RI64(f)
	case func(int):
		return funcAIR(f)
	case func(int64):
		return funcAIR(func(i int) { f(int64(i)) })
	case func() bool:
		return funcARB(f)
	case func() error:
		return funcARE(f)
	case func() string:
		return funcARS(f)
	case func() (string, error):
		return funcARSE(f)
	case func() ([]byte, error):
		return funcARYE(f)
	case func() float64:
		return funcARF(f)
	case func() []string:
		return funcARSs(f)
	case func() ([]int, error):
		return funcARIsE(f)
	case func(int) []int:
		return funcAIRIs(f)
	case func(float64) float64:
		return funcAFRF(f)
	case func(int) float64:
		return funcAIRF(f)
	case func(float64) int:
		return funcAFRI(f)
	case func(float64, float64) float64:
		return funcAFFRF(f)
	case func(int, float64) float64:
		return funcAIFRF(f)
	case func(float64, int) float64:
		return funcAFIRF(f)
	case func(float64, int) bool:
		return funcAFIRB(f)
	case func(float64) bool:
		return funcAFRB(f)
	case func(string) string:
		return funcASRS(f)
	case func(string) []string:
		return funcASRSs(f)
	case func(string) (string, error):
		return funcASRSE(f)
	case func(string) error:
		return funcASRE(f)
	case func(string, string) error:
		return funcASSRE(f)
	case func(string, string) []string:
		return funcASSRSs(f)
	case func(string, string, int) []string:
		return funcASSIRSs(f)
	case func(string, string) int:
		return funcASSRI(f)
	case func(string, string) string:
		return funcASSRS(f)
	case func(string, string) bool:
		return funcASSRB(f)
	case func(int, int) error:
		return funcAIIRE(f)
	case func([]string, string) string:
		return funcASsSRS(f)
	case func(string, int64) error:
		return funcASI64RE(f)
	case func(string, int) string:
		return funcASIRS(f)
	case func(string, int, int) error:
		return funcASIIRE(f)
	case func([]byte) (int, error):
		return funcAYRIE(f)
	case func([]byte) string:
		return funcAYRS(f)
	case func(string) (int, error):
		return funcASRIE(f)
	case func(string) ([]byte, error):
		return funcASRYE(f)
	case func(int) ([]string, error):
		return funcAIRSsE(f)
	case func(int) string:
		return funcAIRS(f)
	default:
		panic(fmt.Errorf("unsupported function type: %T", impl))
	}
}

// funcAR transform a function of 'func()' signature into CallableFunc type.
func funcAR(fn func()) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		fn()
		return vm.UndefinedValue, nil
	}
}

// funcARI transform a function of 'func() int' signature into CallableFunc
// type.
func funcARI(fn func() int) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return &vm.Int{Value: int64(fn())}, nil
	}
}

// funcARI64 transform a function of 'func() int64' signature into CallableFunc
// type.
func funcARI64(fn func() int64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return &vm.Int{Value: fn()}, nil
	}
}

// funcAI64RI64 transform a function of 'func(int64) int64' signature into
// CallableFunc type.
func funcAI64RI64(fn func(int64) int64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}

		i1, ok := vm.ToInt64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return &vm.Int{Value: fn(i1)}, nil
	}
}

// funcAI64R transform a function of 'func(int64)' signature into CallableFunc
// type.
func funcAI64R(fn func(int64)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}

		i1, ok := vm.ToInt64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		fn(i1)
		return vm.UndefinedValue, nil
	}
}

// funcARB transform a function of 'func() bool' signature into CallableFunc
// type.
func funcARB(fn func() bool) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		if fn() {
			return vm.TrueValue, nil
		}
		return vm.FalseValue, nil
	}
}

// funcARE transform a function of 'func() error' signature into CallableFunc
// type.
func funcARE(fn func() error) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return wrapError(fn()), nil
	}
}

// funcARS transform a function of 'func() string' signature into CallableFunc
// type.
func funcARS(fn func() string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		s := fn()
		if len(s) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: s}, nil
	}
}

// funcARSE transform a function of 'func() (string, error)' signature into
// CallableFunc type.
func funcARSE(fn func() (string, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		res, err := fn()
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: res}, nil
	}
}

// funcARYE transform a function of 'func() ([]byte, error)' signature into
// CallableFunc type.
func funcARYE(fn func() ([]byte, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		res, err := fn()
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > vm.MaxBytesLen {
			return nil, vm.ErrBytesLimit
		}
		return &vm.Bytes{Value: res}, nil
	}
}

// funcARF transform a function of 'func() float64' signature into CallableFunc
// type.
func funcARF(fn func() float64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		return &vm.Float{Value: fn()}, nil
	}
}

// funcARSs transform a function of 'func() []string' signature into
// CallableFunc type.
func funcARSs(fn func() []string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		arr := &vm.Array{}
		for _, elem := range fn() {
			if len(elem) > vm.MaxStringLen {
				return nil, vm.ErrStringLimit
			}
			arr.Value = append(arr.Value, &vm.String{Value: elem})
		}
		return arr, nil
	}
}

// funcARIsE transform a function of 'func() ([]int, error)' signature into
// CallableFunc type.
func funcARIsE(fn func() ([]int, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 0 {
			return nil, vm.ErrWrongNumArguments
		}
		res, err := fn()
		if err != nil {
			return wrapError(err), nil
		}
		arr := &vm.Array{}
		for _, v := range res {
			arr.Value = append(arr.Value, &vm.Int{Value: int64(v)})
		}
		return arr, nil
	}
}

// funcAIRIs transform a function of 'func(int) []int' signature into
// CallableFunc type.
func funcAIRIs(fn func(int) []int) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res := fn(i1)
		arr := &vm.Array{}
		for _, v := range res {
			arr.Value = append(arr.Value, &vm.Int{Value: int64(v)})
		}
		return arr, nil
	}
}

// funcAFRF transform a function of 'func(float64) float64' signature into
// CallableFunc type.
func funcAFRF(fn func(float64) float64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		f1, ok := vm.ToFloat64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "float(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return &vm.Float{Value: fn(f1)}, nil
	}
}

// funcAIR transform a function of 'func(int)' signature into CallableFunc type.
func funcAIR(fn func(int)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		fn(i1)
		return vm.UndefinedValue, nil
	}
}

// funcAIRF transform a function of 'func(int) float64' signature into
// CallableFunc type.
func funcAIRF(fn func(int) float64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return &vm.Float{Value: fn(i1)}, nil
	}
}

// funcAFRI transform a function of 'func(float64) int' signature into
// CallableFunc type.
func funcAFRI(fn func(float64) int) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		f1, ok := vm.ToFloat64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "float(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return &vm.Int{Value: int64(fn(f1))}, nil
	}
}

// funcAFFRF transform a function of 'func(float64, float64) float64' signature
// into CallableFunc type.
func funcAFFRF(fn func(float64, float64) float64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		f1, ok := vm.ToFloat64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "float(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		f2, ok := vm.ToFloat64(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "float(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return &vm.Float{Value: fn(f1, f2)}, nil
	}
}

// funcAIFRF transform a function of 'func(int, float64) float64' signature
// into CallableFunc type.
func funcAIFRF(fn func(int, float64) float64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		f2, ok := vm.ToFloat64(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "float(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return &vm.Float{Value: fn(i1, f2)}, nil
	}
}

// funcAFIRF transform a function of 'func(float64, int) float64' signature
// into CallableFunc type.
func funcAFIRF(fn func(float64, int) float64) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		f1, ok := vm.ToFloat64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "float(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		i2, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return &vm.Float{Value: fn(f1, i2)}, nil
	}
}

// funcAFIRB transform a function of 'func(float64, int) bool' signature
// into CallableFunc type.
func funcAFIRB(fn func(float64, int) bool) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		f1, ok := vm.ToFloat64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "float(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		i2, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		if fn(f1, i2) {
			return vm.TrueValue, nil
		}
		return vm.FalseValue, nil
	}
}

// funcAFRB transform a function of 'func(float64) bool' signature
// into CallableFunc type.
func funcAFRB(fn func(float64) bool) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		f1, ok := vm.ToFloat64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "float(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		if fn(f1) {
			return vm.TrueValue, nil
		}
		return vm.FalseValue, nil
	}
}

// funcASRS transform a function of 'func(string) string' signature into
// CallableFunc type. User function will return 'true' if underlying native
// function returns nil.
func funcASRS(fn func(string) string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s := fn(s1)
		if len(s) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: s}, nil
	}
}

// funcASRSs transform a function of 'func(string) []string' signature into
// CallableFunc type.
func funcASRSs(fn func(string) []string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res := fn(s1)
		arr := &vm.Array{}
		for _, elem := range res {
			if len(elem) > vm.MaxStringLen {
				return nil, vm.ErrStringLimit
			}
			arr.Value = append(arr.Value, &vm.String{Value: elem})
		}
		return arr, nil
	}
}

// funcASRSE transform a function of 'func(string) (string, error)' signature
// into CallableFunc type. User function will return 'true' if underlying
// native function returns nil.
func funcASRSE(fn func(string) (string, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(s1)
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: res}, nil
	}
}

// funcASRE transform a function of 'func(string) error' signature into
// CallableFunc type. User function will return 'true' if underlying native
// function returns nil.
func funcASRE(fn func(string) error) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return wrapError(fn(s1)), nil
	}
}

// funcASSRE transform a function of 'func(string, string) error' signature
// into CallableFunc type. User function will return 'true' if underlying
// native function returns nil.
func funcASSRE(fn func(string, string) error) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return wrapError(fn(s1, s2)), nil
	}
}

// funcASSRSs transform a function of 'func(string, string) []string'
// signature into CallableFunc type.
func funcASSRSs(fn func(string, string) []string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		arr := &vm.Array{}
		for _, res := range fn(s1, s2) {
			if len(res) > vm.MaxStringLen {
				return nil, vm.ErrStringLimit
			}
			arr.Value = append(arr.Value, &vm.String{Value: res})
		}
		return arr, nil
	}
}

// funcASSIRSs transform a function of 'func(string, string, int) []string'
// signature into CallableFunc type.
func funcASSIRSs(fn func(string, string, int) []string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 3 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		i3, ok := vm.ToInt(args[2])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
		}
		arr := &vm.Array{}
		for _, res := range fn(s1, s2, i3) {
			if len(res) > vm.MaxStringLen {
				return nil, vm.ErrStringLimit
			}
			arr.Value = append(arr.Value, &vm.String{Value: res})
		}
		return arr, nil
	}
}

// funcASSRI transform a function of 'func(string, string) int' signature into
// CallableFunc type.
func funcASSRI(fn func(string, string) int) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return &vm.Int{Value: int64(fn(s1, s2))}, nil
	}
}

// funcASSRS transform a function of 'func(string, string) string' signature
// into CallableFunc type.
func funcASSRS(fn func(string, string) string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		s := fn(s1, s2)
		if len(s) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: s}, nil
	}
}

// funcASSRB transform a function of 'func(string, string) bool' signature
// into CallableFunc type.
func funcASSRB(fn func(string, string) bool) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		if fn(s1, s2) {
			return vm.TrueValue, nil
		}
		return vm.FalseValue, nil
	}
}

// funcASsSRS transform a function of 'func([]string, string) string' signature
// into CallableFunc type.
func funcASsSRS(fn func([]string, string) string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		var ss1 []string
		switch arg0 := args[0].(type) {
		case *vm.Array:
			for idx, a := range arg0.Value {
				as, ok := vm.ToString(a)
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name:     fmt.Sprintf("first[%d]", idx),
						Expected: "string(compatible)",
						Found:    a.TypeName(),
					}
				}
				ss1 = append(ss1, as)
			}
		case *vm.ImmutableArray:
			for idx, a := range arg0.Value {
				as, ok := vm.ToString(a)
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name:     fmt.Sprintf("first[%d]", idx),
						Expected: "string(compatible)",
						Found:    a.TypeName(),
					}
				}
				ss1 = append(ss1, as)
			}
		default:
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "array",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := vm.ToString(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		s := fn(ss1, s2)
		if len(s) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: s}, nil
	}
}

// funcASI64RE transform a function of 'func(string, int64) error' signature
// into CallableFunc type.
func funcASI64RE(fn func(string, int64) error) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		i2, ok := vm.ToInt64(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return wrapError(fn(s1, i2)), nil
	}
}

// funcAIIRE transform a function of 'func(int, int) error' signature
// into CallableFunc type.
func funcAIIRE(fn func(int, int) error) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		i2, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return wrapError(fn(i1, i2)), nil
	}
}

// funcASIRS transform a function of 'func(string, int) string' signature
// into CallableFunc type.
func funcASIRS(fn func(string, int) string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 2 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		i2, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		s := fn(s1, i2)
		if len(s) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: s}, nil
	}
}

// funcASIIRE transform a function of 'func(string, int, int) error' signature
// into CallableFunc type.
func funcASIIRE(fn func(string, int, int) error) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 3 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		i2, ok := vm.ToInt(args[1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		i3, ok := vm.ToInt(args[2])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
		}
		return wrapError(fn(s1, i2, i3)), nil
	}
}

// funcAYRIE transform a function of 'func([]byte) (int, error)' signature
// into CallableFunc type.
func funcAYRIE(fn func([]byte) (int, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		y1, ok := vm.ToByteSlice(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "bytes(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(y1)
		if err != nil {
			return wrapError(err), nil
		}
		return &vm.Int{Value: int64(res)}, nil
	}
}

// funcAYRS transform a function of 'func([]byte) string' signature into
// CallableFunc type.
func funcAYRS(fn func([]byte) string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		y1, ok := vm.ToByteSlice(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "bytes(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res := fn(y1)
		return &vm.String{Value: res}, nil
	}
}

// funcASRIE transform a function of 'func(string) (int, error)' signature
// into CallableFunc type.
func funcASRIE(fn func(string) (int, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(s1)
		if err != nil {
			return wrapError(err), nil
		}
		return &vm.Int{Value: int64(res)}, nil
	}
}

// funcASRYE transform a function of 'func(string) ([]byte, error)' signature
// into CallableFunc type.
func funcASRYE(fn func(string) ([]byte, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		s1, ok := vm.ToString(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(s1)
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > vm.MaxBytesLen {
			return nil, vm.ErrBytesLimit
		}
		return &vm.Bytes{Value: res}, nil
	}
}

// funcAIRSsE transform a function of 'func(int) ([]string, error)' signature
// into CallableFunc type.
func funcAIRSsE(fn func(int) ([]string, error)) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(i1)
		if err != nil {
			return wrapError(err), nil
		}
		arr := &vm.Array{}
		for _, r := range res {
			if len(r) > vm.MaxStringLen {
				return nil, vm.ErrStringLimit
			}
			arr.Value = append(arr.Value, &vm.String{Value: r})
		}
		return arr, nil
	}
}

// funcAIRS transform a function of 'func(int) string' signature into
// CallableFunc type.
func funcAIRS(fn func(int) string) vm.CallableFunc {
	return func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		i1, ok := vm.ToInt(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s := fn(i1)
		if len(s) > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		return &vm.String{Value: s}, nil
	}
}
