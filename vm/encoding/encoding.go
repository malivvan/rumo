// Package encoding provides functions to marshal and unmarshal data types.
package encoding

import (
	"encoding/binary"
	"errors"
	"math"
	"strconv"
)

// ErrBufTooSmall is returned when the buffer is too small to read/write.
var ErrBufTooSmall = errors.New("buffer too small")

// ErrOverflow is returned when a varint overflows a 64-bit integer.
var ErrOverflow = errors.New("varint overflows a 64-bit integer")

// MarshalFunc is a function that marshals a value of type T into a byte slice.
type MarshalFunc[T any] func(n int, b []byte, t T) int

// SkipString skips a string in the buffer.
func SkipString(n int, b []byte) (int, error) {
	n, us, err := UnmarshalUint(n, b)
	if err != nil {
		return 0, err
	}
	s := int(us)

	if len(b)-n < s {
		return n, ErrBufTooSmall
	}
	return n + s, nil
}

// SizeString returns the bytes needed to marshal a string.
func SizeString(str string) int {
	v := len(str)
	return v + SizeUint(uint(v))
}

// MarshalString marshals a string into the buffer.
func MarshalString(n int, b []byte, str string) int {
	n = MarshalUint(n, b, uint(len(str)))
	return n + copy(b[n:], str)
}

// UnmarshalString unmarshals a string from the buffer.
func UnmarshalString(n int, b []byte) (int, string, error) {
	n, us, err := UnmarshalUint(n, b)
	if err != nil {
		return 0, "", err
	}
	s := int(us)

	if len(b)-n < s {
		return n, "", ErrBufTooSmall
	}
	return n + s, string(b[n : n+s]), nil
}

// StringHeader is a string header.
type StringHeader struct {
	Data *byte
	Len  int
}

// SkipSlice skips a slice in the buffer.
func SkipSlice(n int, b []byte) (int, error) {
	lb := len(b)
	for {
		if lb-n < 4 {
			return 0, ErrBufTooSmall
		}

		if b[n] == 1 && b[n+1] == 1 && b[n+2] == 1 && b[n+3] == 1 {
			return n + 4, nil
		}
		n++
	}
}

// SizeSlice returns the bytes needed to marshal a slice.
func SizeSlice[T any](slice []T, sizer interface{}) (s int) {
	v := len(slice)
	s += 4 + SizeUint(uint(v))

	switch p := sizer.(type) {
	case func() int:
		for i := 0; i < v; i++ {
			s += p()
		}
	case func(T) int:
		for _, t := range slice {
			s += p(t)
		}
	default:
		panic("benc: invalid `sizer` provided in `SizeSlice`")
	}
	return
}

// MarshalSlice marshals a slice into the buffer.
func MarshalSlice[T any](n int, b []byte, slice []T, marshaler MarshalFunc[T]) int {
	n = MarshalUint(n, b, uint(len(slice)))
	for _, t := range slice {
		n = marshaler(n, b, t)
	}

	u := b[n : n+4]
	_ = u[3]
	u[0] = byte(1)
	u[1] = byte(1)
	u[2] = byte(1)
	u[3] = byte(1)
	return n + 4
}

// UnmarshalSlice unmarshals a slice from the buffer.
func UnmarshalSlice[T any](n int, b []byte, unmarshaler interface{}) (int, []T, error) {
	n, us, err := UnmarshalUint(n, b)
	if err != nil {
		return 0, nil, err
	}
	s := int(us)

	var t T
	ts := make([]T, s)

	switch p := unmarshaler.(type) {
	case func(n int, b []byte) (int, T, error):
		for i := 0; i < s; i++ {
			n, t, err = p(n, b)
			if err != nil {
				return 0, nil, err
			}

			ts[i] = t
		}
	case func(n int, b []byte, v *T) (int, error):
		for i := 0; i < s; i++ {
			n, err = p(n, b, &ts[i])
			if err != nil {
				return 0, nil, err
			}
		}
	default:
		panic("benc: invalid `unmarshaler` provided in `UnmarshalSlice`")
	}

	return n + 4, ts, nil
}

// SkipMap skips a map in the buffer.
func SkipMap(n int, b []byte) (int, error) {
	lb := len(b)

	for {
		if lb-n < 4 {
			return 0, ErrBufTooSmall
		}

		if b[n] == 1 && b[n+1] == 1 && b[n+2] == 1 && b[n+3] == 1 {
			return n + 4, nil
		}
		n++
	}
}

