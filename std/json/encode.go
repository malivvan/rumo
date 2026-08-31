// A modified version of Go's JSON implementation.

// Copyright 2010, 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/malivvan/rumo/vm"
)

// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

var hex = "0123456789abcdef"

// encoder tracks encode state for a single Encode call. The visited set
// detects cyclic *Array/*Map/*StructInstance graphs — keyed on the typed
// pointer, which is comparable and needs no unsafe tricks.
type encoder struct {
	visited map[vm.Object]struct{}
}

// Encode returns the JSON encoding of the object.
func Encode(o vm.Object) ([]byte, error) {
	e := &encoder{visited: make(map[vm.Object]struct{})}
	return e.encode(o)
}

func (e *encoder) encode(o vm.Object) ([]byte, error) {
	var b []byte

	switch o := o.(type) {
	case *vm.Array:
		if err := e.enter(o); err != nil {
			return nil, err
		}
		defer e.leave(o)
		b = append(b, '[')
		len1 := len(o.Value) - 1
		for idx, elem := range o.Value {
			eb, err := e.encode(elem)
			if err != nil {
				return nil, err
			}
			b = append(b, eb...)
			if idx < len1 {
				b = append(b, ',')
			}
		}
		b = append(b, ']')
	case *vm.Map:
		if err := e.enter(o); err != nil {
			return nil, err
		}
		defer e.leave(o)
		b = append(b, '{')
		// Sort keys so the output is deterministic for the same input.
		keys := make([]string, 0, len(o.Value))
		for key := range o.Value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for idx, key := range keys {
			b = encodeString(b, key)
			b = append(b, ':')
			eb, err := e.encode(o.Value[key])
			if err != nil {
				return nil, err
			}
			b = append(b, eb...)
			if idx < len(keys)-1 {
				b = append(b, ',')
			}
		}
		b = append(b, '}')
	case *vm.StructInstance:
		if err := e.enter(o); err != nil {
			return nil, err
		}
		defer e.leave(o)
		b = append(b, '{')
		keys := make([]string, 0, len(o.Values))
		for key := range o.Values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for idx, key := range keys {
			b = encodeString(b, key)
			b = append(b, ':')
			eb, err := e.encode(o.Values[key])
			if err != nil {
				return nil, err
			}
			b = append(b, eb...)
			if idx < len(keys)-1 {
				b = append(b, ',')
			}
		}
		b = append(b, '}')
	case *vm.Error:
		// Encode the error's message as a JSON string so error values
		// produce valid JSON instead of `{"k":}`.
		msg := "error"
		if o.Value != nil {
			if s, ok := vm.ToString(o.Value); ok {
				msg = s
			} else {
				msg = o.Value.TypeName()
			}
		}
		b = encodeString(b, msg)
	case *vm.Bool:
		if o.IsFalsy() {
			b = strconv.AppendBool(b, false)
		} else {
			b = strconv.AppendBool(b, true)
		}
	case *vm.Bytes:
		b = append(b, '"')
		encodedLen := base64.StdEncoding.EncodedLen(len(o.Value))
		dst := make([]byte, encodedLen)
		base64.StdEncoding.Encode(dst, o.Value)
		b = append(b, dst...)
		b = append(b, '"')
	case *vm.Char:
		b = strconv.AppendInt(b, int64(o.Value), 10)
	case *vm.Float32:
		var y []byte
		f := float64(o.Value)
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, errors.New("unsupported float value")
		}
		abs := math.Abs(f)
		fmt := byte('f')
		if abs != 0 {
			if abs < 1e-6 || abs >= 1e21 {
				fmt = 'e'
			}
		}
		y = strconv.AppendFloat(y, f, fmt, -1, 32)
		if fmt == 'e' {
			n := len(y)
			if n >= 4 && y[n-4] == 'e' && y[n-3] == '-' && y[n-2] == '0' {
				y[n-2] = y[n-1]
				y = y[:n-1]
			}
		}
		b = append(b, y...)
	case *vm.Float64:
		var y []byte

		f := o.Value
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, errors.New("unsupported float value")
		}

		// Convert as if by ES6 number to string conversion.
		// This matches most other JSON generators.
		abs := math.Abs(f)
		fmt := byte('f')
		if abs != 0 {
			if abs < 1e-6 || abs >= 1e21 {
				fmt = 'e'
			}
		}
		y = strconv.AppendFloat(y, f, fmt, -1, 64)
		if fmt == 'e' {
			// clean up e-09 to e-9
			n := len(y)
			if n >= 4 && y[n-4] == 'e' && y[n-3] == '-' && y[n-2] == '0' {
				y[n-2] = y[n-1]
				y = y[:n-1]
			}
		}

		b = append(b, y...)
	case *vm.Int:
		b = strconv.AppendInt(b, o.Value, 10)
	case *vm.Int8:
		b = strconv.AppendInt(b, int64(o.Value), 10)
	case *vm.Int16:
		b = strconv.AppendInt(b, int64(o.Value), 10)
	case *vm.Int32:
		b = strconv.AppendInt(b, int64(o.Value), 10)
	case *vm.Byte:
		b = strconv.AppendInt(b, int64(int8(o.Value)), 10)
	case *vm.Uint8:
		b = strconv.AppendUint(b, uint64(o.Value), 10)
	case *vm.Uint16:
		b = strconv.AppendUint(b, uint64(o.Value), 10)
	case *vm.Uint:
		b = strconv.AppendUint(b, uint64(o.Value), 10)
	case *vm.Uint64:
		b = strconv.AppendUint(b, o.Value, 10)
	case *vm.String:
		// string encoding bug is fixed with newly introduced function
		// encodeString(). See: https://github.com/malivvan/rumo/issues/268
		b = encodeString(b, o.Value)
	case *vm.Undefined:
		b = append(b, "null"...)
	default:
		// Never silently drop a value: that produced invalid JSON like
		// `[1,,3]` in the past. Report the offending type instead.
		return nil, fmt.Errorf("unsupported type: %s", o.TypeName())
	}
	return b, nil
}

