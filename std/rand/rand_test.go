package rand_test

import (
	"math/rand"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

func TestRand(t *testing.T) {
	// Global rand functions are non-deterministic in Go 1.20+ (auto-seeded).
	// We only verify that they complete without error.
	require.Module(t, "rand").Call("int").ExpectNoError()
	require.Module(t, "rand").Call("float").ExpectNoError()
	require.Module(t, "rand").Call("intn", 111).ExpectNoError()
	require.Module(t, "rand").Call("exp_float").ExpectNoError()
	require.Module(t, "rand").Call("norm_float").ExpectNoError()
	require.Module(t, "rand").Call("perm", 10).ExpectNoError()

	buf2 := &vm.Bytes{Value: make([]byte, 10)}
	require.Module(t, "rand").Call("read", buf2).ExpectNoError()

	// Per-instance rand is still deterministic via rand(seed).
	var seed int64 = 9191
	r := rand.New(rand.NewSource(seed))
	randObj := require.Module(t, "rand").Call("rand", seed)
	randObj.Call("int").Expect(r.Int63())
	randObj.Call("float").Expect(r.Float64())
	randObj.Call("intn", 111).Expect(r.Int63n(111))
	randObj.Call("exp_float").Expect(r.ExpFloat64())
	randObj.Call("norm_float").Expect(r.NormFloat64())
	randObj.Call("perm", 10).Expect(r.Perm(10))

	buf1 := make([]byte, 12)
	buf2 = &vm.Bytes{Value: make([]byte, 12)}
	n, _ := r.Read(buf1)
	randObj.Call("read", buf2).Expect(n)
	require.Equal(t, buf1, buf2.Value)
}
