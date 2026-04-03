package vv_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/malivvan/vv"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/malivvan/vv/vm"
	"github.com/malivvan/vv/vm/require"
	"github.com/malivvan/vv/vm/stdlib"
	"github.com/malivvan/vv/vm/token"
)

func TestExample(t *testing.T) {
	// script code
	src := `
each := func(seq, fn) {
    for x in seq { fn(x) }
}

sum := 0
mul := 1
each([a, b, c, d], func(x) {
	sum += x
	mul *= x
})`

	// create a new script instance
	script := vv.NewScript([]byte(src))

	// add variables with default values
	_ = script.Add("a", 0)
	_ = script.Add("b", 0)
	_ = script.Add("c", 0)
	_ = script.Add("d", 0)

	// compile script to program
	program, err := script.Compile()
	if err != nil {
		panic(err)
	}

	// clone a new instance of the program and set values
	instance := program.Clone()
	_ = instance.Set("a", 1)
	_ = instance.Set("b", 9)
	_ = instance.Set("c", 8)
	_ = instance.Set("d", 4)

	// run the instance
	err = instance.Run()
	if err != nil {
		panic(err)
	}

	// retrieve variable values
	sum := instance.Get("sum")
	mul := instance.Get("mul")
	fmt.Println(sum, mul) // "22 288"
}

func TestScript_Add(t *testing.T) {
	s := vv.NewScript([]byte(`a := b; c := test(b); d := test(5)`))
	require.NoError(t, s.Add("b", 5))     // b = 5
	require.NoError(t, s.Add("b", "foo")) // b = "foo"  (re-define before compilation)
	require.NoError(t, s.Add("test",
		func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
			if len(args) > 0 {
				switch arg := args[0].(type) {
				case *vm.Int:
					return &vm.Int{Value: arg.Value + 1}, nil
				}
			}

			return &vm.Int{Value: 0}, nil
		}))
	p, err := s.Compile()
	require.NoError(t, err)
	require.NoError(t, p.Run())
	require.Equal(t, "foo", p.Get("a").Value())
	require.Equal(t, "foo", p.Get("b").Value())
	require.Equal(t, int64(0), p.Get("c").Value())
	require.Equal(t, int64(6), p.Get("d").Value())
}

func TestScript_Remove(t *testing.T) {
	s := vv.NewScript([]byte(`a := b`))
	err := s.Add("b", 5)
	require.NoError(t, err)
	require.True(t, s.Remove("b")) // b is removed
	_, err = s.Compile()           // should not compile because b is undefined
	require.Error(t, err)
}

func TestScript_Run(t *testing.T) {
	s := vv.NewScript([]byte(`a := b`))
	err := s.Add("b", 5)
	require.NoError(t, err)
	p, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, p)
	programGet(t, p, "a", int64(5))
}

func TestScript_BuiltinModules(t *testing.T) {
	s := vv.NewScript([]byte(`math := import("math"); a := math.abs(-19.84)`))
	s.SetImports(stdlib.GetModuleMap("math"))
	p, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, p)
	programGet(t, p, "a", 19.84)

	p, err = s.Run()
	require.NoError(t, err)
	require.NotNil(t, p)
	programGet(t, p, "a", 19.84)

	s.SetImports(stdlib.GetModuleMap("os"))
	_, err = s.Run()
	require.Error(t, err)

	s.SetImports(nil)
	_, err = s.Run()
	require.Error(t, err)
}

func TestScript_SourceModules(t *testing.T) {
	s := vv.NewScript([]byte(`
enum := import("enum")
a := enum.all([1,2,3], func(_, v) { 
	return v > 0 
})
`))
	s.SetImports(stdlib.GetModuleMap("enum"))
	c, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, c)
	programGet(t, c, "a", true)

	s.SetImports(nil)
	_, err = s.Run()
	require.Error(t, err)
}

