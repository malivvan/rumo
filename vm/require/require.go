package require

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/malivvan/rumo"
	stdtime "github.com/malivvan/rumo/std/time"
	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/parser"
	"github.com/malivvan/rumo/vm/token"
)

// ARR -
type ARR = []interface{}

// MAP -
type MAP = map[string]interface{}

// IARR -
type IARR []interface{}

// IMAP -
type IMAP map[string]interface{}

// Module loads a module and returns it as a CallRes for chaining calls.
func Module(t *testing.T, moduleName string) CallRes {
	mod := rumo.GetModuleMap(moduleName).GetBuiltinModule(moduleName)
	if mod == nil {
		return CallRes{T: t, E: fmt.Errorf("module not found: %s", moduleName)}
	}

	return CallRes{T: t, O: mod}
}

// Object converts a Go value to a vm.Object for comparison in tests.
func Object(v interface{}) vm.Object {
	switch v := v.(type) {
	case vm.Object:
		return v
	case string:
		return &vm.String{Value: v}
	case int64:
		return &vm.Int{Value: v}
	case int: // for convenience
		return &vm.Int{Value: int64(v)}
	case bool:
		if v {
			return vm.TrueValue
		}
		return vm.FalseValue
	case rune:
		return &vm.Char{Value: v}
	case byte: // for convenience
		return &vm.Char{Value: rune(v)}
	case float32:
		return &vm.Float32{Value: v}
	case float64:
		return &vm.Float64{Value: v}
	case []byte:
		return &vm.Bytes{Value: v}
	case MAP:
		objs := make(map[string]vm.Object)
		for k, v := range v {
			objs[k] = Object(v)
		}

		return &vm.Map{Value: objs}
	case ARR:
		var objs []vm.Object
		for _, e := range v {
			objs = append(objs, Object(e))
		}

		return &vm.Array{Value: objs}
	case IMAP:
		objs := make(map[string]vm.Object)
		for k, v := range v {
			objs[k] = Object(v)
		}

		return &vm.ImmutableMap{Value: objs}
	case IARR:
		var objs []vm.Object
		for _, e := range v {
			objs = append(objs, Object(e))
		}

		return &vm.ImmutableArray{Value: objs}
	case time.Time:
		return stdtime.TimeObject(v)
	case []int:
		var objs []vm.Object
		for _, e := range v {
			objs = append(objs, &vm.Int{Value: int64(e)})
		}

		return &vm.Array{Value: objs}
	}

	panic(fmt.Errorf("unknown type: %T", v))
}

// Expect runs the input script and asserts that the value of "out" is equal to expected.
func Expect(t *testing.T, input string, expected interface{}) {
	s := rumo.NewScript(rumo.MapFS(map[string][]byte{"main.rumo": []byte(input)}), "main.rumo")
	s.SetImports(rumo.GetModuleMap(rumo.AllModuleNames()...))
	c, err := s.Run()
	NoError(t, err)
	NotNil(t, c)
	v := c.Get("out")
	NotNil(t, v)
	Equal(t, expected, v.Value())
}

// CallRes is a helper struct for chaining calls to module functions and asserting results.
type CallRes struct {
	T *testing.T
	O interface{}
	E error
}

// Call calls a function on the result of a previous call and returns a new CallRes for chaining.
func (c CallRes) Call(funcName string, args ...interface{}) CallRes {
	if c.E != nil {
		return c
	}

	var oargs []vm.Object
	for _, v := range args {
		oargs = append(oargs, Object(v))
	}

	switch o := c.O.(type) {
	case *vm.BuiltinModule:
		m, ok := o.Attrs[funcName]
		if !ok {
			return CallRes{T: c.T, E: fmt.Errorf(
				"function not found: %s", funcName)}
		}

		f, ok := m.(*vm.BuiltinFunction)
		if !ok {
			return CallRes{T: c.T, E: fmt.Errorf(
				"non-callable: %s", funcName)}
		}

		res, err := f.Value(context.Background(), oargs...)
		return CallRes{T: c.T, O: res, E: err}
	case *vm.BuiltinFunction:
		res, err := o.Value(context.Background(), oargs...)
		return CallRes{T: c.T, O: res, E: err}
	case *vm.ImmutableMap:
		m, ok := o.Value[funcName]
		if !ok {
			return CallRes{T: c.T, E: fmt.Errorf("function not found: %s", funcName)}
		}

		f, ok := m.(*vm.BuiltinFunction)
		if !ok {
			return CallRes{T: c.T, E: fmt.Errorf("non-callable: %s", funcName)}
		}

		res, err := f.Value(context.Background(), oargs...)
		return CallRes{T: c.T, O: res, E: err}
	default:
		panic(fmt.Errorf("unexpected object: %v (%T)", o, o))
	}
}