// SizeMap returns the bytes needed to marshal a map.
func SizeMap[K comparable, V any](m map[K]V, kSizer interface{}, vSizer interface{}) (s int) {
	s += 4 + SizeUint(uint(len(m)))

	for k, v := range m {
		switch p := kSizer.(type) {
		case func() int:
			s += p()
		case func(K) int:
			s += p(k)
		default:
			panic("benc: invalid `kSizer` provided in `SizeMap`")
		}

		switch p := vSizer.(type) {
		case func() int:
			s += p()
		case func(V) int:
			s += p(v)
		default:
			panic("benc: invalid `vSizer` provided in `SizeMap`")
		}
	}
	return
}

// MarshalMap marshals a map into the buffer.
func MarshalMap[K comparable, V any](n int, b []byte, m map[K]V, kMarshaler MarshalFunc[K], vMarshaler MarshalFunc[V]) int {
	n = MarshalUint(n, b, uint(len(m)))
	for k, v := range m {
		n = kMarshaler(n, b, k)
		n = vMarshaler(n, b, v)
	}

	u := b[n : n+4]
	_ = u[3]
	u[0] = byte(1)
	u[1] = byte(1)
	u[2] = byte(1)
	u[3] = byte(1)
	return n + 4
}

// UnmarshalMap unmarshals a map from the buffer.
func UnmarshalMap[K comparable, V any](n int, b []byte, kUnmarshaler interface{}, vUnmarshaler interface{}) (int, map[K]V, error) {
	n, us, err := UnmarshalUint(n, b)
	if err != nil {
		return 0, nil, err
	}
	s := int(us)

	var k K
	var v V
	ts := make(map[K]V, s)

	for i := 0; i < s; i++ {
		switch p := kUnmarshaler.(type) {
		case func(n int, b []byte) (int, K, error):
			n, k, err = p(n, b)
			if err != nil {
				return 0, nil, err
			}
		case func(n int, b []byte, k *K) (int, error):
			n, err = p(n, b, &k)
			if err != nil {
				return 0, nil, err
			}
		default:
			panic("benc: invalid `kUnmarshaler` provided in `UnmarshalMap`")
		}

		switch p := vUnmarshaler.(type) {
		case func(n int, b []byte) (int, V, error):
			n, v, err = p(n, b)
			if err != nil {
				return 0, nil, err
			}
		case func(n int, b []byte, v *V) (int, error):
			n, err = p(n, b, &v)
			if err != nil {
				return 0, nil, err
			}
		default:
			panic("benc: invalid `kUnmarshaler` provided in `UnmarshalMap`")
		}

		ts[k] = v
	}

	return n + 4, ts, nil
}

// SkipByte skips a byte in the buffer.
func SkipByte(n int, b []byte) (int, error) {
	if len(b)-n < 1 {
		return n, ErrBufTooSmall
	}
	return n + 1, nil
}

// SizeByte returns the bytes needed to marshal a byte.
func SizeByte() int {
	return 1
}

// MarshalByte marshals a byte into the buffer.
func MarshalByte(n int, b []byte, byt byte) int {
	b[n] = byt
	return n + 1
}

// UnmarshalByte unmarshals a byte from the buffer.
func UnmarshalByte(n int, b []byte) (int, byte, error) {
	if len(b)-n < 1 {
		return n, 0, ErrBufTooSmall
	}
	return n + 1, b[n], nil
}

// SkipBytes skips a byte slice in the buffer.
func SkipBytes(n int, b []byte) (int, error) {
	n, us, err := UnmarshalUint(n, b)
	if err != nil {
		return 0, err
	}
	s := int(us)
	if len(b)-n < s {
		return n, ErrBufTooSmall
	}
	return n + s, nil
}

// SizeBytes returns the bytes needed to marshal a byte slice.
func SizeBytes(bs []byte) int {
	v := len(bs)
	return v + SizeUint(uint(v))
}

// MarshalBytes marshals a byte slice into the buffer.
func MarshalBytes(n int, b []byte, bs []byte) int {
	n = MarshalUint(n, b, uint(len(bs)))
	return n + copy(b[n:], bs)
}

// UnmarshalBytes unmarshals a byte slice from the buffer.
func UnmarshalBytes(n int, b []byte) (int, []byte, error) {
	n, us, err := UnmarshalUint(n, b)
	if err != nil {
		return 0, nil, err
	}
	s := int(us)
	if len(b)-n < s {
		return 0, nil, ErrBufTooSmall
	}
	return n + s, b[n : n+s], nil
}

