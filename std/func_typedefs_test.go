package std_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/malivvan/vv/std"
	"github.com/malivvan/vv/vm"
	"github.com/malivvan/vv/vm/require"
)

func TestFuncAIR(t *testing.T) {
	uf := std.FuncAIR(func(int) {})
	ret, err := funcCall(uf, &vm.Int{Value: 10})
	require.NoError(t, err)
	require.Equal(t, vm.UndefinedValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAR(t *testing.T) {
	uf := std.FuncAR(func() {})
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, vm.UndefinedValue, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARI(t *testing.T) {
	uf := std.FuncARI(func() int { return 10 })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 10}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARE(t *testing.T) {
	uf := std.FuncARE(func() error { return nil })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	uf = std.FuncARE(func() error { return errors.New("some error") })
	ret, err = funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Error{
		Value: &vm.String{Value: "some error"},
	}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARIsE(t *testing.T) {
	uf := std.FuncARIsE(func() ([]int, error) {
		return []int{1, 2, 3}, nil
	})
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, array(&vm.Int{Value: 1},
		&vm.Int{Value: 2}, &vm.Int{Value: 3}), ret)
	uf = std.FuncARIsE(func() ([]int, error) {
		return nil, errors.New("some error")
	})
	ret, err = funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Error{
		Value: &vm.String{Value: "some error"},
	}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARS(t *testing.T) {
	uf := std.FuncARS(func() string { return "foo" })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "foo"}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARSE(t *testing.T) {
	uf := std.FuncARSE(func() (string, error) { return "foo", nil })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "foo"}, ret)
	uf = std.FuncARSE(func() (string, error) {
		return "", errors.New("some error")
	})
	ret, err = funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Error{
		Value: &vm.String{Value: "some error"},
	}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARSs(t *testing.T) {
	uf := std.FuncARSs(func() []string { return []string{"foo", "bar"} })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, array(&vm.String{Value: "foo"},
		&vm.String{Value: "bar"}), ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASRE(t *testing.T) {
	uf := std.FuncASRE(func(a string) error { return nil })
	ret, err := funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	uf = std.FuncASRE(func(a string) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &vm.Error{
		Value: &vm.String{Value: "some error"},
	}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASRS(t *testing.T) {
	uf := std.FuncASRS(func(a string) string { return a })
	ret, err := funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "foo"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASRSs(t *testing.T) {
	uf := std.FuncASRSs(func(a string) []string { return []string{a} })
	ret, err := funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, array(&vm.String{Value: "foo"}), ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASI64RE(t *testing.T) {
	uf := std.FuncASI64RE(func(a string, b int64) error { return nil })
	ret, err := funcCall(uf, &vm.String{Value: "foo"}, &vm.Int{Value: 5})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	uf = std.FuncASI64RE(func(a string, b int64) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.String{Value: "foo"}, &vm.Int{Value: 5})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAIIRE(t *testing.T) {
	uf := std.FuncAIIRE(func(a, b int) error { return nil })
	ret, err := funcCall(uf, &vm.Int{Value: 5}, &vm.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	uf = std.FuncAIIRE(func(a, b int) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.Int{Value: 5}, &vm.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASIIRE(t *testing.T) {
	uf := std.FuncASIIRE(func(a string, b, c int) error { return nil })
	ret, err := funcCall(uf, &vm.String{Value: "foo"}, &vm.Int{Value: 5},
		&vm.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	uf = std.FuncASIIRE(func(a string, b, c int) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.String{Value: "foo"}, &vm.Int{Value: 5},
		&vm.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASRSE(t *testing.T) {
	uf := std.FuncASRSE(func(a string) (string, error) { return a, nil })
	ret, err := funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "foo"}, ret)
	uf = std.FuncASRSE(func(a string) (string, error) {
		return a, errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASSRE(t *testing.T) {
	uf := std.FuncASSRE(func(a, b string) error { return nil })
	ret, err := funcCall(uf, &vm.String{Value: "foo"},
		&vm.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	uf = std.FuncASSRE(func(a, b string) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.String{Value: "foo"},
		&vm.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf, &vm.String{Value: "foo"})
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASsRS(t *testing.T) {
	uf := std.FuncASsSRS(func(a []string, b string) string {
		return strings.Join(a, b)
	})
	ret, err := funcCall(uf, array(&vm.String{Value: "foo"},
		&vm.String{Value: "bar"}), &vm.String{Value: " "})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "foo bar"}, ret)
	_, err = funcCall(uf, &vm.String{Value: "foo"})
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARF(t *testing.T) {
	uf := std.FuncARF(func() float64 { return 10.0 })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Float{Value: 10.0}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAFRF(t *testing.T) {
	uf := std.FuncAFRF(func(a float64) float64 { return a })
	ret, err := funcCall(uf, &vm.Float{Value: 10.0})
	require.NoError(t, err)
	require.Equal(t, &vm.Float{Value: 10.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAIRF(t *testing.T) {
	uf := std.FuncAIRF(func(a int) float64 {
		return float64(a)
	})
	ret, err := funcCall(uf, &vm.Int{Value: 10.0})
	require.NoError(t, err)
	require.Equal(t, &vm.Float{Value: 10.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAFRI(t *testing.T) {
	uf := std.FuncAFRI(func(a float64) int {
		return int(a)
	})
	ret, err := funcCall(uf, &vm.Float{Value: 10.5})
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 10}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAFRB(t *testing.T) {
	uf := std.FuncAFRB(func(a float64) bool {
		return a > 0.0
	})
	ret, err := funcCall(uf, &vm.Float{Value: 0.1})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAFFRF(t *testing.T) {
	uf := std.FuncAFFRF(func(a, b float64) float64 {
		return a + b
	})
	ret, err := funcCall(uf, &vm.Float{Value: 10.0},
		&vm.Float{Value: 20.0})
	require.NoError(t, err)
	require.Equal(t, &vm.Float{Value: 30.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASIRS(t *testing.T) {
	uf := std.FuncASIRS(func(a string, b int) string {
		return strings.Repeat(a, b)
	})
	ret, err := funcCall(uf, &vm.String{Value: "ab"}, &vm.Int{Value: 2})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "abab"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAIFRF(t *testing.T) {
	uf := std.FuncAIFRF(func(a int, b float64) float64 {
		return float64(a) + b
	})
	ret, err := funcCall(uf, &vm.Int{Value: 10}, &vm.Float{Value: 20.0})
	require.NoError(t, err)
	require.Equal(t, &vm.Float{Value: 30.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAFIRF(t *testing.T) {
	uf := std.FuncAFIRF(func(a float64, b int) float64 {
		return a + float64(b)
	})
	ret, err := funcCall(uf, &vm.Float{Value: 10.0}, &vm.Int{Value: 20})
	require.NoError(t, err)
	require.Equal(t, &vm.Float{Value: 30.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAFIRB(t *testing.T) {
	uf := std.FuncAFIRB(func(a float64, b int) bool {
		return a < float64(b)
	})
	ret, err := funcCall(uf, &vm.Float{Value: 10.0}, &vm.Int{Value: 20})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAIRSsE(t *testing.T) {
	uf := std.FuncAIRSsE(func(a int) ([]string, error) {
		return []string{"foo", "bar"}, nil
	})
	ret, err := funcCall(uf, &vm.Int{Value: 10})
	require.NoError(t, err)
	require.Equal(t, array(&vm.String{Value: "foo"},
		&vm.String{Value: "bar"}), ret)
	uf = std.FuncAIRSsE(func(a int) ([]string, error) {
		return nil, errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.Int{Value: 10})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASSRSs(t *testing.T) {
	uf := std.FuncASSRSs(func(a, b string) []string {
		return []string{a, b}
	})
	ret, err := funcCall(uf, &vm.String{Value: "foo"},
		&vm.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, array(&vm.String{Value: "foo"},
		&vm.String{Value: "bar"}), ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASSIRSs(t *testing.T) {
	uf := std.FuncASSIRSs(func(a, b string, c int) []string {
		return []string{a, b, strconv.Itoa(c)}
	})
	ret, err := funcCall(uf, &vm.String{Value: "foo"},
		&vm.String{Value: "bar"}, &vm.Int{Value: 5})
	require.NoError(t, err)
	require.Equal(t, array(&vm.String{Value: "foo"},
		&vm.String{Value: "bar"}, &vm.String{Value: "5"}), ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARB(t *testing.T) {
	uf := std.FuncARB(func() bool { return true })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARYE(t *testing.T) {
	uf := std.FuncARYE(func() ([]byte, error) {
		return []byte("foo bar"), nil
	})
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Bytes{Value: []byte("foo bar")}, ret)
	uf = std.FuncARYE(func() ([]byte, error) {
		return nil, errors.New("some error")
	})
	ret, err = funcCall(uf)
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf, vm.TrueValue)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASRIE(t *testing.T) {
	uf := std.FuncASRIE(func(a string) (int, error) { return 5, nil })
	ret, err := funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 5}, ret)
	uf = std.FuncASRIE(func(a string) (int, error) {
		return 0, errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAYRIE(t *testing.T) {
	uf := std.FuncAYRIE(func(a []byte) (int, error) { return 5, nil })
	ret, err := funcCall(uf, &vm.Bytes{Value: []byte("foo")})
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 5}, ret)
	uf = std.FuncAYRIE(func(a []byte) (int, error) {
		return 0, errors.New("some error")
	})
	ret, err = funcCall(uf, &vm.Bytes{Value: []byte("foo")})
	require.NoError(t, err)
	require.Equal(t,
		&vm.Error{Value: &vm.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASSRI(t *testing.T) {
	uf := std.FuncASSRI(func(a, b string) int { return len(a) + len(b) })
	ret, err := funcCall(uf,
		&vm.String{Value: "foo"}, &vm.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 6}, ret)
	_, err = funcCall(uf, &vm.String{Value: "foo"})
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASSRS(t *testing.T) {
	uf := std.FuncASSRS(func(a, b string) string { return a + b })
	ret, err := funcCall(uf,
		&vm.String{Value: "foo"}, &vm.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "foobar"}, ret)
	_, err = funcCall(uf, &vm.String{Value: "foo"})
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASSRB(t *testing.T) {
	uf := std.FuncASSRB(func(a, b string) bool { return len(a) > len(b) })
	ret, err := funcCall(uf,
		&vm.String{Value: "123"}, &vm.String{Value: "12"})
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, ret)
	_, err = funcCall(uf, &vm.String{Value: "foo"})
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAIRS(t *testing.T) {
	uf := std.FuncAIRS(func(a int) string { return strconv.Itoa(a) })
	ret, err := funcCall(uf, &vm.Int{Value: 55})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "55"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAIRIs(t *testing.T) {
	uf := std.FuncAIRIs(func(a int) []int { return []int{a, a} })
	ret, err := funcCall(uf, &vm.Int{Value: 55})
	require.NoError(t, err)
	require.Equal(t, array(&vm.Int{Value: 55}, &vm.Int{Value: 55}), ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAI64R(t *testing.T) {
	uf := std.FuncAIR(func(a int) {})
	ret, err := funcCall(uf, &vm.Int{Value: 55})
	require.NoError(t, err)
	require.Equal(t, vm.UndefinedValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncARI64(t *testing.T) {
	uf := std.FuncARI64(func() int64 { return 55 })
	ret, err := funcCall(uf)
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 55}, ret)
	_, err = funcCall(uf, &vm.Int{Value: 55})
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncASsSRS(t *testing.T) {
	uf := std.FuncASsSRS(func(a []string, b string) string {
		return strings.Join(a, b)
	})
	ret, err := funcCall(uf,
		array(&vm.String{Value: "abc"}, &vm.String{Value: "def"}),
		&vm.String{Value: "-"})
	require.NoError(t, err)
	require.Equal(t, &vm.String{Value: "abc-def"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func TestFuncAI64RI64(t *testing.T) {
	uf := std.FuncAI64RI64(func(a int64) int64 { return a * 2 })
	ret, err := funcCall(uf, &vm.Int{Value: 55})
	require.NoError(t, err)
	require.Equal(t, &vm.Int{Value: 110}, ret)
	_, err = funcCall(uf)
	require.Equal(t, vm.ErrWrongNumArguments, err)
}

func funcCall(
	fn vm.CallableFunc,
	args ...vm.Object,
) (vm.Object, error) {
	userFunc := &vm.BuiltinFunction{Value: fn}
	return userFunc.Call(context.Background(), args...)
}

func array(elements ...vm.Object) *vm.Array {
	return &vm.Array{Value: elements}
}