// Expect asserts that the result of the previous call is equal to expected and that no error occurred.
func (c CallRes) Expect(expected interface{}, msgAndArgs ...interface{}) {
	NoError(c.T, c.E, msgAndArgs...)
	Equal(c.T, Object(expected), c.O, msgAndArgs...)
}

// ExpectError asserts that the previous call resulted in an error.
func (c CallRes) ExpectError() {
	Error(c.T, c.E)
}

// ExpectNoError asserts that the previous call did not result in an error.
func (c CallRes) ExpectNoError() {
	NoError(c.T, c.E)
}

// NoError asserts err is not an error.
func NoError(t *testing.T, err error, msg ...interface{}) {
	if err != nil {
		failExpectedActual(t, "no error", err, msg...)
	}
}

// Error asserts err is an error.
func Error(t *testing.T, err error, msg ...interface{}) {
	if err == nil {
		failExpectedActual(t, "error", err, msg...)
	}
}

// Nil asserts v is nil.
func Nil(t *testing.T, v interface{}, msg ...interface{}) {
	if !isNil(v) {
		failExpectedActual(t, "nil", v, msg...)
	}
}

// True asserts v is true.
func True(t *testing.T, v bool, msg ...interface{}) {
	if !v {
		failExpectedActual(t, "true", v, msg...)
	}
}

// False asserts vis false.
func False(t *testing.T, v bool, msg ...interface{}) {
	if v {
		failExpectedActual(t, "false", v, msg...)
	}
}

// NotNil asserts v is not nil.
func NotNil(t *testing.T, v interface{}, msg ...interface{}) {
	if isNil(v) {
		failExpectedActual(t, "not nil", v, msg...)
	}
}

// IsType asserts expected and actual are of the same type.
func IsType(
	t *testing.T,
	expected, actual interface{},
	msg ...interface{},
) {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		failExpectedActual(t, reflect.TypeOf(expected),
			reflect.TypeOf(actual), msg...)
	}
}

