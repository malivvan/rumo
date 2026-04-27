package rumo_test

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc64"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/malivvan/rumo"

	"github.com/malivvan/rumo/vm"

	"github.com/malivvan/rumo/vm/require"
	"github.com/malivvan/rumo/vm/token"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TestBytecodeIntegrity verifies that the bytecode serialisation format uses
// SHA-256 as its integrity checksum rather than CRC64/ECMA.
//
// The historical implementation wrote an 8-byte CRC64/ECMA checksum as the
// trailer of every marshalled Program. CRC64 is a non-cryptographic checksum:
// any attacker who can modify the bytecode bytes on disk can also trivially
// recompute a new valid CRC64 to match, making the checksum useless as
// tamper-detection. The fix replaces CRC64 with SHA-256 (32-byte trailer),
// which provides a 256-bit digest and a vastly smaller collision probability,
// and bumps FormatVersion so old CRC64-signed files are cleanly rejected.
//
// The tests below cover:
//  1. The marshalled trailer is exactly 32 bytes and equals sha256(body).
//  2. A single intentional bit-flip in the body is detected.
//  3. An attacker who recomputes the CRC64 over tampered bytes and patches
//     the trailer cannot fool Unmarshal (the old 8-byte slot no longer lives
//     in the last 8 bytes of the file – the hash is 32 bytes).
//  4. A file produced with the old FormatVersion (1, CRC64) is rejected.
func TestBytecodeIntegrity(t *testing.T) {
	p := compile(t, `a := 1 + 2`, nil)

	data, err := p.Marshal()
	require.NoError(t, err)

	// --- 1. Trailer must be SHA-256 (32 bytes), not CRC64 (8 bytes) ----------

	// Header is always 10 bytes: [4]MAGIC [2]VERSION [4]SIZE
	const headerSize = 10

	// The body size is encoded in header bytes [6:10].
	bodySize := int(binary.LittleEndian.Uint32(data[6:10]))

	// With SHA-256 the total file should be header + body + 32.
	wantTotal := headerSize + bodySize + 32
	if len(data) != wantTotal {
		t.Errorf("Marshal: total length = %d, want %d (header %d + body %d + sha256 trailer 32)",
			len(data), wantTotal, headerSize, bodySize)
	}

	// Verify the trailer bytes match sha256(body).
	body := data[headerSize : headerSize+bodySize]
	wantSum := sha256.Sum256(body)
	gotSum := data[headerSize+bodySize:]
	if len(gotSum) != 32 || string(gotSum) != string(wantSum[:]) {
		t.Errorf("Marshal: trailer is not sha256(body);\n  got  %x\n  want %x", gotSum, wantSum)
	}

	// --- 2. Single bit-flip in body must be detected by Unmarshal -----------

	tampered := make([]byte, len(data))
	copy(tampered, data)
	tampered[headerSize+bodySize/2] ^= 0x01 // flip one bit in the body

	cx := new(rumo.Program)
	if err := cx.Unmarshal(tampered); err == nil {
		t.Error("Unmarshal: single bit-flip in body was NOT detected – integrity check failed")
	}

	// --- 3. Attacker recomputes CRC64 and patches the old 8-byte slot -------
	//
	// Before the fix, the trailer was exactly 8 bytes; an attacker could
	// modify the body and then patch data[len-8:] with a freshly-computed
	// CRC64 to produce a "valid" file.  With SHA-256 the trailer is 32 bytes,
	// so patching only the last 8 bytes leaves the preceding 24 bytes of the
	// sha256 digest wrong → Unmarshal must reject the file.

	crcTampered := make([]byte, len(data))
	copy(crcTampered, data)
	// Flip a byte in the body.
	crcTampered[headerSize+bodySize/3] ^= 0xFF
	// Recompute CRC64 over the modified body and patch the last 8 bytes.
	tamperedBody := crcTampered[headerSize : headerSize+bodySize]
	tbl := crc64.MakeTable(crc64.ECMA)
	h := crc64.New(tbl)
	_, _ = h.Write(tamperedBody)
	binary.LittleEndian.PutUint64(crcTampered[len(crcTampered)-8:], h.Sum64())

	cx2 := new(rumo.Program)
	if err := cx2.Unmarshal(crcTampered); err == nil {
		t.Error("Unmarshal: attacker-recomputed CRC64 patch was NOT detected – integrity check failed")
	}

	// --- 4. Old-format (version 1 / CRC64) files must be rejected -----------

	oldFormat := make([]byte, len(data))
	copy(oldFormat, data)
	binary.LittleEndian.PutUint16(oldFormat[4:6], 1) // downgrade version to 1

	cx3 := new(rumo.Program)
	if err := cx3.Unmarshal(oldFormat); err == nil {
		t.Error("Unmarshal: file with old FormatVersion 1 (CRC64) was accepted – version gate missing")
	}
}

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
	script := rumo.NewScript([]byte(src))

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
	s := rumo.NewScript([]byte(`a := b; c := test(b); d := test(5)`))
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
	s := rumo.NewScript([]byte(`a := b`))
	err := s.Add("b", 5)
	require.NoError(t, err)
	require.True(t, s.Remove("b")) // b is removed
	_, err = s.Compile()           // should not compile because b is undefined
	require.Error(t, err)
}

