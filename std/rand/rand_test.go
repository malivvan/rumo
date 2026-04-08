package rand_test

import (
	"math/rand"
	"os"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

func TestRand(t *testing.T) {
	if goDebug, ok := os.LookupEnv("GODEBUG"); !ok || goDebug != "randseednop=0" {
		err := os.Setenv("GODEBUG", "randseednop=0")
		require.NoError(t, err)
		defer func() {
			err = os.Unsetenv("GODEBUG")
			require.NoError(t, err)
		}()

	}

	var seed int64 = 1234
	r := rand.New(rand.NewSource(seed))

	require.Module(t, "rand").Call("seed", seed).Expect(vm.UndefinedValue)
	require.Module(t, "rand").Call("int").Expect(r.Int63())
	require.Module(t, "rand").Call("float").Expect(r.Float64())
	require.Module(t, "rand").Call("intn", 111).Expect(r.Int63n(111))
	require.Module(t, "rand").Call("exp_float").Expect(r.ExpFloat64())
	require.Module(t, "rand").Call("norm_float").Expect(r.NormFloat64())
	require.Module(t, "rand").Call("perm", 10).Expect(r.Perm(10))

	buf1 := make([]byte, 10)
	buf2 := &vm.Bytes{Value: make([]byte, 10)}
	n, _ := r.Read(buf1)
	require.Module(t, "rand").Call("read", buf2).Expect(n)
	require.Equal(t, buf1, buf2.Value)

	seed = 9191
	r = rand.New(rand.NewSource(seed))
	randObj := require.Module(t, "rand").Call("rand", seed)
	randObj.Call("seed", seed).Expect(vm.UndefinedValue)
	randObj.Call("int").Expect(r.Int63())
	randObj.Call("float").Expect(r.Float64())
	randObj.Call("intn", 111).Expect(r.Int63n(111))
	randObj.Call("exp_float").Expect(r.ExpFloat64())
	randObj.Call("norm_float").Expect(r.NormFloat64())
	randObj.Call("perm", 10).Expect(r.Perm(10))

	buf1 = make([]byte, 12)
	buf2 = &vm.Bytes{Value: make([]byte, 12)}
	n, _ = r.Read(buf1)
	randObj.Call("read", buf2).Expect(n)
	require.Equal(t, buf1, buf2.Value)
}
