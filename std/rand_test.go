package std_test

import (
	"math/rand"
	"os"
	"testing"

	"github.com/malivvan/vv/vm"
	"github.com/malivvan/vv/vm/require"
)

func TestRand(t *testing.T) {
	if os.Getenv("GODEBUG") != "randseednop=0" {
		t.Skip("GODEBUG=randseednop=0 is not set, skipping test")
	}
	var seed int64 = 1234
	r := rand.New(rand.NewSource(seed))

	module(t, "rand").call("seed", seed).expect(vm.UndefinedValue)
	module(t, "rand").call("int").expect(r.Int63())
	module(t, "rand").call("float").expect(r.Float64())
	module(t, "rand").call("intn", 111).expect(r.Int63n(111))
	module(t, "rand").call("exp_float").expect(r.ExpFloat64())
	module(t, "rand").call("norm_float").expect(r.NormFloat64())
	module(t, "rand").call("perm", 10).expect(r.Perm(10))

	buf1 := make([]byte, 10)
	buf2 := &vm.Bytes{Value: make([]byte, 10)}
	n, _ := r.Read(buf1)
	module(t, "rand").call("read", buf2).expect(n)
	require.Equal(t, buf1, buf2.Value)

	seed = 9191
	r = rand.New(rand.NewSource(seed))
	randObj := module(t, "rand").call("rand", seed)
	randObj.call("seed", seed).expect(vm.UndefinedValue)
	randObj.call("int").expect(r.Int63())
	randObj.call("float").expect(r.Float64())
	randObj.call("intn", 111).expect(r.Int63n(111))
	randObj.call("exp_float").expect(r.ExpFloat64())
	randObj.call("norm_float").expect(r.NormFloat64())
	randObj.call("perm", 10).expect(r.Perm(10))

	buf1 = make([]byte, 12)
	buf2 = &vm.Bytes{Value: make([]byte, 12)}
	n, _ = r.Read(buf1)
	randObj.call("read", buf2).expect(n)
	require.Equal(t, buf1, buf2.Value)
}