func TestScript_SetMaxConstObjects(t *testing.T) {
	// one constant '5'
	s := vv.NewScript([]byte(`a := 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err := s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 1", err.Error())

	// two constants '5' and '1'
	s = vv.NewScript([]byte(`a := 5 + 1`))
	s.SetMaxConstObjects(2) // limit = 2
	_, err = s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 2", err.Error())

	// duplicates will be removed
	s = vv.NewScript([]byte(`a := 5 + 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 1", err.Error())

	// no limit set
	s = vv.NewScript([]byte(`a := 1 + 2 + 3 + 4 + 5`))
	_, err = s.Compile()
	require.NoError(t, err)
}

func TestScriptConcurrency(t *testing.T) {
	solve := func(a, b, c int) (d, e int) {
		a += 2
		b += c
		a += b * 2
		d = a + b + c
		e = 0
		for i := 1; i <= d; i++ {
			e += i
		}
		e *= 2
		return
	}

	code := []byte(`
mod1 := import("mod1")

a += 2
b += c
a += b * 2

arr := [a, b, c]
arrstr := string(arr)
map := {a: a, b: b, c: c}

d := a + b + c
s := 0

for i:=1; i<=d; i++ {
	s += i
}

e := mod1.double(s)
`)
	mod1 := map[string]vm.Object{
		"double": &vm.BuiltinFunction{
			Value: func(ctx context.Context, args ...vm.Object) (
				ret vm.Object,
				err error,
			) {
				arg0, _ := vm.ToInt64(args[0])
				ret = &vm.Int{Value: arg0 * 2}
				return
			},
		},
	}

	scr := vv.NewScript(code)
	_ = scr.Add("a", 0)
	_ = scr.Add("b", 0)
	_ = scr.Add("c", 0)
	mods := vm.NewModuleMap()
	mods.AddBuiltinModule("mod1", mod1)
	scr.SetImports(mods)
	compiled, err := scr.Compile()
	require.NoError(t, err)

	executeFn := func(compiled *vv.Program, a, b, c int) (d, e int) {
		_ = compiled.Set("a", a)
		_ = compiled.Set("b", b)
		_ = compiled.Set("c", c)
		err := compiled.Run()
		require.NoError(t, err)
		d = compiled.Get("d").Int()
		e = compiled.Get("e").Int()
		return
	}

	concurrency := 500
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(compiled *vv.Program) {
			time.Sleep(time.Duration(rand.Int63n(50)) * time.Millisecond)
			defer wg.Done()

			a := rand.Intn(10)
			b := rand.Intn(10)
			c := rand.Intn(10)

			d, e := executeFn(compiled, a, b, c)
			expectedD, expectedE := solve(a, b, c)

			require.Equal(t, expectedD, d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, expectedE, e, "input: %d, %d, %d", a, b, c)
		}(compiled.Clone())
	}
	wg.Wait()
}

type Counter struct {
	vm.ObjectImpl
	value int64
}

func (o *Counter) TypeName() string {
	return "counter"
}

func (o *Counter) String() string {
	return fmt.Sprintf("Counter(%d)", o.value)
}

func (o *Counter) BinaryOp(
	op token.Token,
	rhs vm.Object,
) (vm.Object, error) {
	switch rhs := rhs.(type) {
	case *Counter:
		switch op {
		case token.Add:
			return &Counter{value: o.value + rhs.value}, nil
		case token.Sub:
			return &Counter{value: o.value - rhs.value}, nil
		}
	case *vm.Int:
		switch op {
		case token.Add:
			return &Counter{value: o.value + rhs.Value}, nil
		case token.Sub:
			return &Counter{value: o.value - rhs.Value}, nil
		}
	}

	return nil, errors.New("invalid operator")
}

func (o *Counter) IsFalsy() bool {
	return o.value == 0
}

func (o *Counter) Equals(t vm.Object) bool {
	if tc, ok := t.(*Counter); ok {
		return o.value == tc.value
	}

	return false
}

func (o *Counter) Copy() vm.Object {
	return &Counter{value: o.value}
}

func (o *Counter) Call(_ context.Context, _ ...vm.Object) (vm.Object, error) {
	return &vm.Int{Value: o.value}, nil
}

func (o *Counter) CanCall() bool {
	return true
}

func TestScript_CustomObjects(t *testing.T) {
	p := compile(t, `a := c1(); s := string(c1); c2 := c1; c2++`, M{
		"c1": &Counter{value: 5},
	})
	programRun(t, p)
	programGet(t, p, "a", int64(5))
	programGet(t, p, "s", "Counter(5)")
	compiledGetCounter(t, p, "c2", &Counter{value: 6})

	p = compile(t, `
arr := [1, 2, 3, 4]
for x in arr {
	c1 += x
}
out := c1()
`, M{
		"c1": &Counter{value: 5},
	})
	programRun(t, p)
	programGet(t, p, "out", int64(15))
}

func compiledGetCounter(t *testing.T, p *vv.Program, name string, expected *Counter) {
	v := p.Get(name)
	require.NotNil(t, v)

	actual := v.Value().(*Counter)
	require.NotNil(t, actual)
	require.Equal(t, expected.value, actual.value)
}

func TestScriptSourceModule(t *testing.T) {
	// script1 imports "mod1"
	scr := vv.NewScript([]byte(`out := import("mod")`))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`export 5`))
	scr.SetImports(mods)
	p, err := scr.Run()
	require.NoError(t, err)
	require.Equal(t, int64(5), p.Get("out").Value())

	// executing module function
	scr = vv.NewScript([]byte(`fn := import("mod"); out := fn()`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod",
		[]byte(`a := 3; export func() { return a + 5 }`))
	scr.SetImports(mods)
	p, err = scr.Run()
	require.NoError(t, err)
	require.Equal(t, int64(8), p.Get("out").Value())

	scr = vv.NewScript([]byte(`out := import("mod")`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod",
		[]byte(`text := import("text"); export text.title("foo")`))
	mods.AddBuiltinModule("text",
		map[string]vm.Object{
			"title": &vm.BuiltinFunction{
				Name: "title",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					s, _ := vm.ToString(args[0])
					return &vm.String{Value: strings.Title(s)}, nil
				}},
		})
	scr.SetImports(mods)
	p, err = scr.Run()
	require.NoError(t, err)
	require.Equal(t, "Foo", p.Get("out").Value())
	scr.SetImports(nil)
	_, err = scr.Run()
	require.Error(t, err)
}