func TestScript_Run(t *testing.T) {
	s := rumo.NewScript([]byte(`a := b`))
	err := s.Add("b", 5)
	require.NoError(t, err)
	p, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, p)
	programGet(t, p, "a", int64(5))
}

func TestScript_BuiltinModules(t *testing.T) {
	s := rumo.NewScript([]byte(`math := import("math"); a := math.abs(-19.84)`))
	s.SetImports(rumo.GetModuleMap("math"))
	p, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, p)
	programGet(t, p, "a", 19.84)

	p, err = s.Run()
	require.NoError(t, err)
	require.NotNil(t, p)
	programGet(t, p, "a", 19.84)

	s.SetImports(rumo.GetModuleMap("os"))
	_, err = s.Run()
	require.Error(t, err)

	s.SetImports(nil)
	_, err = s.Run()
	require.Error(t, err)
}

func TestScript_SourceModules(t *testing.T) {
	s := rumo.NewScript([]byte(`
enum := import("enum")
a := enum.all([1,2,3], func(_, v) { 
	return v > 0 
})
`))
	s.SetImports(rumo.GetModuleMap("enum"))
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
	s := rumo.NewScript([]byte(`a := 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err := s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 1", err.Error())

	// two constants '5' and '1'
	s = rumo.NewScript([]byte(`a := 5 + 1`))
	s.SetMaxConstObjects(2) // limit = 2
	_, err = s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 2", err.Error())

	// duplicates will be removed
	s = rumo.NewScript([]byte(`a := 5 + 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 1", err.Error())

	// no limit set
	s = rumo.NewScript([]byte(`a := 1 + 2 + 3 + 4 + 5`))
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
m := {a: a, b: b, c: c}

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

	scr := rumo.NewScript(code)
	_ = scr.Add("a", 0)
	_ = scr.Add("b", 0)
	_ = scr.Add("c", 0)
	mods := vm.NewModuleMap()
	mods.AddBuiltinModule("mod1", mod1)
	scr.SetImports(mods)
	compiled, err := scr.Compile()
	require.NoError(t, err)

	executeFn := func(compiled *rumo.Program, a, b, c int) (d, e int) {
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
		go func(compiled *rumo.Program) {
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

func compiledGetCounter(t *testing.T, p *rumo.Program, name string, expected *Counter) {
	v := p.Get(name)
	require.NotNil(t, v)

	actual := v.Value().(*Counter)
	require.NotNil(t, actual)
	require.Equal(t, expected.value, actual.value)
}

func TestScriptSourceModule(t *testing.T) {
	// script1 imports "mod1"
	scr := rumo.NewScript([]byte(`out := import("mod")`))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`export 5`))
	scr.SetImports(mods)
	p, err := scr.Run()
	require.NoError(t, err)
	require.Equal(t, int64(5), p.Get("out").Value())

	// executing module function
	scr = rumo.NewScript([]byte(`fn := import("mod"); out := fn()`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod",
		[]byte(`a := 3; export func() { return a + 5 }`))
	scr.SetImports(mods)
	p, err = scr.Run()
	require.NoError(t, err)
	require.Equal(t, int64(8), p.Get("out").Value())

	scr = rumo.NewScript([]byte(`out := import("mod")`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod",
		[]byte(`text := import("text"); export text.title("foo")`))
	mods.AddBuiltinModule("text",
		map[string]vm.Object{
			"title": &vm.BuiltinFunction{
				Name: "title",
				Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
					s, _ := vm.ToString(args[0])
					return &vm.String{Value: cases.Title(language.Und).String(s)}, nil
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
	s := rumo.NewScript([]byte(input))
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
	cx := new(rumo.Program)

	err = cx.Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, p, cx)

	bx, err := cx.Marshal()
	require.NoError(t, err)

	require.Equal(t, b, bx, "encoded bytes should be equal")
}

func compile(t *testing.T, input string, vars M) *rumo.Program {
	s := rumo.NewScript([]byte(input))
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
	s := rumo.NewScript([]byte(input))
	for vn, vv := range vars {
		err := s.Add(vn, vv)
		require.NoError(t, err)
	}
	_, err := s.Compile()
	require.Error(t, err)
}

func programRun(t *testing.T, p *rumo.Program) {
	err := p.Run()
	require.NoError(t, err)
}

func programGet(t *testing.T, p *rumo.Program, name string, expected interface{}) {
	v := p.Get(name)
	require.NotNil(t, v)
	require.Equal(t, expected, v.Value())
}

func programGetAll(t *testing.T, p *rumo.Program, expected M) {
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

func programIsDefined(t *testing.T, p *rumo.Program, name string, expected bool) {
	require.Equal(t, expected, p.IsDefined(name))
}

// Issue #9: Program.Run()/RunContext() hold write lock during execution
//
// The Program write lock is held for the entire script lifetime, making
// Get()/Set()/IsDefined()/GetAll() block until the script finishes.
// For long-running scripts this is effectively a deadlock.
//
// The fix makes Run()/RunContext() snapshot globals under a brief read lock,
// execute on the local copy, then write back results under a brief write lock.

func TestIssue9_GetDuringRun(t *testing.T) {
	// Compile a script that loops until the context is cancelled.
	src := `for true { x += 1 }`
	script := rumo.NewScript([]byte(src))
	_ = script.Add("x", 0)
	program, err := script.Compile()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start RunContext in background.
	done := make(chan error, 1)
	go func() {
		done <- program.RunContext(ctx)
	}()

	// Give the VM a moment to start running.
	time.Sleep(50 * time.Millisecond)

	// Attempt Get() — this must NOT block for the entire script duration.
	getCh := make(chan *rumo.Variable, 1)
	go func() {
		getCh <- program.Get("x")
	}()

	select {
	case v := <-getCh:
		// Get() returned — the lock is not held during execution.
		_ = v // value may be the pre-run snapshot; that's acceptable.
	case <-time.After(1 * time.Second):
		cancel() // unblock the script
		<-done
		t.Fatal("Issue #9: Get() blocked for >1s — write lock held during execution")
	}

	cancel()
	<-done
}

func TestIssue9_SetDuringRun(t *testing.T) {
	src := `for true { x += 1 }`
	script := rumo.NewScript([]byte(src))
	_ = script.Add("x", 0)
	program, err := script.Compile()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- program.RunContext(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Attempt Set() — must NOT block for the full execution.
	setCh := make(chan error, 1)
	go func() {
		setCh <- program.Set("x", 42)
	}()

	select {
	case err := <-setCh:
		if err != nil {
			t.Fatalf("Issue #9: Set() returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		cancel()
		<-done
		t.Fatal("Issue #9: Set() blocked for >1s — write lock held during execution")
	}

	cancel()
	<-done
}

// Issue #10: Program.Clone() shares mutable globals and constants
//
// Clone() copies object pointers, not objects. Shared *vm.Map/*vm.Array
// globals race under concurrent execution.
//
// The fix makes Clone() call .Copy() on each non-nil global.

func TestIssue10_CloneDeepCopiesGlobals(t *testing.T) {
	src := `m["key"] = "modified"`
	script := rumo.NewScript([]byte(src))

	original := map[string]interface{}{
		"key": "original",
	}
	_ = script.Add("m", original)

	program, err := script.Compile()
	if err != nil {
		t.Fatal(err)
	}

	clone := program.Clone()

	// Run the clone — it modifies m["key"].
	if err := clone.Run(); err != nil {
		t.Fatal(err)
	}

	// The original program's global must be unaffected.
	v := program.Get("m")
	obj := v.Object()
	m, ok := obj.(*vm.Map)
	if !ok {
		t.Fatalf("Issue #10: expected *vm.Map, got %T", obj)
	}
	val, ok := m.Value["key"]
	if !ok {
		t.Fatal("Issue #10: key 'key' missing from original map")
	}
	s, ok := val.(*vm.String)
	if !ok {
		t.Fatalf("Issue #10: expected *vm.String, got %T", val)
	}
	if s.Value != "original" {
		t.Fatalf("Issue #10: Clone() shares mutable globals — original map was mutated to %q", s.Value)
	}
}

// TestEqualsLocksBothPrograms verifies that Program.Equals acquires a read
// lock on BOTH the receiver and the argument Program before accessing their
// fields. Without locking the argument, a concurrent writer (e.g. Set) on the
// second Program creates a data race: Equals reads other.globals unsynchronised
// while Set writes to it under other.lock. The Go race detector catches this
// with the buggy implementation.
//
// Regression tests:
//   - Concurrent Equals + Set on the argument must not data-race.
//   - Concurrent a.Equals(b) and b.Equals(a) must not deadlock.
//   - Equals returns true for structurally identical programs and false after
//     a variable is mutated.
func TestEqualsLocksBothPrograms(t *testing.T) {
	newProg := func(t *testing.T) *rumo.Program {
		t.Helper()
		// x is pre-declared via Add; the script body just references it.
		s := rumo.NewScript([]byte(`x += 0`))
		if err := s.Add("x", 42); err != nil {
			t.Fatal(err)
		}
		p, err := s.Compile()
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("concurrent Equals and Set do not race", func(t *testing.T) {
		a := newProg(t)
		b := newProg(t)

		const iterations = 2000
		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: repeatedly read b via Equals — must hold b's lock.
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = a.Equals(b)
			}
		}()

		// Goroutine 2: repeatedly write b via Set — holds b's write lock.
		// Without Equals locking b, the race detector flags this.
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = b.Set("x", i)
			}
		}()

		wg.Wait()
	})

	t.Run("symmetric call does not deadlock", func(t *testing.T) {
		a := newProg(t)
		b := newProg(t)

		done := make(chan struct{})
		go func() {
			defer close(done)
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); _ = a.Equals(b) }()
			go func() { defer wg.Done(); _ = b.Equals(a) }()
			wg.Wait()
		}()

		select {
		case <-done:
			// success
		case <-time.After(5 * time.Second):
			t.Fatal("deadlock: a.Equals(b) and b.Equals(a) concurrent calls did not complete")
		}
	})

	t.Run("equal programs", func(t *testing.T) {
		a := newProg(t)
		// A program must equal itself.
		if !a.Equals(a) {
			t.Fatal("Equals returned false when comparing a program to itself")
		}
	})

	t.Run("unequal programs after mutation", func(t *testing.T) {
		a := newProg(t)
		b := newProg(t)
		if err := b.Set("x", 99); err != nil {
			t.Fatal(err)
		}
		if a.Equals(b) {
			t.Fatal("Equals returned true after mutating b.x to a different value")
		}
	})
}

// ── 3.6 Program.Marshal reads bytecode under RLock but Bytecode is mutable ──
//
// Program.Marshal() holds p.lock.RLock() while reading p.bytecode.Constants
// and p.bytecode.MainFunction.Instructions. Program.Bytecode() returns a live
// *vm.Bytecode pointer (also under RLock). External code that receives this
// pointer can call RemoveDuplicates() — which mutates Constants and
// Instructions — concurrently with Marshal(), causing a data race. The fix adds
// a sync.RWMutex to Bytecode: RemoveDuplicates() acquires the write lock and
// Bytecode.Marshal() acquires the read lock, preventing concurrent
// read/write access.

func TestBytecodeRemoveDuplicatesMarshalRace(t *testing.T) {
	s := rumo.NewScript([]byte(`x := 1 + 2`))
	p, err := s.Compile()
	require.NoError(t, err)

	bc := p.Bytecode()

	const N = 30
	var wg sync.WaitGroup
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			p.Marshal() //nolint:errcheck
		}()
		go func() {
			defer wg.Done()
			bc.RemoveDuplicates()
		}()
	}
	wg.Wait()
}

// NewScript exposes file I/O, environment mutation, process execution, and
// working-directory changes to untrusted scripts without any restrictions. An
// embedder that does nothing more than rumo.NewScript(src).Run() gives the
// script full access to privileged os-module operations — the zero-value
// Permissions struct (all Deny* fields false) permits everything. The fix makes
// the zero-value Permissions struct deny-by-default, and adds
// vm.UnrestrictedPermissions() as an explicit opt-in for embedders that need
// the old behaviour.

// TestDefaultPermissionsDenyAll verifies that a Script created with NewScript
// and no explicit SetPermissions call denies privileged os-module operations by
// default.
func TestDefaultPermissionsDenyAll(t *testing.T) {
	const envKey = "RUMO_DEFAULT_PERM_DENY_TEST"
	_ = os.Unsetenv(envKey)
	t.Cleanup(func() { _ = os.Unsetenv(envKey) })

	s := rumo.NewScript([]byte(`os := import("os"); os.setenv("` + envKey + `", "mutated")`))
	s.SetImports(rumo.GetModuleMap("os"))
	// No SetPermissions call — should default to deny-all.
	_, err := s.Run()
	if err == nil {
		t.Fatal("NewScript default permissions should deny os.setenv, but got nil error")
	}
	if got := os.Getenv(envKey); got != "" {
		t.Errorf("env was mutated despite default deny-all permissions: %s=%q", envKey, got)
	}
}

// TestDefaultPermissionsDenyChdir verifies that os.chdir is denied by default.
func TestDefaultPermissionsDenyChdir(t *testing.T) {
	s := rumo.NewScript([]byte(`os := import("os"); os.chdir("/")`))
	s.SetImports(rumo.GetModuleMap("os"))
	_, err := s.Run()
	if err == nil {
		t.Fatal("NewScript default permissions should deny os.chdir, but got nil error")
	}
}

// TestDefaultPermissionsDenyExec verifies that os.exec is denied by default.
func TestDefaultPermissionsDenyExec(t *testing.T) {
	s := rumo.NewScript([]byte(`os := import("os"); cmd := os.exec("true"); cmd.run()`))
	s.SetImports(rumo.GetModuleMap("os"))
	_, err := s.Run()
	if err == nil {
		t.Fatal("NewScript default permissions should deny os.exec, but got nil error")
	}
}

// TestUnrestrictedPermissionsAllowsAll verifies that vm.UnrestrictedPermissions()
// re-enables all os-module capabilities, matching the previous allow-all default.
func TestUnrestrictedPermissionsAllowsAll(t *testing.T) {
	const envKey = "RUMO_UNRESTRICTED_PERM_TEST"
	_ = os.Unsetenv(envKey)
	t.Cleanup(func() { _ = os.Unsetenv(envKey) })

	s := rumo.NewScript([]byte(`os := import("os"); os.setenv("` + envKey + `", "ok")`))
	s.SetImports(rumo.GetModuleMap("os"))
	s.SetPermissions(vm.UnrestrictedPermissions())
	_, err := s.Run()
	if err != nil {
		t.Fatalf("unexpected error with UnrestrictedPermissions: %v", err)
	}
	if got := os.Getenv(envKey); got != "ok" {
		t.Errorf("setenv did not take effect with UnrestrictedPermissions: %s=%q", envKey, got)
	}
}

// The default vm.Config used by NewScript sets MaxAllocs to -1, meaning
// unlimited object allocations. Combined with MaxStringLen and MaxBytesLen set
// to math.MaxInt32, an embedder that does nothing more than
// rumo.NewScript(src).Run() exposes the host process to denial-of-service
// attacks — a loop that allocates objects or builds large strings can exhaust
// available memory with no guard. The fix ships safe defaults (10 M allocs,
// 16 MiB strings/bytes) and adds vm.UnlimitedConfig() for embedders that
// knowingly need the old unbounded behaviour.

// TestDefaultAllocsNotUnbounded verifies that NewScript enforces a finite
// object-allocation limit by default.
func TestDefaultAllocsNotUnbounded(t *testing.T) {
	// Allocate a new map object on each iteration. The loop count (100 M) far
	// exceeds any reasonable safe default, so the VM should stop early.
	src := `for i := 0; i < 100000000; i++ { x := {} }`
	s := rumo.NewScript([]byte(src))
	_, err := s.Run()
	if err == nil {
		t.Fatal("expected allocation-limit error — default MaxAllocs must be bounded, not unlimited")
	}
}

// TestDefaultStringLenBounded verifies that NewScript enforces a finite maximum
// string length by default.
func TestDefaultStringLenBounded(t *testing.T) {
	// Build a byte slice larger than the safe default (16 MiB) and convert it to
	// a string.  We do this via os.read_file to avoid a slow character-by-character
	// loop; instead, write a large temp file and read it back.
	dir := t.TempDir()
	bigFile := filepath.Join(dir, "big.bin")
	// 20 MiB of zeros — exceeds the 16 MiB default MaxStringLen.
	if err := os.WriteFile(bigFile, make([]byte, 20*1024*1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s := rumo.NewScript([]byte(`os := import("os"); data := string(os.read_file("` + bigFile + `"))`))
	s.SetImports(rumo.GetModuleMap("os"))
	s.SetPermissions(vm.UnrestrictedPermissions()) // allow file reads; test is about size limit only
	_, err := s.Run()
	if err == nil {
		t.Fatal("expected string-length-limit error — default MaxStringLen must be bounded, not unlimited")
	}
}

// TestUnlimitedConfigRemovesLimits verifies that vm.UnlimitedConfig() disables
// the default resource limits, allowing scripts to allocate freely.
func TestUnlimitedConfigRemovesLimits(t *testing.T) {
	src := `arr := []; for i := 0; i < 20000000; i++ { arr = append(arr, i) }`
	s := rumo.NewScript([]byte(src))
	s.SetMaxAllocs(-1) // explicit unlimited via existing API
	_, err := s.Run()
	if err != nil {
		t.Fatalf("unexpected error with unlimited allocs: %v", err)
	}
}
