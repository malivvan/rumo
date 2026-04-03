// Package codec is a copy of https://github.com/deneonet/benc/std without unsafe imports
//
// # MIT License
//
// Copyright (c) 2023-2024 The Benc Project (https://github.com/deneonet/benc)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package encoding

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func SizeAll(sizers ...func() int) int {
	s := 0
	for _, sizer := range sizers {
		ts := sizer()
		if ts == 0 {
			return 0
		}
		s += ts
	}
	return s
}

func SkipAll(b []byte, skipers ...func(n int, b []byte) (int, error)) error {
	n := 0
	var err error
	for i, skiper := range skipers {
		n, err = skiper(n, b)
		if err != nil {
			return fmt.Errorf("(skip) at idx %d: error: %s", i, err.Error())
		}
	}
	if n != len(b) {
		return errors.New("skip failed: something doesn't match in the marshal- and skip progrss")
	}
	return nil
}

func SkipOnceVerify(b []byte, skiper func(n int, b []byte) (int, error)) error {
	n := 0
	var err error
	n, err = skiper(n, b)
	if err != nil {
		return fmt.Errorf("skip: error: %s", err.Error())
	}
	if n != len(b) {
		return errors.New("skip failed: something doesn't match in the marshal- and skip progrss")
	}
	return nil
}

func MarshalAll(s int, values []any, marshals ...func(n int, b []byte, v any) int) ([]byte, error) {
	n := 0
	b := make([]byte, s)
	for i, marshal := range marshals {
		n = marshal(n, b, values[i])
		if n == 0 {
			// error already logged
			return nil, nil
		}
	}
	if n != len(b) {
		return nil, errors.New("marshal failed: something doesn't match in the marshal- and size progrss")
	}
	return b, nil
}

func UnmarshalAllVerify(b []byte, values []any, unmarshals ...func(n int, b []byte) (int, any, error)) error {
	n := 0
	var v any
	var err error
	for i, unmarshal := range unmarshals {
		n, v, err = unmarshal(n, b)
		if err != nil {
			return fmt.Errorf("(unmarshal) at idx %d: error: %s", i, err.Error())
		}
		if !reflect.DeepEqual(v, values[i]) {
			return fmt.Errorf("(unmarshal) at idx %d: no match: expected %v, got %v --- (%T - %T)", i, values[i], v, values[i], v)
		}
	}
	if n != len(b) {
		return errors.New("unmarshal failed: something doesn't match in the marshal- and unmarshal progrss")
	}
	return nil
}

func UnmarshalAllVerifyError(expected error, buffers [][]byte, unmarshals ...func(n int, b []byte) (int, any, error)) error {
	var err error
	for i, unmarshal := range unmarshals {
		_, _, err = unmarshal(0, buffers[i])
		if err != expected {
			return fmt.Errorf("(unmarshal) at idx %d: expected a %s error", i, expected)
		}
	}
	return nil
}

func SkipAllVerifyError(expected error, buffers [][]byte, skipers ...func(n int, b []byte) (int, error)) error {
	var err error
	for i, skiper := range skipers {
		_, err = skiper(0, buffers[i])
		if err != expected {
			return fmt.Errorf("(skip) at idx %d: expected a %s error, got %s", i, expected, err)
		}
	}
	return nil
}

