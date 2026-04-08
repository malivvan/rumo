package hex_test

import (
	"testing"

	"github.com/malivvan/rumo/vm/require"
)

var hexBytes1 = []byte{
	0x06, 0xAC, 0x76, 0x1B, 0x1D, 0x6A, 0xFA, 0x9D, 0xB1, 0xA0,
}

const hex1 = "06ac761b1d6afa9db1a0"

func TestHex(t *testing.T) {
	require.Module(t, `hex`).Call("encode", hexBytes1).Expect(hex1)
	require.Module(t, `hex`).Call("decode", hex1).Expect(hexBytes1)
}
