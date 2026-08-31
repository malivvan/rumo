package json

import (
	"testing"

	"github.com/malivvan/rumo/vm"
)

// TestEncodeCyclicData returns an error instead of overflowing the stack.
func TestEncodeCyclicData(t *testing.T) {
	m := &vm.Map{Value: map[string]vm.Object{}}
	m.Value["self"] = m
	if _, err := Encode(m); err == nil {
		t.Fatal("expected error for cyclic map, got nil")
	}

	a := &vm.Array{}
	a.Value = []vm.Object{a}
	if _, err := Encode(a); err == nil {
		t.Fatal("expected error for cyclic array, got nil")
	}
}

// TestEncodeSortedKeys checks that map keys are emitted in sorted order so
// output is deterministic.
func TestEncodeSortedKeys(t *testing.T) {
	m := &vm.Map{Value: map[string]vm.Object{
		"b": &vm.Int{Value: 1},
		"a": &vm.Int{Value: 2},
		"c": &vm.Int{Value: 3},
	}}
	b, err := Encode(m)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `{"a":2,"b":1,"c":3}`
	if string(b) != want {
		t.Fatalf("got %s, want %s", b, want)
	}
}

// TestEncodeUnsupportedType reports an error instead of emitting invalid
// JSON (e.g. `[1,,3]`).
func TestEncodeUnsupportedType(t *testing.T) {
	if _, err := Encode(&vm.Array{Value: []vm.Object{
		&vm.Int{Value: 1},
		&vm.Ptr{},
	}}); err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

// TestDecodeMaxDepth returns a clean SyntaxError for pathological nesting.
func TestDecodeMaxDepth(t *testing.T) {
	depth := 20000
	buf := make([]byte, depth)
	for i := range buf {
		buf[i] = '['
	}
	if _, err := Decode(buf); err == nil {
		t.Fatal("expected max-depth error, got nil")
	}
}

// TestDecodeIntegerPrecision keeps integers exact beyond 2^53.
func TestDecodeIntegerPrecision(t *testing.T) {
	o, err := Decode([]byte("9007199254740993"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	n, ok := o.(*vm.Int)
	if !ok {
		t.Fatalf("expected *vm.Int, got %T", o)
	}
	if n.Value != 9007199254740993 {
		t.Fatalf("precision lost: got %d", n.Value)
	}

	// Round-trip through encode.
	b, err := Encode(o)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(b) != "9007199254740993" {
		t.Fatalf("round-trip mismatch: %s", b)
	}
}