func TestDataTypes(t *testing.T) {
	testStr := "Hello World!"
	sizeTestStr := func() int {
		return SizeString(testStr)
	}

	testBs := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	s := SizeAll(SizeBool, SizeBool, SizeByte, SizeFloat32, SizeFloat64, func() int { return SizeInt(math.MaxInt) }, SizeInt16, SizeInt32, SizeInt64, func() int { return SizeUint(math.MaxUint) }, SizeUint16, SizeUint32, SizeUint64,
		sizeTestStr, sizeTestStr, func() int {
			return SizeBytes(testBs)
		})

	values := []any{true, false, byte(128), rand.Float32(), rand.Float64(), int(math.MaxInt), int16(16), rand.Int31(), rand.Int63(), uint(math.MaxUint), uint16(160), rand.Uint32(), rand.Uint64(), testStr, testStr, testBs}
	buf, err := MarshalAll(s, values,
		func(n int, b []byte, v any) int { return MarshalBool(n, b, v.(bool)) },
		func(n int, b []byte, v any) int { return MarshalBool(n, b, v.(bool)) },
		func(n int, b []byte, v any) int { return MarshalByte(n, b, v.(byte)) },
		func(n int, b []byte, v any) int { return MarshalFloat32(n, b, v.(float32)) },
		func(n int, b []byte, v any) int { return MarshalFloat64(n, b, v.(float64)) },
		func(n int, b []byte, v any) int { return MarshalInt(n, b, v.(int)) },
		func(n int, b []byte, v any) int { return MarshalInt16(n, b, v.(int16)) },
		func(n int, b []byte, v any) int { return MarshalInt32(n, b, v.(int32)) },
		func(n int, b []byte, v any) int { return MarshalInt64(n, b, v.(int64)) },
		func(n int, b []byte, v any) int { return MarshalUint(n, b, v.(uint)) },
		func(n int, b []byte, v any) int { return MarshalUint16(n, b, v.(uint16)) },
		func(n int, b []byte, v any) int { return MarshalUint32(n, b, v.(uint32)) },
		func(n int, b []byte, v any) int { return MarshalUint64(n, b, v.(uint64)) },
		func(n int, b []byte, v any) int { return MarshalString(n, b, v.(string)) },
		func(n int, b []byte, v any) int { return MarshalString(n, b, v.(string)) },
		func(n int, b []byte, v any) int { return MarshalBytes(n, b, v.([]byte)) },
	)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err = SkipAll(buf, SkipBool, SkipBool, SkipByte, SkipFloat32, SkipFloat64, SkipVarint, SkipInt16, SkipInt32, SkipInt64, SkipVarint, SkipUint16, SkipUint32, SkipUint64, SkipString, SkipString, SkipBytes); err != nil {
		t.Fatal(err.Error())
	}

	if err = UnmarshalAllVerify(buf, values,
		func(n int, b []byte) (int, any, error) { return UnmarshalBool(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBool(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalByte(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalFloat32(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalFloat64(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt16(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt32(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt64(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint16(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint32(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint64(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
	); err != nil {
		t.Fatal(err.Error())
	}
}

func TestErrBufTooSmall(t *testing.T) {
	buffers := [][]byte{{}, {}, {1, 2, 3}, {1, 2, 3, 4, 5, 6, 7}, {}, {1}, {1, 2, 3}, {1, 2, 3, 4, 5, 6, 7}, {}, {1}, {1, 2, 3}, {1, 2, 3, 4, 5, 6, 7}, {}, {2, 0}, {4, 1, 2, 3}, {8, 1, 2, 3, 4, 5, 6, 7}, {}, {2, 0}, {4, 1, 2, 3}, {8, 1, 2, 3, 4, 5, 6, 7}, {}, {2, 0}, {4, 1, 2, 3}, {8, 1, 2, 3, 4, 5, 6, 7}, {}, {2, 0}, {4, 1, 2, 3}, {8, 1, 2, 3, 4, 5, 6, 7}, {}, {2, 0}, {4, 1, 2, 3}, {8, 1, 2, 3, 4, 5, 6, 7}}
	if err := UnmarshalAllVerifyError(ErrBufTooSmall, buffers,
		func(n int, b []byte) (int, any, error) { return UnmarshalBool(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalByte(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalFloat32(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalFloat64(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt16(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt32(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalInt64(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint16(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint32(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalUint64(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalSlice[byte](n, b, UnmarshalByte) },
		func(n int, b []byte) (int, any, error) { return UnmarshalSlice[byte](n, b, UnmarshalByte) },
		func(n int, b []byte) (int, any, error) { return UnmarshalSlice[byte](n, b, UnmarshalByte) },
		func(n int, b []byte) (int, any, error) { return UnmarshalSlice[byte](n, b, UnmarshalByte) },
		func(n int, b []byte) (int, any, error) {
			return UnmarshalMap[byte, byte](n, b, UnmarshalByte, UnmarshalByte)
		},
		func(n int, b []byte) (int, any, error) {
			return UnmarshalMap[byte, byte](n, b, UnmarshalByte, UnmarshalByte)
		},
		func(n int, b []byte) (int, any, error) {
			return UnmarshalMap[byte, byte](n, b, UnmarshalByte, UnmarshalByte)
		},
		func(n int, b []byte) (int, any, error) {
			return UnmarshalMap[byte, byte](n, b, UnmarshalByte, UnmarshalByte)
		},
	); err != nil {
		t.Fatal(err.Error())
	}

	skipSliceOfBytes := func(n int, b []byte) (int, error) { return SkipSlice(n, b) }
	skipMapOfBytes := func(n int, b []byte) (int, error) { return SkipMap(n, b) }
	if err := SkipAllVerifyError(ErrBufTooSmall, buffers, SkipBool, SkipByte, SkipFloat32, SkipFloat64, SkipVarint, SkipInt16, SkipInt32, SkipInt64, SkipVarint, SkipUint16, SkipUint32, SkipUint64, SkipString, SkipString, SkipString, SkipString, SkipString, SkipString, SkipString, SkipString, SkipBytes, SkipBytes, SkipBytes, SkipBytes, skipSliceOfBytes, skipSliceOfBytes, skipSliceOfBytes, skipSliceOfBytes, skipMapOfBytes, skipMapOfBytes, skipMapOfBytes, skipMapOfBytes); err != nil {
		t.Fatal(err.Error())
	}
}

func TestErrBufTooSmall_2(t *testing.T) {
	buffers := [][]byte{{}, {2, 0}, {}, {2, 0}, {}, {2, 0}, {}, {10, 0, 0, 0, 1}, {}, {10, 0, 0, 0, 1}}
	if err := UnmarshalAllVerifyError(ErrBufTooSmall, buffers,
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalString(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalBytes(n, b) },
		func(n int, b []byte) (int, any, error) { return UnmarshalSlice[byte](n, b, UnmarshalByte) },
		func(n int, b []byte) (int, any, error) { return UnmarshalSlice[byte](n, b, UnmarshalByte) },
		func(n int, b []byte) (int, any, error) {
			return UnmarshalMap[byte, byte](n, b, UnmarshalByte, UnmarshalByte)
		},
		func(n int, b []byte) (int, any, error) {
			return UnmarshalMap[byte, byte](n, b, UnmarshalByte, UnmarshalByte)
		},
	); err != nil {
		t.Fatal(err.Error())
	}

	skipSliceOfBytes := func(n int, b []byte) (int, error) { return SkipSlice(n, b) }
	skipMapOfBytes := func(n int, b []byte) (int, error) { return SkipMap(n, b) }
	if err := SkipAllVerifyError(ErrBufTooSmall, buffers, SkipString, SkipString, SkipString, SkipString, SkipBytes, SkipBytes, skipSliceOfBytes, skipSliceOfBytes, skipMapOfBytes, skipMapOfBytes); err != nil {
		t.Fatal(err.Error())
	}
}

func TestSlices(t *testing.T) {
	slice := []string{"sliceelement1", "sliceelement2", "sliceelement3", "sliceelement4", "sliceelement5"}
	s := SizeSlice(slice, SizeString)
	buf := make([]byte, s)
	MarshalSlice(0, buf, slice, MarshalString)

	if err := SkipOnceVerify(buf, func(n int, b []byte) (int, error) {
		return SkipSlice(n, b)
	}); err != nil {
		t.Fatal(err.Error())
	}

	_, retSlice, err := UnmarshalSlice[string](0, buf, UnmarshalString)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !reflect.DeepEqual(retSlice, slice) {
		t.Logf("org %v\ndec %v", slice, retSlice)
		t.Fatal("no match!")
	}
}

func TestMaps(t *testing.T) {
	m := make(map[string]string)
	m["mapkey1"] = "mapvalue1"
	m["mapkey2"] = "mapvalue2"
	m["mapkey3"] = "mapvalue3"
	m["mapkey4"] = "mapvalue4"
	m["mapkey5"] = "mapvalue5"

	s := SizeMap(m, SizeString, SizeString)
	buf := make([]byte, s)
	MarshalMap(0, buf, m, MarshalString, MarshalString)
	fmt.Println(buf)

	if err := SkipOnceVerify(buf, func(n int, b []byte) (int, error) {
		return SkipMap(n, b)
	}); err != nil {
		t.Fatal(err.Error())
	}

	_, retMap, err := UnmarshalMap[string, string](0, buf, UnmarshalString, UnmarshalString)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !reflect.DeepEqual(retMap, m) {
		t.Logf("org %v\ndec %v", m, retMap)
		t.Fatal("no match!")
	}
}

func TestMaps_2(t *testing.T) {
	m := make(map[int32]string)
	m[1] = "mapvalue1"
	m[2] = "mapvalue2"
	m[3] = "mapvalue3"
	m[4] = "mapvalue4"
	m[5] = "mapvalue5"

	s := SizeMap(m, SizeInt32, SizeString)
	buf := make([]byte, s)
	MarshalMap(0, buf, m, MarshalInt32, MarshalString)
	fmt.Println(buf)

	if err := SkipOnceVerify(buf, func(n int, b []byte) (int, error) {
		return SkipMap(n, b)
	}); err != nil {
		t.Fatal(err.Error())
	}

	_, retMap, err := UnmarshalMap[int32, string](0, buf, UnmarshalInt32, UnmarshalString)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !reflect.DeepEqual(retMap, m) {
		t.Logf("org %v\ndec %v", m, retMap)
		t.Fatal("no match!")
	}
}

func TestEmptyString(t *testing.T) {
	str := ""

	s := SizeString(str)
	buf := make([]byte, s)
	MarshalString(0, buf, str)

	if err := SkipOnceVerify(buf, SkipString); err != nil {
		t.Fatal(err.Error())
	}

	_, retStr, err := UnmarshalString(0, buf)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !reflect.DeepEqual(retStr, str) {
		t.Logf("org %v\ndec %v", str, retStr)
		t.Fatal("no match!")
	}
}

func TestLongString(t *testing.T) {
	str := ""
	for i := 0; i < math.MaxUint16+1; i++ {
		str += "H"
	}

	s := SizeString(str)
	buf := make([]byte, s)
	MarshalString(0, buf, str)

	if err := SkipOnceVerify(buf, SkipString); err != nil {
		t.Fatal(err.Error())
	}

	_, retStr, err := UnmarshalString(0, buf)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !reflect.DeepEqual(retStr, str) {
		t.Logf("org %v\ndec %v", str, retStr)
		t.Fatal("no match!")
	}
}

type ComplexData struct {
	ID              int64
	Title           string
	Items           []SubItem
	Metadata        map[string]int32
	SubData         SubComplexData
	LargeBinaryData [][]byte
	HugeList        []int64
}

func (complexData *ComplexData) SizePlain() (s int) {
	s += SizeInt64()
	s += SizeString(complexData.Title)
	s += SizeSlice(complexData.Items, func(s SubItem) int { return s.SizePlain() })
	s += SizeMap(complexData.Metadata, SizeString, SizeInt32)
	s += complexData.SubData.SizePlain()
	s += SizeSlice(complexData.LargeBinaryData, SizeBytes)
	s += SizeSlice(complexData.HugeList, SizeInt64)
	return
}

func (complexData *ComplexData) MarshalPlain(tn int, b []byte) (n int) {
	n = tn
	n = MarshalInt64(n, b, complexData.ID)
	n = MarshalString(n, b, complexData.Title)
	n = MarshalSlice(n, b, complexData.Items, func(n int, b []byte, s SubItem) int { return s.MarshalPlain(n, b) })
	n = MarshalMap(n, b, complexData.Metadata, MarshalString, MarshalInt32)
	n = complexData.SubData.MarshalPlain(n, b)
	n = MarshalSlice(n, b, complexData.LargeBinaryData, MarshalBytes)
	n = MarshalSlice(n, b, complexData.HugeList, MarshalInt64)
	return n
}

func (complexData *ComplexData) UnmarshalPlain(tn int, b []byte) (n int, err error) {
	n = tn
	if n, complexData.ID, err = UnmarshalInt64(n, b); err != nil {
		return
	}
	if n, complexData.Title, err = UnmarshalString(n, b); err != nil {
		return
	}
	if n, complexData.Items, err = UnmarshalSlice[SubItem](n, b, func(n int, b []byte, s *SubItem) (int, error) { return s.UnmarshalPlain(n, b) }); err != nil {
		return
	}
	if n, complexData.Metadata, err = UnmarshalMap[string, int32](n, b, UnmarshalString, UnmarshalInt32); err != nil {
		return
	}
	if n, err = complexData.SubData.UnmarshalPlain(n, b); err != nil {
		return
	}
	if n, complexData.LargeBinaryData, err = UnmarshalSlice[[]byte](n, b, UnmarshalBytes); err != nil {
		return
	}
	if n, complexData.HugeList, err = UnmarshalSlice[int64](n, b, UnmarshalInt64); err != nil {
		return
	}
	return
}

type SubItem struct {
	SubID       int32
	Description string
	SubItems    []SubSubItem
}

func (subItem *SubItem) SizePlain() (s int) {
	s += SizeInt32()
	s += SizeString(subItem.Description)
	s += SizeSlice(subItem.SubItems, func(s SubSubItem) int { return s.SizePlain() })
	return
}

func (subItem *SubItem) MarshalPlain(tn int, b []byte) (n int) {
	n = tn
	n = MarshalInt32(n, b, subItem.SubID)
	n = MarshalString(n, b, subItem.Description)
	n = MarshalSlice(n, b, subItem.SubItems, func(n int, b []byte, s SubSubItem) int { return s.MarshalPlain(n, b) })
	return n
}

func (subItem *SubItem) UnmarshalPlain(tn int, b []byte) (n int, err error) {
	n = tn
	if n, subItem.SubID, err = UnmarshalInt32(n, b); err != nil {
		return
	}
	if n, subItem.Description, err = UnmarshalString(n, b); err != nil {
		return
	}
	if n, subItem.SubItems, err = UnmarshalSlice[SubSubItem](n, b, func(n int, b []byte, s *SubSubItem) (int, error) { return s.UnmarshalPlain(n, b) }); err != nil {
		return
	}
	return
}

type SubSubItem struct {
	SubSubID   string
	SubSubData []byte
}

func (subSubItem *SubSubItem) SizePlain() (s int) {
	s += SizeString(subSubItem.SubSubID)
	s += SizeBytes(subSubItem.SubSubData)
	return
}

func (subSubItem *SubSubItem) MarshalPlain(tn int, b []byte) (n int) {
	n = tn
	n = MarshalString(n, b, subSubItem.SubSubID)
	n = MarshalBytes(n, b, subSubItem.SubSubData)
	return n
}

func (subSubItem *SubSubItem) UnmarshalPlain(tn int, b []byte) (n int, err error) {
	n = tn
	if n, subSubItem.SubSubID, err = UnmarshalString(n, b); err != nil {
		return
	}
	if n, subSubItem.SubSubData, err = UnmarshalBytes(n, b); err != nil {
		return
	}
	return
}

type SubComplexData struct {
	SubID         int32
	SubTitle      string
	SubBinaryData [][]byte
	SubItems      []SubItem
	SubMetadata   map[string]string
}

func (subComplexData *SubComplexData) SizePlain() (s int) {
	s += SizeInt32()
	s += SizeString(subComplexData.SubTitle)
	s += SizeSlice(subComplexData.SubBinaryData, SizeBytes)
	s += SizeSlice(subComplexData.SubItems, func(s SubItem) int { return s.SizePlain() })
	s += SizeMap(subComplexData.SubMetadata, SizeString, SizeString)
	return
}

func (subComplexData *SubComplexData) MarshalPlain(tn int, b []byte) (n int) {
	n = tn
	n = MarshalInt32(n, b, subComplexData.SubID)
	n = MarshalString(n, b, subComplexData.SubTitle)
	n = MarshalSlice(n, b, subComplexData.SubBinaryData, MarshalBytes)
	n = MarshalSlice(n, b, subComplexData.SubItems, func(n int, b []byte, s SubItem) int { return s.MarshalPlain(n, b) })
	n = MarshalMap(n, b, subComplexData.SubMetadata, MarshalString, MarshalString)
	return n
}

func (subComplexData *SubComplexData) UnmarshalPlain(tn int, b []byte) (n int, err error) {
	n = tn
	if n, subComplexData.SubID, err = UnmarshalInt32(n, b); err != nil {
		return
	}
	if n, subComplexData.SubTitle, err = UnmarshalString(n, b); err != nil {
		return
	}
	if n, subComplexData.SubBinaryData, err = UnmarshalSlice[[]byte](n, b, UnmarshalBytes); err != nil {
		return
	}
	if n, subComplexData.SubItems, err = UnmarshalSlice[SubItem](n, b, func(n int, b []byte, s *SubItem) (int, error) { return s.UnmarshalPlain(n, b) }); err != nil {
		return
	}
	if n, subComplexData.SubMetadata, err = UnmarshalMap[string, string](n, b, UnmarshalString, UnmarshalString); err != nil {
		return
	}
	return
}

func TestComplex(t *testing.T) {
	data := ComplexData{
		ID:    12345,
		Title: "Example Complex Data",
		Items: []SubItem{
			{
				SubID:       1,
				Description: "SubItem 1",
				SubItems: []SubSubItem{
					{
						SubSubID:   "subsub1",
						SubSubData: []byte{0x01, 0x02, 0x03},
					},
				},
			},
		},
		Metadata: map[string]int32{
			"key1": 10,
			"key2": 20,
		},
		SubData: SubComplexData{
			SubID:    999,
			SubTitle: "Sub Complex Data",
			SubBinaryData: [][]byte{
				{0x11, 0x22, 0x33},
				{0x44, 0x55, 0x66},
			},
			SubItems: []SubItem{
				{
					SubID:       2,
					Description: "SubItem 2",
					SubItems: []SubSubItem{
						{
							SubSubID:   "subsub2",
							SubSubData: []byte{0xAA, 0xBB, 0xCC},
						},
					},
				},
			},
			SubMetadata: map[string]string{
				"meta1": "value1",
				"meta2": "value2",
			},
		},
		LargeBinaryData: [][]byte{
			{0xFF, 0xEE, 0xDD},
		},
		HugeList: []int64{1000000, 2000000, 3000000},
	}

	s := data.SizePlain()
	b := make([]byte, s)
	data.MarshalPlain(0, b)

	var retData ComplexData
	if _, err := retData.UnmarshalPlain(0, b); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data, retData) {
		t.Fatalf("no match\norg: %v\ndec: %v\n", data, retData)
	}
}