func BenchmarkArrayIndex(b *testing.B) {
	bench(b.N, `a := [1, 2, 3, 4, 5, 6, 7, 8, 9];
        for i := 0; i < 1000; i++ {
            a[0]; a[1]; a[2]; a[3]; a[4]; a[5]; a[6]; a[7]; a[7];
        }
    `)
}

func BenchmarkArrayIndexCompare(b *testing.B) {
	bench(b.N, `a := [1, 2, 3, 4, 5, 6, 7, 8, 9];
        for i := 0; i < 1000; i++ {
            1; 2; 3; 4; 5; 6; 7; 8; 9;
        }
    `)
}

func bench(n int, input string) {
	s := vv.NewScript([]byte(input))
	c, err := s.Compile()
	if err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		if err := c.Run(); err != nil {
			panic(err)
		}
	}
}

type M map[string]interface{}

func TestProgram_Get(t *testing.T) {
	// simple script
	c := compile(t, `a := 5`, nil)
	programRun(t, c)
	programGet(t, c, "a", int64(5))

	// user-defined variables
	compileError(t, `a := b`, nil)          // compile error because "b" is not defined
	c = compile(t, `a := b`, M{"b": "foo"}) // now compile with b = "foo" defined
	programGet(t, c, "a", nil)              // a = undefined; because it's before Compiled.Run()
	programRun(t, c)                        // Compiled.Run()
	programGet(t, c, "a", "foo")            // a = "foo"
}