// enter registers o in the visited set, failing on cyclic graphs.
func (e *encoder) enter(o vm.Object) error {
	if _, ok := e.visited[o]; ok {
		return errors.New("unsupported value: cyclic data structure")
	}
	e.visited[o] = struct{}{}
	return nil
}

// leave removes o from the visited set (deferred by encode).
func (e *encoder) leave(o vm.Object) {
	delete(e.visited, o)
}

// encodeString encodes given string as JSON string according to
// https://www.json.org/img/string.png
// Implementation is inspired by https://github.com/json-iterator/go
// See encodeStringSlowPath() for more information.
func encodeString(b []byte, val string) []byte {
	valLen := len(val)
	buf := bytes.NewBuffer(b)
	buf.WriteByte('"')

	// write string, the fast path, without utf8 and escape support
	i := 0
	for ; i < valLen; i++ {
		c := val[i]
		if c > 31 && c != '"' && c != '\\' {
			buf.WriteByte(c)
		} else {
			break
		}
	}
	if i == valLen {
		buf.WriteByte('"')
		return buf.Bytes()
	}
	encodeStringSlowPath(buf, i, val, valLen)
	buf.WriteByte('"')
	return buf.Bytes()
}

// encodeStringSlowPath is ported from Go 1.14.2 encoding/json package.
// U+2028 U+2029 JSONP security holes can be fixed with addition call to
// json.html_escape() thus it is removed from the implementation below.
// Note: Invalid runes are not checked as they are checked in original
// implementation.
func encodeStringSlowPath(buf *bytes.Buffer, i int, val string, valLen int) {
	start := i
	for i < valLen {
		if b := val[i]; b < utf8.RuneSelf {
			if safeSet[b] {
				i++
				continue
			}
			if start < i {
				buf.WriteString(val[start:i])
			}
			buf.WriteByte('\\')
			switch b {
			case '\\', '"':
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				buf.WriteString(`u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		i++
		continue
	}
	if start < valLen {
		buf.WriteString(val[start:])
	}
}