var maxVarintLenMap = map[int]int{
	64: binary.MaxVarintLen64,
	32: binary.MaxVarintLen32,
}

var maxVarintLen = maxVarintLenMap[strconv.IntSize]

// SkipVarint skips a varint in the buffer.
func SkipVarint(n int, buf []byte) (int, error) {
	for i, b := range buf[n:] {
		if i == maxVarintLen {
			return 0, ErrOverflow
		}
		if b < 0x80 {
			if i == maxVarintLen-1 && b > 1 {
				return 0, ErrOverflow
			}
			return n + i + 1, nil
		}
	}
	return 0, ErrBufTooSmall
}

// SizeInt returns the bytes needed to marshal a signed integer.
func SizeInt(sv int) int {
	v := uint(encodeZigZag(sv))
	i := 0
	for v >= 0x80 {
		v >>= 7
		i++
	}
	return i + 1
}

// MarshalInt marshals a signed integer into the buffer.
func MarshalInt(n int, b []byte, sv int) int {
	v := uint(encodeZigZag(sv))
	i := n
	for v >= 0x80 {
		b[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	b[i] = byte(v)
	return i + 1
}

// UnmarshalInt unmarshals a signed integer from the buffer.
func UnmarshalInt(n int, buf []byte) (int, int, error) {
	var x uint
	var s uint
	for i, b := range buf[n:] {
		if i == maxVarintLen {
			return 0, 0, ErrOverflow
		}
		if b < 0x80 {
			if i == maxVarintLen-1 && b > 1 {
				return 0, 0, ErrOverflow
			}
			return n + i + 1, int(decodeZigZag(x | uint(b)<<s)), nil
		}
		x |= uint(b&0x7f) << s
		s += 7
	}
	return 0, 0, ErrBufTooSmall
}

// SizeUint returns the bytes needed to marshal an unsigned integer.
func SizeUint(v uint) int {
	i := 0
	for v >= 0x80 {
		v >>= 7
		i++
	}
	return i + 1
}

// MarshalUint marshals an unsigned integer into the buffer.
func MarshalUint(n int, b []byte, v uint) int {
	i := n
	for v >= 0x80 {
		b[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	b[i] = byte(v)
	return i + 1
}

// UnmarshalUint unmarshals an unsigned integer from the buffer.
func UnmarshalUint(n int, buf []byte) (int, uint, error) {
	var x uint
	var s uint
	for i, b := range buf[n:] {
		if i == maxVarintLen {
			return 0, 0, ErrOverflow
		}
		if b < 0x80 {
			if i == maxVarintLen-1 && b > 1 {
				return 0, 0, ErrOverflow
			}
			return n + i + 1, x | uint(b)<<s, nil
		}
		x |= uint(b&0x7f) << s
		s += 7
	}
	return 0, 0, ErrBufTooSmall
}

// SkipUint64 skips a uint64 in the buffer.
func SkipUint64(n int, b []byte) (int, error) {
	if len(b)-n < 8 {
		return n, ErrBufTooSmall
	}
	return n + 8, nil
}

// SizeUint64 returns the bytes needed to marshal a uint64.
func SizeUint64() int {
	return 8
}

// MarshalUint64 marshals a uint64 into the buffer.
func MarshalUint64(n int, b []byte, v uint64) int {
	u := b[n : n+8]
	_ = u[7]
	u[0] = byte(v)
	u[1] = byte(v >> 8)
	u[2] = byte(v >> 16)
	u[3] = byte(v >> 24)
	u[4] = byte(v >> 32)
	u[5] = byte(v >> 40)
	u[6] = byte(v >> 48)
	u[7] = byte(v >> 56)
	return n + 8
}

// UnmarshalUint64 unmarshals a uint64 from the buffer.
func UnmarshalUint64(n int, b []byte) (int, uint64, error) {
	if len(b)-n < 8 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+8]
	_ = u[7]
	v := uint64(u[0]) | uint64(u[1])<<8 | uint64(u[2])<<16 | uint64(u[3])<<24 |
		uint64(u[4])<<32 | uint64(u[5])<<40 | uint64(u[6])<<48 | uint64(u[7])<<56
	return n + 8, v, nil
}

// SkipUint32 skips a uint32 in the buffer.
func SkipUint32(n int, b []byte) (int, error) {
	if len(b)-n < 4 {
		return n, ErrBufTooSmall
	}
	return n + 4, nil
}

// SizeUint32 returns the bytes needed to marshal a uint32.
func SizeUint32() int {
	return 4
}

// MarshalUint32 marshals a uint32 into the buffer.
func MarshalUint32(n int, b []byte, v uint32) int {
	u := b[n : n+4]
	_ = u[3]
	u[0] = byte(v)
	u[1] = byte(v >> 8)
	u[2] = byte(v >> 16)
	u[3] = byte(v >> 24)
	return n + 4
}

// UnmarshalUint32 unmarshals a uint32 from the buffer.
func UnmarshalUint32(n int, b []byte) (int, uint32, error) {
	if len(b)-n < 4 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+4]
	_ = u[3]
	v := uint32(u[0]) | uint32(u[1])<<8 | uint32(u[2])<<16 | uint32(u[3])<<24
	return n + 4, v, nil
}

// SkipUint16 skips a uint16 in the buffer.
func SkipUint16(n int, b []byte) (int, error) {
	if len(b)-n < 2 {
		return n, ErrBufTooSmall
	}
	return n + 2, nil
}

// SizeUint16 returns the bytes needed to marshal a uint16.
func SizeUint16() int {
	return 2
}

// MarshalUint16 marshals a uint16 into the buffer.
func MarshalUint16(n int, b []byte, v uint16) int {
	u := b[n : n+2]
	_ = u[1]
	u[0] = byte(v)
	u[1] = byte(v >> 8)
	return n + 2
}

// UnmarshalUint16 unmarshals a uint16 from the buffer.
func UnmarshalUint16(n int, b []byte) (int, uint16, error) {
	if len(b)-n < 2 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+2]
	_ = u[1]
	v := uint16(u[0]) | uint16(u[1])<<8
	return n + 2, v, nil
}

// SkipInt64 skips an int64 in the buffer.
func SkipInt64(n int, b []byte) (int, error) {
	if len(b)-n < 8 {
		return n, ErrBufTooSmall
	}
	return n + 8, nil
}

// SizeInt64 returns the bytes needed to marshal an int64.
func SizeInt64() int {
	return 8
}

// MarshalInt64 marshals an int64 into the buffer.
func MarshalInt64(n int, b []byte, v int64) int {
	v64 := uint64(encodeZigZag(v))
	u := b[n : n+8]
	_ = u[7]
	u[0] = byte(v64)
	u[1] = byte(v64 >> 8)
	u[2] = byte(v64 >> 16)
	u[3] = byte(v64 >> 24)
	u[4] = byte(v64 >> 32)
	u[5] = byte(v64 >> 40)
	u[6] = byte(v64 >> 48)
	u[7] = byte(v64 >> 56)
	return n + 8
}

// UnmarshalInt64 unmarshals an int64 from the buffer.
func UnmarshalInt64(n int, b []byte) (int, int64, error) {
	if len(b)-n < 8 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+8]
	_ = u[7]
	v := uint64(u[0]) | uint64(u[1])<<8 | uint64(u[2])<<16 | uint64(u[3])<<24 |
		uint64(u[4])<<32 | uint64(u[5])<<40 | uint64(u[6])<<48 | uint64(u[7])<<56
	return n + 8, int64(decodeZigZag(v)), nil
}

