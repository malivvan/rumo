package base64_test

import (
	"testing"

	"github.com/malivvan/rumo/vm/require"
)

var base64Bytes1 = []byte{
	0x06, 0xAC, 0x76, 0x1B, 0x1D, 0x6A, 0xFA, 0x9D, 0xB1, 0xA0,
}

const (
	base64Std    = "Bqx2Gx1q+p2xoA=="
	base64URL    = "Bqx2Gx1q-p2xoA=="
	base64RawStd = "Bqx2Gx1q+p2xoA"
	base64RawURL = "Bqx2Gx1q-p2xoA"
)

func TestBase64(t *testing.T) {
	require.Module(t, `base64`).Call("encode", base64Bytes1).Expect(base64Std)
	require.Module(t, `base64`).Call("decode", base64Std).Expect(base64Bytes1)
	require.Module(t, `base64`).Call("url_encode", base64Bytes1).Expect(base64URL)
	require.Module(t, `base64`).Call("url_decode", base64URL).Expect(base64Bytes1)
	require.Module(t, `base64`).Call("raw_encode", base64Bytes1).Expect(base64RawStd)
	require.Module(t, `base64`).Call("raw_decode", base64RawStd).Expect(base64Bytes1)
	require.Module(t, `base64`).Call("raw_url_encode", base64Bytes1).Expect(base64RawURL)
	require.Module(t, `base64`).Call("raw_url_decode", base64RawURL).Expect(base64Bytes1)
}