func TestProgram_GetAll(t *testing.T) {
	c := compile(t, `a := 5`, nil)
	programRun(t, c)
	programGetAll(t, c, M{"a": int64(5)})

	c = compile(t, `a := b`, M{"b": "foo"})
	programRun(t, c)
	programGetAll(t, c, M{"a": "foo", "b": "foo"})

	c = compile(t, `a := b; b = 5`, M{"b": "foo"})
	programRun(t, c)
	programGetAll(t, c, M{"a": "foo", "b": int64(5)})
}

func TestProgram_IsDefined(t *testing.T) {
	c := compile(t, `a := 5`, nil)
	programIsDefined(t, c, "a", false) // a is not defined before Run()
	programRun(t, c)
	programIsDefined(t, c, "a", true)
	programIsDefined(t, c, "b", false)
}

func TestProgram_Set(t *testing.T) {
	p := compile(t, `a := b`, M{"b": "foo"})
	programRun(t, p)
	programGet(t, p, "a", "foo")

	// replace value of 'b'
	err := p.Set("b", "bar")
	require.NoError(t, err)
	programRun(t, p)
	programGet(t, p, "a", "bar")

	// try to replace undefined variable
	err = p.Set("c", 1984)
	require.Error(t, err) // 'c' is not defined

	// case #2
	p = compile(t, `
a := func() { 
	return func() {
		return b + 5
	}() 
}()`, M{"b": 5})
	programRun(t, p)
	programGet(t, p, "a", int64(10))
	err = p.Set("b", 10)
	require.NoError(t, err)
	programRun(t, p)
	programGet(t, p, "a", int64(15))
}

func TestProgram_RunContext(t *testing.T) {
	// machine completes normally
	p := compile(t, `a := 5`, nil)
	err := p.RunContext(context.Background())
	require.NoError(t, err)
	programGet(t, p, "a", int64(5))

	// timeout
	p = compile(t, `for true {}`, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	err = p.RunContext(ctx)
	require.Equal(t, context.DeadlineExceeded, err)
}

func TestProgram_EncodeDecode(t *testing.T) {
	p := compile(t, `for true {}`, nil)
	p.Bytecode().MainFunction.SourceMap = nil

	b, err := p.Marshal()
	require.NoError(t, err)
	cx := new(vv.Program)

	err = cx.Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, p, cx)

	bx, err := cx.Marshal()
	require.NoError(t, err)

	require.Equal(t, b, bx, "encoded bytes should be equal")
}

func compile(t *testing.T, input string, vars M) *vv.Program {
	s := vv.NewScript([]byte(input))
	for vn, vv := range vars {
		err := s.Add(vn, vv)
		require.NoError(t, err)
	}

	c, err := s.Compile()
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

func compileError(t *testing.T, input string, vars M) {
	s := vv.NewScript([]byte(input))
	for vn, vv := range vars {
		err := s.Add(vn, vv)
		require.NoError(t, err)
	}
	_, err := s.Compile()
	require.Error(t, err)
}

func programRun(t *testing.T, p *vv.Program) {
	err := p.Run()
	require.NoError(t, err)
}

func programGet(t *testing.T, p *vv.Program, name string, expected interface{}) {
	v := p.Get(name)
	require.NotNil(t, v)
	require.Equal(t, expected, v.Value())
}

func programGetAll(t *testing.T, p *vv.Program, expected M) {
	vars := p.GetAll()
	require.Equal(t, len(expected), len(vars))

	for k, v := range expected {
		var found bool
		for _, e := range vars {
			if e.Name() == k {
				require.Equal(t, v, e.Value())
				found = true
			}
		}
		require.True(t, found, "variable '%s' not found", k)
	}
}

func programIsDefined(t *testing.T, p *vv.Program, name string, expected bool) {
	require.Equal(t, expected, p.IsDefined(name))
}
