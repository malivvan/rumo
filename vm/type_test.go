package vm_test

import (
	"strings"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/parser"
)

// -------------------------------------------------------------------------
// Parser: round-trip the three supported forms of `type` statement.
// -------------------------------------------------------------------------

func TestType_ParseStruct(t *testing.T) {
	src := `
type Point struct {
    x
    y
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
	if len(st.Fields) != 2 || st.Fields[0].Name != "x" || st.Fields[1].Name != "y" {
		t.Fatalf("bad fields: %+v", st.Fields)
	}
}

func TestType_ParseFunc(t *testing.T) {
	src := `type Handler func(req, resp)`
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("type_func", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	stmt := file.Stmts[0].(*parser.TypeStmt)
	ft, ok := stmt.Type.(*parser.FuncType)
	if !ok {
		t.Fatalf("got underlying %T, want *parser.FuncType", stmt.Type)
	}
	if ft.Params.NumFields() != 2 {
		t.Fatalf("want 2 params, got %d", ft.Params.NumFields())
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
// Runtime: the three forms behave like Go-style constructors.
// -------------------------------------------------------------------------

// TestType_StructConstructAndAccess covers positional construction plus
// field get/set via selector syntax.
func TestType_StructConstructAndAccess(t *testing.T) {
	expectRun(t, `
type Point struct { x; y }
p := Point(3, 4)
out = p.x + p.y
`, Opts().Skip2ndPass(), 7)
}

// Keyword-style construction via a map literal.
func TestType_StructKeywordConstruct(t *testing.T) {
	expectRun(t, `
type Point struct { x; y }
p := Point({x: 1, y: 2})
out = p.y
`, Opts().Skip2ndPass(), 2)
}

// Zero-value construction when no arguments are passed.
func TestType_StructZeroValue(t *testing.T) {
	expectRun(t, `
type Point struct { x; y }
p := Point()
out = is_undefined(p.x)
`, Opts().Skip2ndPass(), true)
}

// Assigning to fields updates the instance.
func TestType_StructFieldSet(t *testing.T) {
	expectRun(t, `
type Point struct { x; y }
p := Point(1, 2)
p.x = 10
out = p.x
`, Opts().Skip2ndPass(), 10)
}

// Accessing or setting an unknown field is a runtime error.
func TestType_StructUnknownField(t *testing.T) {
	expectError(t, `
type Point struct { x; y }
p := Point(1, 2)
z := p.z
`, Opts().Skip2ndPass(), "no such field")
}

// Func type acts as a runtime type assertion.
func TestType_FuncAccepts(t *testing.T) {
	expectRun(t, `
type Handler func(a, b)
h := Handler(func(a, b) { return a + b })
out = h(10, 20)
`, Opts().Skip2ndPass(), 30)
}

func TestType_FuncWrongArity(t *testing.T) {
	expectError(t, `
type Handler func(a, b)
h := Handler(func(a) { return a })
`, Opts().Skip2ndPass(), "takes 1")
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
	src := `type P struct { x; x }`
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
type Point struct { x; y }
p := Point(1, 2)
out = type_name(p)
`, Opts().Skip2ndPass(), "Point")
}

// Struct instances iterate as (fieldName, value) pairs in declared order.
func TestType_StructIterate(t *testing.T) {
	expectRun(t, `
type Point struct { x; y }
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

// A UserType is itself callable, so referencing its name should report the
// type pointer rather than resolving as an undefined identifier.
func TestType_StructIsCallable(t *testing.T) {
	// Build a program, compile it, and inspect that the Point symbol
	// resolves to a *vm.UserType object.
	src := `type Point struct { x; y }`
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
	if !strings.Contains(ut.String(), "struct") {
		t.Fatalf("unexpected string form: %q", ut.String())
	}
}
