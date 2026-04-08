package json_test

import (
	gojson "encoding/json"
	"testing"

	"github.com/malivvan/rumo/std/json"
	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

type ARR = []interface{}
type MAP = map[string]interface{}

func TestJSON(t *testing.T) {
	testJSONEncodeDecode(t, nil)

	testJSONEncodeDecode(t, 0)
	testJSONEncodeDecode(t, 1)
	testJSONEncodeDecode(t, -1)
	testJSONEncodeDecode(t, 1984)
	testJSONEncodeDecode(t, -1984)

	testJSONEncodeDecode(t, 0.0)
	testJSONEncodeDecode(t, 1.0)
	testJSONEncodeDecode(t, -1.0)
	testJSONEncodeDecode(t, 19.84)
	testJSONEncodeDecode(t, -19.84)

	testJSONEncodeDecode(t, "")
	testJSONEncodeDecode(t, "foo")
	testJSONEncodeDecode(t, "foo bar")
	testJSONEncodeDecode(t, "foo \"bar\"")
	// See: https://github.com/malivvan/rumo/issues/268
	testJSONEncodeDecode(t, "1\u001C04")
	testJSONEncodeDecode(t, "çığöşü")
	testJSONEncodeDecode(t, "ç1\u001C04IĞÖŞÜ")
	testJSONEncodeDecode(t, "错误测试")

	testJSONEncodeDecode(t, true)
	testJSONEncodeDecode(t, false)

	testJSONEncodeDecode(t, ARR{})
	testJSONEncodeDecode(t, ARR{0})
	testJSONEncodeDecode(t, ARR{false})
	testJSONEncodeDecode(t, ARR{1, 2, 3,
		"four", false})
	testJSONEncodeDecode(t, ARR{1, 2, 3,
		"four", false, MAP{"a": 0, "b": "bee", "bool": true}})

	testJSONEncodeDecode(t, MAP{})
	testJSONEncodeDecode(t, MAP{"a": 0})
	testJSONEncodeDecode(t, MAP{"a": 0, "b": "bee"})
	testJSONEncodeDecode(t, MAP{"a": 0, "b": "bee", "bool": true})

	testJSONEncodeDecode(t, MAP{"a": 0, "b": "bee",
		"arr": ARR{1, 2, 3, "four"}})
	testJSONEncodeDecode(t, MAP{"a": 0, "b": "bee",
		"arr": ARR{1, 2, 3, MAP{"a": false, "b": 109.4}}})
}

func TestDecode(t *testing.T) {
	testDecodeError(t, `{`)
	testDecodeError(t, `}`)
	testDecodeError(t, `{}a`)
	testDecodeError(t, `{{}`)
	testDecodeError(t, `{}}`)
	testDecodeError(t, `[`)
	testDecodeError(t, `]`)
	testDecodeError(t, `[]a`)
	testDecodeError(t, `[[]`)
	testDecodeError(t, `[]]`)
	testDecodeError(t, `"`)
	testDecodeError(t, `"abc`)
	testDecodeError(t, `abc"`)
	testDecodeError(t, `.123`)
	testDecodeError(t, `123.`)
	testDecodeError(t, `1.2.3`)
	testDecodeError(t, `'a'`)
	testDecodeError(t, `true, false`)
	testDecodeError(t, `{"a:"b"}`)
	testDecodeError(t, `{a":"b"}`)
	testDecodeError(t, `{"a":"b":"c"}`)
}

func testDecodeError(t *testing.T, input string) {
	_, err := json.Decode([]byte(input))
	require.Error(t, err)
}

func testJSONEncodeDecode(t *testing.T, v interface{}) {
	o, err := vm.FromInterface(v)
	require.NoError(t, err)

	b, err := json.Encode(o)
	require.NoError(t, err)

	a, err := json.Decode(b)
	require.NoError(t, err, string(b))

	vj, err := gojson.Marshal(v)
	require.NoError(t, err)

	aj, err := gojson.Marshal(vm.ToInterface(a))
	require.NoError(t, err)

	require.Equal(t, vj, aj)
}

func TestModule(t *testing.T) {
	require.Module(t, "json").Call("encode", 5).Expect([]byte("5"))
	require.Module(t, "json").Call("encode", "foobar").Expect([]byte(`"foobar"`))
	require.Module(t, "json").Call("encode", MAP{"foo": 5}).Expect([]byte("{\"foo\":5}"))
	require.Module(t, "json").Call("encode", require.IMAP{"foo": 5}).Expect([]byte("{\"foo\":5}"))
	require.Module(t, "json").Call("encode", ARR{1, 2, 3}).Expect([]byte("[1,2,3]"))
	require.Module(t, "json").Call("encode", require.IARR{1, 2, 3}).Expect([]byte("[1,2,3]"))
	require.Module(t, "json").Call("encode", MAP{"foo": "bar"}).Expect([]byte("{\"foo\":\"bar\"}"))
	require.Module(t, "json").Call("encode", MAP{"foo": 1.8}).Expect([]byte("{\"foo\":1.8}"))
	require.Module(t, "json").Call("encode", MAP{"foo": true}).Expect([]byte("{\"foo\":true}"))
	require.Module(t, "json").Call("encode", MAP{"foo": '8'}).Expect([]byte("{\"foo\":56}"))
	require.Module(t, "json").Call("encode", MAP{"foo": []byte("foo")}).Expect([]byte("{\"foo\":\"Zm9v\"}")) // json encoding returns []byte as base64 encoded string
	require.Module(t, "json").Call("encode", MAP{"foo": ARR{"bar", 1, 1.8, '8', true}}).Expect([]byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	require.Module(t, "json").Call("encode", MAP{"foo": require.IARR{"bar", 1, 1.8, '8', true}}).Expect([]byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	require.Module(t, "json").Call("encode", MAP{"foo": ARR{ARR{"bar", 1}, ARR{"bar", 1}}}).Expect([]byte("{\"foo\":[[\"bar\",1],[\"bar\",1]]}"))
	require.Module(t, "json").Call("encode", MAP{"foo": MAP{"string": "bar"}}).Expect([]byte("{\"foo\":{\"string\":\"bar\"}}"))
	require.Module(t, "json").Call("encode", MAP{"foo": require.IMAP{"string": "bar"}}).Expect([]byte("{\"foo\":{\"string\":\"bar\"}}"))
	require.Module(t, "json").Call("encode", MAP{"foo": MAP{"map1": MAP{"string": "bar"}}}).Expect([]byte("{\"foo\":{\"map1\":{\"string\":\"bar\"}}}"))
	require.Module(t, "json").Call("encode", ARR{ARR{"bar", 1}, ARR{"bar", 1}}).Expect([]byte("[[\"bar\",1],[\"bar\",1]]"))
	require.Module(t, "json").Call("decode", `5`).Expect(5.0)
	require.Module(t, "json").Call("decode", `"foo"`).Expect("foo")
	require.Module(t, "json").Call("decode", `[1,2,3,"bar"]`).Expect(ARR{1.0, 2.0, 3.0, "bar"})
	require.Module(t, "json").Call("decode", `{"foo":5}`).Expect(MAP{"foo": 5.0})
	require.Module(t, "json").Call("decode", `{"foo":2.5}`).Expect(MAP{"foo": 2.5})
	require.Module(t, "json").Call("decode", `{"foo":true}`).Expect(MAP{"foo": true})
	require.Module(t, "json").Call("decode", `{"foo":"bar"}`).Expect(MAP{"foo": "bar"})
	require.Module(t, "json").Call("decode", `{"foo":[1,2,3,"bar"]}`).Expect(MAP{"foo": ARR{1.0, 2.0, 3.0, "bar"}})
	require.Module(t, "json").Call("indent", []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"), "", "  ").Expect([]byte(`{
  "foo": [
    "bar",
    1,
    1.8,
    56,
    true
  ]
}`))
	require.Module(t, "json").Call("html_escape", []byte(`{"M":"<html>foo &`+"\xe2\x80\xa8 \xe2\x80\xa9"+`</html>"}`)).Expect([]byte(`{"M":"\u003chtml\u003efoo \u0026\u2028 \u2029\u003c/html\u003e"}`))
}
