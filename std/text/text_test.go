package text_test

import (
	"regexp"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

func TestTextRE(t *testing.T) {
	// re_match(pattern, text)
	for _, d := range []struct {
		pattern string
		text    string
	}{
		{"abc", ""},
		{"abc", "abc"},
		{"a", "abc"},
		{"b", "abc"},
		{"^a", "abc"},
		{"^b", "abc"},
	} {
		expected := regexp.MustCompile(d.pattern).MatchString(d.text)
		require.Module(t, "text").Call("re_match", d.pattern, d.text).Expect(expected, "pattern: %q, src: %q", d.pattern, d.text)
		require.Module(t, "text").Call("re_compile", d.pattern).Call("match", d.text).Expect(expected, "patter: %q, src: %q", d.pattern, d.text)
	}

	// re_find(pattern, text)
	for _, d := range []struct {
		pattern  string
		text     string
		expected interface{}
	}{
		{"a(b)", "", vm.UndefinedValue},
		{"a(b)", "ab", require.ARR{require.ARR{require.IMAP{"text": "ab", "begin": 0, "end": 2}, require.IMAP{"text": "b", "begin": 1, "end": 2}}}},
		{"a(bc)d", "abcdefgabcd", require.ARR{require.ARR{require.IMAP{"text": "abcd", "begin": 0, "end": 4}, require.IMAP{"text": "bc", "begin": 1, "end": 3}}}},
		{"(a)b(c)d", "abcdefgabcd", require.ARR{require.ARR{require.IMAP{"text": "abcd", "begin": 0, "end": 4}, require.IMAP{"text": "a", "begin": 0, "end": 1}, require.IMAP{"text": "c", "begin": 2, "end": 3}}}},
	} {
		require.Module(t, "text").Call("re_find", d.pattern, d.text).Expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		require.Module(t, "text").Call("re_compile", d.pattern).Call("find", d.text).Expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_find(pattern, text, count))
	for _, d := range []struct {
		pattern  string
		text     string
		count    int
		expected interface{}
	}{
		{"a(b)", "", -1, vm.UndefinedValue},
		{"a(b)", "ab", -1, require.ARR{require.ARR{require.IMAP{"text": "ab", "begin": 0, "end": 2}, require.IMAP{"text": "b", "begin": 1, "end": 2}}}},
		{"a(bc)d", "abcdefgabcd", -1, require.ARR{
			require.ARR{require.IMAP{"text": "abcd", "begin": 0, "end": 4}, require.IMAP{"text": "bc", "begin": 1, "end": 3}},
			require.ARR{require.IMAP{"text": "abcd", "begin": 7, "end": 11}, require.IMAP{"text": "bc", "begin": 8, "end": 10}},
		}},
		{"(a)b(c)d", "abcdefgabcd", -1, require.ARR{
			require.ARR{require.IMAP{"text": "abcd", "begin": 0, "end": 4}, require.IMAP{"text": "a", "begin": 0, "end": 1}, require.IMAP{"text": "c", "begin": 2, "end": 3}},
			require.ARR{require.IMAP{"text": "abcd", "begin": 7, "end": 11}, require.IMAP{"text": "a", "begin": 7, "end": 8}, require.IMAP{"text": "c", "begin": 9, "end": 10}},
		}},
		{"(a)b(c)d", "abcdefgabcd", 0, vm.UndefinedValue},
		{"(a)b(c)d", "abcdefgabcd", 1, require.ARR{require.ARR{require.IMAP{"text": "abcd", "begin": 0, "end": 4}, require.IMAP{"text": "a", "begin": 0, "end": 1}, require.IMAP{"text": "c", "begin": 2, "end": 3}}}},
	} {
		require.Module(t, "text").Call("re_find", d.pattern, d.text, d.count).Expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		require.Module(t, "text").Call("re_compile", d.pattern).Call("find", d.text, d.count).Expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_replace(pattern, text, repl)
	for _, d := range []struct {
		pattern string
		text    string
		repl    string
	}{
		{"a", "", "b"},
		{"a", "a", "b"},
		{"a", "acac", "b"},
		{"b", "acac", "x"},
		{"a", "acac", "123"},
		{"ac", "acac", "99"},
		{"ac$", "acac", "foo"},
		{"a(b)", "ababab", "$1"},
		{"a(b)(c)", "abcabcabc", "$2$1"},
		{"(a(b)c)", "abcabcabc", "$1$2"},
		{"(일(2)삼)", "일2삼12삼일23", "$1$2"},
		{"((일)(2)3)", "일23\n일이3\n일23", "$1$2$3"},
		{"(a(b)c)", "abc\nabc\nabc", "$1$2"},
	} {
		expected := regexp.MustCompile(d.pattern).ReplaceAllString(d.text, d.repl)
		require.Module(t, "text").Call("re_replace", d.pattern, d.text, d.repl).Expect(expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
		require.Module(t, "text").Call("re_compile", d.pattern).Call("replace", d.text, d.repl).Expect(expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
	}

	// re_split(pattern, text)
	for _, d := range []struct {
		pattern string
		text    string
	}{
		{"a", ""},
		{"a", "abcabc"},
		{"ab", "abcabc"},
		{"^a", "abcabc"},
	} {
		var expected []interface{}
		for _, ex := range regexp.MustCompile(d.pattern).Split(d.text, -1) {
			expected = append(expected, ex)
		}
		require.Module(t, "text").Call("re_split", d.pattern, d.text).Expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
		require.Module(t, "text").Call("re_compile", d.pattern).Call("split", d.text).Expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_split(pattern, text, count))
	for _, d := range []struct {
		pattern string
		text    string
		count   int
	}{
		{"a", "", -1},
		{"a", "abcabc", -1},
		{"ab", "abcabc", -1},
		{"^a", "abcabc", -1},
		{"a", "abcabc", 0},
		{"a", "abcabc", 1},
		{"a", "abcabc", 2},
		{"a", "abcabc", 3},
		{"b", "abcabc", 1},
		{"b", "abcabc", 2},
		{"b", "abcabc", 3},
	} {
		var expected []interface{}
		for _, ex := range regexp.MustCompile(d.pattern).Split(d.text, d.count) {
			expected = append(expected, ex)
		}
		require.Module(t, "text").Call("re_split", d.pattern, d.text, d.count).Expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
		require.Module(t, "text").Call("re_compile", d.pattern).Call("split", d.text, d.count).Expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
	}
}

func TestText(t *testing.T) {
	require.Module(t, "text").Call("compare", "", "").Expect(0)
	require.Module(t, "text").Call("compare", "", "a").Expect(-1)
	require.Module(t, "text").Call("compare", "a", "").Expect(1)
	require.Module(t, "text").Call("compare", "a", "a").Expect(0)
	require.Module(t, "text").Call("compare", "a", "b").Expect(-1)
	require.Module(t, "text").Call("compare", "b", "a").Expect(1)
	require.Module(t, "text").Call("compare", "abcde", "abcde").Expect(0)
	require.Module(t, "text").Call("compare", "abcde", "abcdf").Expect(-1)
	require.Module(t, "text").Call("compare", "abcdf", "abcde").Expect(1)

	require.Module(t, "text").Call("contains", "", "").Expect(true)
	require.Module(t, "text").Call("contains", "", "a").Expect(false)
	require.Module(t, "text").Call("contains", "a", "").Expect(true)
	require.Module(t, "text").Call("contains", "a", "a").Expect(true)
	require.Module(t, "text").Call("contains", "abcde", "a").Expect(true)
	require.Module(t, "text").Call("contains", "abcde", "abcde").Expect(true)
	require.Module(t, "text").Call("contains", "abc", "abcde").Expect(false)
	require.Module(t, "text").Call("contains", "ab cd", "bc").Expect(false)

	require.Module(t, "text").Call("replace", "", "", "", -1).Expect("")
	require.Module(t, "text").Call("replace", "abcd", "a", "x", -1).Expect("xbcd")
	require.Module(t, "text").Call("replace", "aaaa", "a", "x", -1).Expect("xxxx")
	require.Module(t, "text").Call("replace", "aaaa", "a", "x", 0).Expect("aaaa")
	require.Module(t, "text").Call("replace", "aaaa", "a", "x", 2).Expect("xxaa")
	require.Module(t, "text").Call("replace", "abcd", "bc", "x", -1).Expect("axd")

	require.Module(t, "text").Call("format_bool", true).Expect("true")
	require.Module(t, "text").Call("format_bool", false).Expect("false")
	require.Module(t, "text").Call("format_float", -19.84, 'f', -1, 64).Expect("-19.84")
	require.Module(t, "text").Call("format_int", -1984, 10).Expect("-1984")
	require.Module(t, "text").Call("format_int", 1984, 8).Expect("3700")
	require.Module(t, "text").Call("parse_bool", "true").Expect(true)
	require.Module(t, "text").Call("parse_bool", "0").Expect(false)
	require.Module(t, "text").Call("parse_float", "-19.84", 64).Expect(-19.84)
	require.Module(t, "text").Call("parse_int", "-1984", 10, 64).Expect(-1984)
}

func TestReplaceLimit(t *testing.T) {
	curMaxStringLen := vm.MaxStringLen
	defer func() { vm.MaxStringLen = curMaxStringLen }()
	vm.MaxStringLen = 12

	require.Module(t, "text").Call("replace", "123456789012", "1", "x", -1).Expect("x234567890x2")
	require.Module(t, "text").Call("replace", "123456789012", "12", "x", -1).Expect("x34567890x")
	require.Module(t, "text").Call("replace", "123456789012", "1", "xy", -1).ExpectError()
	require.Module(t, "text").Call("replace", "123456789012", "0", "xy", -1).ExpectError()
	require.Module(t, "text").Call("replace", "123456789012", "012", "xyz", -1).Expect("123456789xyz")
	require.Module(t, "text").Call("replace", "123456789012", "012", "xyzz", -1).ExpectError()

	require.Module(t, "text").Call("re_replace", "1", "123456789012", "x").Expect("x234567890x2")
	require.Module(t, "text").Call("re_replace", "12", "123456789012", "x").Expect("x34567890x")
	require.Module(t, "text").Call("re_replace", "1", "123456789012", "xy").ExpectError()
	require.Module(t, "text").Call("re_replace", "1(2)", "123456789012", "x$1").Expect("x234567890x2")
	require.Module(t, "text").Call("re_replace", "(1)(2)", "123456789012", "$2$1").Expect("213456789021")
	require.Module(t, "text").Call("re_replace", "(1)(2)", "123456789012", "${2}${1}x").ExpectError()
}

func TestTextRepeat(t *testing.T) {
	curMaxStringLen := vm.MaxStringLen
	defer func() { vm.MaxStringLen = curMaxStringLen }()
	vm.MaxStringLen = 12

	require.Module(t, "text").Call("repeat", "1234", "3").Expect("123412341234")
	require.Module(t, "text").Call("repeat", "1234", "4").ExpectError()
	require.Module(t, "text").Call("repeat", "1", "12").Expect("111111111111")
	require.Module(t, "text").Call("repeat", "1", "13").ExpectError()
}

func TestSubstr(t *testing.T) {
	require.Module(t, "text").Call("substr", "", 0, 0).Expect("")
	require.Module(t, "text").Call("substr", "abcdef", 0, 3).Expect("abc")
	require.Module(t, "text").Call("substr", "abcdef", 0, 6).Expect("abcdef")
	require.Module(t, "text").Call("substr", "abcdef", 0, 10).Expect("abcdef")
	require.Module(t, "text").Call("substr", "abcdef", -10, 10).Expect("abcdef")
	require.Module(t, "text").Call("substr", "abcdef", 0).Expect("abcdef")
	require.Module(t, "text").Call("substr", "abcdef", 3).Expect("def")

	require.Module(t, "text").Call("substr", "", 10, 0).ExpectError()
	require.Module(t, "text").Call("substr", "", "10", 0).ExpectError()
	require.Module(t, "text").Call("substr", "", 10, "0").ExpectError()
	require.Module(t, "text").Call("substr", "", "10", "0").ExpectError()

	require.Module(t, "text").Call("substr", 0, 0, 1).Expect("0")
	require.Module(t, "text").Call("substr", 123, 0, 1).Expect("1")
	require.Module(t, "text").Call("substr", 123.456, 4, 7).Expect("456")
}

func TestPadLeft(t *testing.T) {
	require.Module(t, "text").Call("pad_left", "ab", 7, 0).Expect("00000ab")
	require.Module(t, "text").Call("pad_right", "ab", 7, 0).Expect("ab00000")
	require.Module(t, "text").Call("pad_left", "ab", 7, "+-").Expect("-+-+-ab")
	require.Module(t, "text").Call("pad_right", "ab", 7, "+-").Expect("ab+-+-+")
}