// Equal asserts expected and actual are equal.
func Equal(t *testing.T, expected, actual interface{}, msg ...interface{}) {
	if isNil(expected) {
		Nil(t, actual, "expected nil, but got not nil")
		return
	}
	NotNil(t, actual, "expected not nil, but got nil")
	IsType(t, expected, actual, msg...)

	switch expected := expected.(type) {
	case int:
		if expected != actual.(int) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case int64:
		if expected != actual.(int64) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case float64:
		if expected != actual.(float64) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case string:
		if expected != actual.(string) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case []byte:
		if !bytes.Equal(expected, actual.([]byte)) {
			failExpectedActual(t, string(expected),
				string(actual.([]byte)), msg...)
		}
	case []string:
		if !equalStringSlice(expected, actual.([]string)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case []int:
		if !equalIntSlice(expected, actual.([]int)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case bool:
		if expected != actual.(bool) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case rune:
		if expected != actual.(rune) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case *vm.Symbol:
		if !equalSymbol(expected, actual.(*vm.Symbol)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case parser.Pos:
		if expected != actual.(parser.Pos) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case token.Token:
		if expected != actual.(token.Token) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case []vm.Object:
		equalObjectSlice(t, expected, actual.([]vm.Object), msg...)
	case *vm.Int:
		Equal(t, expected.Value, actual.(*vm.Int).Value, msg...)
	case *vm.Float32:
		Equal(t, expected.Value, actual.(*vm.Float32).Value, msg...)
	case *vm.Float64:
		Equal(t, expected.Value, actual.(*vm.Float64).Value, msg...)
	case *vm.String:
		Equal(t, expected.Value, actual.(*vm.String).Value, msg...)
	case *vm.Char:
		Equal(t, expected.Value, actual.(*vm.Char).Value, msg...)
	case *vm.Bool:
		if expected != actual {
			failExpectedActual(t, expected, actual, msg...)
		}
	case *vm.Array:
		equalObjectSlice(t, expected.Value,
			actual.(*vm.Array).Value, msg...)
	case *vm.ImmutableArray:
		equalObjectSlice(t, expected.Value,
			actual.(*vm.ImmutableArray).Value, msg...)
	case *vm.Bytes:
		if !bytes.Equal(expected.Value, actual.(*vm.Bytes).Value) {
			failExpectedActual(t, string(expected.Value),
				string(actual.(*vm.Bytes).Value), msg...)
		}
	case *vm.Map:
		equalObjectMap(t, expected.Value,
			actual.(*vm.Map).Value, msg...)
	case *vm.ImmutableMap:
		equalObjectMap(t, expected.Value,
			actual.(*vm.ImmutableMap).Value, msg...)
	case *vm.CompiledFunction:
		equalCompiledFunction(t, expected,
			actual.(*vm.CompiledFunction), msg...)
	case *vm.Undefined:
		if expected != actual {
			failExpectedActual(t, expected, actual, msg...)
		}
	case *vm.Error:
		Equal(t, expected.Value, actual.(*vm.Error).Value, msg...)
	case vm.Object:
		if !expected.Equals(actual.(vm.Object)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case *parser.SourceFileSet:
		if !expected.Equals(actual.(*parser.SourceFileSet)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case *parser.SourceFile:
		if !expected.Equals(actual.(*parser.SourceFile)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case error:
		if expected != actual.(error) {
			failExpectedActual(t, expected, actual, msg...)
		}
	case *rumo.Program:
		if !expected.Equals(actual.(*rumo.Program)) {
			failExpectedActual(t, expected, actual, msg...)
		}
	default:
		panic(fmt.Errorf("type not implemented: %T", expected))
	}
}

// Fail marks the function as having failed but continues execution.
func Fail(t *testing.T, msg ...interface{}) {
	t.Logf("\nError trace:\n\t%s\n%s", strings.Join(errorTrace(), "\n\t"),
		message(msg...))
	t.Fail()
}

func failExpectedActual(
	t *testing.T,
	expected, actual interface{},
	msg ...interface{},
) {
	var addMsg string
	if len(msg) > 0 {
		addMsg = "\nMessage:  " + message(msg...)
	}

	t.Logf("\nError trace:\n\t%s\nExpected: %v\nActual:   %v%s",
		strings.Join(errorTrace(), "\n\t"),
		expected, actual,
		addMsg)
	t.FailNow()
}

func message(formatArgs ...interface{}) string {
	var format string
	var args []interface{}
	if len(formatArgs) > 0 {
		format = formatArgs[0].(string)
	}
	if len(formatArgs) > 1 {
		args = formatArgs[1:]
	}
	return fmt.Sprintf(format, args...)
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalSymbol(a, b *vm.Symbol) bool {
	return a.Name == b.Name &&
		a.Index == b.Index &&
		a.Scope == b.Scope
}

func equalObjectSlice(
	t *testing.T,
	expected, actual []vm.Object,
	msg ...interface{},
) {
	Equal(t, len(expected), len(actual), msg...)
	for i := 0; i < len(expected); i++ {
		Equal(t, expected[i], actual[i], msg...)
	}
}

func equalObjectMap(
	t *testing.T,
	expected, actual map[string]vm.Object,
	msg ...interface{},
) {
	Equal(t, len(expected), len(actual), msg...)
	for key, expectedVal := range expected {
		actualVal := actual[key]
		Equal(t, expectedVal, actualVal, msg...)
	}
}

func equalCompiledFunction(
	t *testing.T,
	expected, actual vm.Object,
	msg ...interface{},
) {
	expectedT := expected.(*vm.CompiledFunction)
	actualT := actual.(*vm.CompiledFunction)
	Equal(t,
		vm.FormatInstructions(expectedT.Instructions, 0),
		vm.FormatInstructions(actualT.Instructions, 0), msg...)
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	value := reflect.ValueOf(v)
	kind := value.Kind()
	return kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil()
}

func errorTrace() []string {
	var pc uintptr
	file := ""
	line := 0
	var ok bool
	name := ""

	var callers []string
	for i := 0; ; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			break
		}

		if file == "<autogenerated>" {
			break
		}

		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		name = f.Name()

		if name == "testing.tRunner" {
			break
		}

		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
		if len(parts) > 1 {
			dir := parts[len(parts)-2]
			if dir != "require" ||
				file == "mock_test.go" {
				callers = append(callers, fmt.Sprintf("%s:%d", file, line))
			}
		}

		// Drop the package
		segments := strings.Split(name, ".")
		name = segments[len(segments)-1]
		if isTest(name, "Test") ||
			isTest(name, "Benchmark") ||
			isTest(name, "Example") {
			break
		}
	}
	return callers
}

func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}
