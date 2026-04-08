package fmt_test

import (
	"testing"

	"github.com/malivvan/rumo/vm/require"
)

func TestFmtSprintf(t *testing.T) {
	require.Module(t, `fmt`).Call("sprintf", "").Expect("")
	require.Module(t, `fmt`).Call("sprintf", "foo").Expect("foo")
	require.Module(t, `fmt`).Call("sprintf", `foo %d %v %s`, 1, 2, "bar").Expect("foo 1 2 bar")
	require.Module(t, `fmt`).Call("sprintf", "foo %v", require.ARR{1, "bar", true}).Expect(`foo [1, "bar", true]`)
	require.Module(t, `fmt`).Call("sprintf", "foo %v %d", require.ARR{1, "bar", true}, 19).Expect(`foo [1, "bar", true] 19`)
	require.Module(t, `fmt`).Call("sprintf", "foo %v", require.MAP{"a": require.IMAP{"b": require.IMAP{"c": require.ARR{1, 2, 3}}}}).Expect(`foo {a: {b: {c: [1, 2, 3]}}}`)
	require.Module(t, `fmt`).Call("sprintf", "%v", require.IARR{1, require.IARR{2, require.IARR{3, 4}}}).Expect(`[1, [2, [3, 4]]]`)
}