// SkipInt32 skips an int32 in the buffer.
func SkipInt32(n int, b []byte) (int, error) {
	if len(b)-n < 4 {
		return n, ErrBufTooSmall
	}
	return n + 4, nil
}

// SizeInt32 returns the bytes needed to marshal an int32.
func SizeInt32() int {
	return 4
}

// MarshalInt32 marshals an int32 into the buffer.
func MarshalInt32(n int, b []byte, v int32) int {
	v32 := uint32(encodeZigZag(v))
	u := b[n : n+4]
	_ = u[3]
	u[0] = byte(v32)
	u[1] = byte(v32 >> 8)
	u[2] = byte(v32 >> 16)
	u[3] = byte(v32 >> 24)
	return n + 4
}

// UnmarshalInt32 unmarshals an int32 from the buffer.
func UnmarshalInt32(n int, b []byte) (int, int32, error) {
	if len(b)-n < 4 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+4]
	_ = u[3]
	v := uint32(u[0]) | uint32(u[1])<<8 | uint32(u[2])<<16 | uint32(u[3])<<24
	return n + 4, int32(decodeZigZag(v)), nil
}

// SkipInt16 skips an int16 in the buffer.
func SkipInt16(n int, b []byte) (int, error) {
	if len(b)-n < 2 {
		return n, ErrBufTooSmall
	}
	return n + 2, nil
}

