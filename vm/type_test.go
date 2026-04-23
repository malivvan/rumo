package vm_test

import (
	"strings"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/parser"
)

// -------------------------------------------------------------------------
// Parser: each supported form round-trips.
// -------------------------------------------------------------------------

func TestType_ParseStruct(t *testing.T) {
	src := `
type Point struct {
    x int
    y int
}
`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("type_struct", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var stmt *parser.TypeStmt
	for _, s := range file.Stmts {
		if ts, ok := s.(*parser.TypeStmt); ok {
			stmt = ts
			break
		}
	}
	if stmt == nil {
		t.Fatalf("no TypeStmt parsed")
	}
	if stmt.Name.Name != "Point" {
		t.Fatalf("got name %q, want Point", stmt.Name.Name)
	}
	st, ok := stmt.Type.(*parser.StructType)
	if !ok {
		t.Fatalf("got underlying %T, want *parser.StructType", stmt.Type)
	}
	if len(st.Fields) != 2 {
		t.Fatalf("want 2 field groups, got %d", len(st.Fields))
	}
	if st.Fields[0].Names[0].Name != "x" || st.Fields[0].Type.String() != "int" {
		t.Fatalf("unexpected first field: %+v / type=%q", st.Fields[0].Names, st.Fields[0].Type.String())
	}
	if st.Fields[1].Names[0].Name != "y" || st.Fields[1].Type.String() != "int" {
		t.Fatalf("unexpected second field: %+v / type=%q", st.Fields[1].Names, st.Fields[1].Type.String())
	}
}

func TestType_ParseStructGroupedNames(t *testing.T) {
	src := `type Point struct { x, y int }`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("grouped", -1, len(src))
	file, err := parser.NewParser(srcFile, []byte(src), nil).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	st := file.Stmts[0].(*parser.TypeStmt).Type.(*parser.StructType)
	if len(st.Fields) != 1 {
		t.Fatalf("want 1 field group, got %d", len(st.Fields))
	}
	if len(st.Fields[0].Names) != 2 {
		t.Fatalf("want 2 names in group, got %d", len(st.Fields[0].Names))
	}
	if st.Fields[0].Type.String() != "int" {
		t.Fatalf("unexpected shared type: %q", st.Fields[0].Type.String())
	}
}

func TestType_ParseFunc(t *testing.T) {
	src := `type Handler func(req string, resp string) int`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("type_func", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	stmt := file.Stmts[0].(*parser.TypeStmt)
	ft, ok := stmt.Type.(*parser.TypedFuncType)
	if !ok {
		t.Fatalf("got underlying %T, want *parser.TypedFuncType", stmt.Type)
	}
	if len(ft.Params) != 2 {
		t.Fatalf("want 2 params, got %d", len(ft.Params))
	}
	if ft.Params[0].Name.Name != "req" || ft.Params[0].Type.String() != "string" {
		t.Fatalf("bad first param: %+v", ft.Params[0])
	}
	if ft.Result == nil || ft.Result.String() != "int" {
		t.Fatalf("bad result: %v", ft.Result)
	}
}

func TestType_ParseFuncVarargs(t *testing.T) {
	src := `type Sink func(name string, rest ...int)`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("vararg", -1, len(src))
	file, err := parser.NewParser(srcFile, []byte(src), nil).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ft := file.Stmts[0].(*parser.TypeStmt).Type.(*parser.TypedFuncType)
	if !ft.VarArgs {
		t.Fatalf("expected VarArgs=true")
	}
	if len(ft.Params) != 2 {
		t.Fatalf("want 2 params, got %d", len(ft.Params))
	}
	if ft.Params[1].Type.String() != "int" {
		t.Fatalf("vararg type = %q", ft.Params[1].Type.String())
	}
}

func TestType_ParseValue(t *testing.T) {
	src := `type MyInt int`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("type_value", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	stmt := file.Stmts[0].(*parser.TypeStmt)
	if id, ok := stmt.Type.(*parser.Ident); !ok || id.Name != "int" {
		t.Fatalf("got underlying %v, want ident 'int'", stmt.Type)
	}
}

// -------------------------------------------------------------------------
// Runtime: the three forms behave like Go-style constructors with type
// validation.
// -------------------------------------------------------------------------

// Positional construction and field read.
func TestType_StructConstructAndAccess(t *testing.T) {
	expectRun(t, `
type Point struct { x, y int }
p := Point(3, 4)
out = p.x + p.y
`, Opts().Skip2ndPass(), 7)
}

// Keyword-style construction via a map literal; unspecified fields default
// to the typed zero value.
func TestType_StructKeywordConstruct(t *testing.T) {
	expectRun(t, `
type Point struct { x, y int }
p := Point({y: 2})
out = p.x + p.y
`, Opts().Skip2ndPass(), 2)
}

// Zero-value construction: fields get typed zero values, not undefined.
func TestType_StructZeroValue(t *testing.T) {
	expectRun(t, `
type Point struct { x, y int }
p := Point()
out = is_int(p.x) && p.x == 0 && p.y == 0
`, Opts().Skip2ndPass(), true)
}

// Assignment to a field succeeds when the type matches.
func TestType_StructFieldSet(t *testing.T) {
	expectRun(t, `
type Point struct { x, y int }
p := Point(1, 2)
p.x = 10
out = p.x
`, Opts().Skip2ndPass(), 10)
}

// Assignment to a field of the wrong type is rejected.
func TestType_StructFieldSetTypeMismatch(t *testing.T) {
	expectError(t, `
type Point struct { x, y int }
p := Point(1, 2)
p.x = "nope"
`, Opts().Skip2ndPass(), "type mismatch")
}

// Positional construction enforces field types.
func TestType_StructPositionalTypeMismatch(t *testing.T) {
	expectError(t, `
type Point struct { x, y int }
p := Point("nope", 2)
`, Opts().Skip2ndPass(), "type mismatch")
}

// Keyword construction enforces field types.
func TestType_StructKeywordTypeMismatch(t *testing.T) {
	expectError(t, `
type Point struct { x, y int }
p := Point({x: "nope"})
`, Opts().Skip2ndPass(), "type mismatch")
}

// Accessing an unknown field is a runtime error.
func TestType_StructUnknownField(t *testing.T) {
	expectError(t, `
type Point struct { x, y int }
p := Point(1, 2)
z := p.z
`, Opts().Skip2ndPass(), "no such field")
}

// Func type: valid call.
func TestType_FuncAccepts(t *testing.T) {
	expectRun(t, `
type Handler func(a int, b int) int
h := Handler(func(a, b) { return a + b })
out = h(10, 20)
`, Opts().Skip2ndPass(), 30)
}

// Func type: argument type mismatch is rejected at call time.
func TestType_FuncArgTypeMismatch(t *testing.T) {
	expectError(t, `
type Handler func(a int, b int) int
h := Handler(func(a, b) { return a + b })
out := h("nope", 2)
`, Opts().Skip2ndPass(), "type mismatch")
}

// Func type: return type mismatch is rejected.
func TestType_FuncReturnTypeMismatch(t *testing.T) {
	expectError(t, `
type Handler func(a int, b int) int
h := Handler(func(a, b) { return "nope" })
out := h(1, 2)
`, Opts().Skip2ndPass(), "return value type mismatch")
}

// Func type: wrong declared arity on the wrapped callable.
func TestType_FuncWrongArity(t *testing.T) {
	expectError(t, `
type Handler func(a int, b int) int
h := Handler(func(a) { return a })
`, Opts().Skip2ndPass(), "takes 1")
}

// Func type with varargs validates fixed + trailing types.
func TestType_FuncVarargs(t *testing.T) {
	expectRun(t, `
type Summer func(base int, rest ...int) int
s := Summer(func(base, ...rest) {
    total := base
    for _, v in rest {
        total += v
    }
    return total
})
out = s(1, 2, 3, 4)
`, Opts().Skip2ndPass(), 10)
}

func TestType_FuncVarargsTypeMismatch(t *testing.T) {
	expectError(t, `
type Summer func(base int, rest ...int) int
s := Summer(func(base, ...rest) { return base })
dummy := s(1, "nope")
`, Opts().Skip2ndPass(), "varargs element")
}

// Value types delegate to the underlying builtin converter.
func TestType_ValueIntFromString(t *testing.T) {
	expectRun(t, `
type MyInt int
n := MyInt("42")
out = n + 1
`, Opts().Skip2ndPass(), 43)
}

func TestType_ValueStringFromInt(t *testing.T) {
	expectRun(t, `
type MyStr string
s := MyStr(5)
out = s
`, Opts().Skip2ndPass(), "5")
}

// Unknown underlying type at compile time.
func TestType_ValueUnknownUnderlying(t *testing.T) {
	expectError(t, `type Foo whatever`, Opts().Skip2ndPass(), "unknown underlying type")
}

// Duplicate field rejection at parse time.
func TestType_StructDuplicateField(t *testing.T) {
	src := `type P struct { x, x int }`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("dup", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	_, err := p.ParseFile()
	if err == nil {
		t.Fatalf("expected parse error for duplicate field, got none")
	}
	if !strings.Contains(err.Error(), "duplicate field name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// type_name() reports the declared name for struct instances.
func TestType_StructTypeName(t *testing.T) {
	expectRun(t, `
type Point struct { x, y int }
p := Point(1, 2)
out = type_name(p)
`, Opts().Skip2ndPass(), "Point")
}

// Struct instances iterate as (fieldName, value) pairs in declared order.
func TestType_StructIterate(t *testing.T) {
	expectRun(t, `
type Point struct { x, y int }
p := Point(10, 20)
names := ""
total := 0
for k, v in p {
    names += k
    total += v
}
out = names + ":" + string(total)
`, Opts().Skip2ndPass(), "xy:30")
}

// UserType exposes enough metadata to be independently inspected after
// compilation.
func TestType_StructIsCallable(t *testing.T) {
	src := `type Point struct { x, y int }`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("callable", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := vm.NewCompiler(srcFile, nil, nil, nil, nil)
	if err := c.Compile(file); err != nil {
		t.Fatalf("compile: %v", err)
	}
	bc := c.Bytecode()
	var ut *vm.UserType
	for _, cn := range bc.Constants {
		if t, ok := cn.(*vm.UserType); ok {
			ut = t
			break
		}
	}
	if ut == nil {
		t.Fatalf("no UserType constant found in bytecode")
	}
	if !ut.CanCall() {
		t.Fatalf("UserType not callable")
	}
	if len(ut.Fields) != 2 || len(ut.FieldTypes) != 2 {
		t.Fatalf("expected 2 typed fields, got %v/%v", ut.Fields, ut.FieldTypes)
	}
	if ut.FieldTypes[0] != "int" || ut.FieldTypes[1] != "int" {
		t.Fatalf("unexpected field types: %v", ut.FieldTypes)
	}
	if !strings.Contains(ut.String(), "x int") {
		t.Fatalf("unexpected string form: %q", ut.String())
	}
}