// SizeInt16 returns the bytes needed to marshal an int16.
func SizeInt16() int {
	return 2
}

// MarshalInt16 marshals an int16 into the buffer.
func MarshalInt16(n int, b []byte, v int16) int {
	v16 := uint16(encodeZigZag(v))
	u := b[n : n+2]
	_ = u[1]
	u[0] = byte(v16)
	u[1] = byte(v16 >> 8)
	return n + 2
}

// UnmarshalInt16 unmarshals an int16 from the buffer.
func UnmarshalInt16(n int, b []byte) (int, int16, error) {
	if len(b)-n < 2 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+2]
	_ = u[1]
	v := uint16(u[0]) | uint16(u[1])<<8
	return n + 2, int16(decodeZigZag(v)), nil
}

// SkipFloat64 skips a float64 in the buffer.
func SkipFloat64(n int, b []byte) (int, error) {
	if len(b)-n < 8 {
		return n, ErrBufTooSmall
	}
	return n + 8, nil
}

// SizeFloat64 returns the bytes needed to marshal a float64.
func SizeFloat64() int {
	return 8
}

// MarshalFloat64 marshals a float64 into the buffer.
func MarshalFloat64(n int, b []byte, v float64) int {
	v64 := math.Float64bits(v)
	u := b[n : n+8]
	_ = u[7]
	u[0] = byte(v64)
	u[1] = byte(v64 >> 8)
	u[2] = byte(v64 >> 16)
	u[3] = byte(v64 >> 24)
	u[4] = byte(v64 >> 32)
	u[5] = byte(v64 >> 40)
	u[6] = byte(v64 >> 48)
	u[7] = byte(v64 >> 56)
	return n + 8
}

// UnmarshalFloat64 unmarshals a float64 from the buffer.
func UnmarshalFloat64(n int, b []byte) (int, float64, error) {
	if len(b)-n < 8 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+8]
	_ = u[7]
	v := uint64(u[0]) | uint64(u[1])<<8 | uint64(u[2])<<16 | uint64(u[3])<<24 |
		uint64(u[4])<<32 | uint64(u[5])<<40 | uint64(u[6])<<48 | uint64(u[7])<<56
	return n + 8, math.Float64frombits(v), nil
}

// SkipFloat32 skips a float32 in the buffer.
func SkipFloat32(n int, b []byte) (int, error) {
	if len(b)-n < 4 {
		return n, ErrBufTooSmall
	}
	return n + 4, nil
}

// SizeFloat32 returns the bytes needed to marshal a float32.
func SizeFloat32() int {
	return 4
}

// MarshalFloat32 marshals a float32 into the buffer.
func MarshalFloat32(n int, b []byte, v float32) int {
	v32 := math.Float32bits(v)
	u := b[n : n+4]
	_ = u[3]
	u[0] = byte(v32)
	u[1] = byte(v32 >> 8)
	u[2] = byte(v32 >> 16)
	u[3] = byte(v32 >> 24)
	return n + 4
}

// UnmarshalFloat32 unmarshals a float32 from the buffer.
func UnmarshalFloat32(n int, b []byte) (int, float32, error) {
	if len(b)-n < 4 {
		return n, 0, ErrBufTooSmall
	}
	u := b[n : n+4]
	_ = u[3]
	v := uint32(u[0]) | uint32(u[1])<<8 | uint32(u[2])<<16 | uint32(u[3])<<24
	return n + 4, math.Float32frombits(v), nil
}

// SkipBool skips a bool in the buffer.
func SkipBool(n int, b []byte) (int, error) {
	if len(b)-n < 1 {
		return 0, ErrBufTooSmall
	}
	return n + 1, nil
}

// SizeBool returns the bytes needed to marshal a bool.
func SizeBool() int {
	return 1
}

// MarshalBool marshals a bool into the buffer.
func MarshalBool(n int, b []byte, v bool) int {
	var i byte
	if v {
		i = 1
	}
	b[n] = i
	return n + 1
}

// UnmarshalBool unmarshals a bool from the buffer.
func UnmarshalBool(n int, b []byte) (int, bool, error) {
	if len(b)-n < 1 {
		return 0, false, ErrBufTooSmall
	}
	return n + 1, uint8(b[n]) == 1, nil
}

func encodeZigZag[T Signed](t T) T {
	if t < 0 {
		return ^(t << 1)
	}
	return t << 1
}

func decodeZigZag[T Unsigned](t T) T {
	if t&1 == 1 {
		return ^(t >> 1)
	}
	return t >> 1
}
