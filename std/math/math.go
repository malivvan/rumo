package math

import (
	"math"

	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin("math").
	Const("e float", math.E).
	Const("pi float", math.Pi).
	Const("phi float", math.Phi).
	Const("sqrt2 float", math.Sqrt2).
	Const("sqrtE float", math.SqrtE).
	Const("sqrtPi float", math.SqrtPi).
	Const("sqrtPhi float", math.SqrtPhi).
	Const("ln2 float", math.Ln2).
	Const("log2E float", math.Log2E).
	Const("ln10 float", math.Ln10).
	Const("log10E float", math.Log10E).
	Func("abs(x float) (r float)", math.Abs).
	Func("acos(x float) (r float)", math.Acos).
	Func("acosh(x float) (r float)", math.Acosh).
	Func("asin(x float) (r float)", math.Asin).
	Func("asinh(x float) (r float)", math.Asinh).
	Func("atan(x float) (r float)", math.Atan).
	Func("atan2(y, x float) (r float)", math.Atan2).
	Func("atanh(x float) (r float)", math.Atanh).
	Func("cbrt(x float) (r float)", math.Cbrt).
	Func("ceil(x float) (r float)", math.Ceil).
	Func("copysign(x, y float) (r float)", math.Copysign).
	Func("cos(x float) (r float)", math.Cos).
	Func("cosh(x float) (r float)", math.Cosh).
	Func("dim(x, y float) (r float)", math.Dim).
	Func("erf(x float) (r float)", math.Erf).
	Func("erfc(x float) (r float)", math.Erfc).
	Func("exp(x float) (r float)", math.Exp).
	Func("exp2(x float) (r float)", math.Exp2).
	Func("expm1(x float) (r float)", math.Expm1).
	Func("floor(x float) (r float)", math.Floor).
	Func("gamma(x float) (r float)", math.Gamma).
	Func("hypot(x, y float) (r float)", math.Hypot).
	Func("ilogb(x float) (r int)", math.Ilogb).
	Func("inf(sign int) (r float)", math.Inf).
	Func("is_inf(x float, sign int) (r bool)", math.IsInf).
	Func("is_nan(x float) (r bool)", math.IsNaN).
	Func("j0(x float) (r float)", math.J0).
	Func("j1(x float) (r float)", math.J1).
	Func("jn(n int, x float) (r float)", math.Jn).
	Func("ldexp(frac float, exp int) (r float)", math.Ldexp).
	Func("log(x float) (r float)", math.Log).
	Func("log10(x float) (r float)", math.Log10).
	Func("log1p(x float) (r float)", math.Log1p).
	Func("log2(x float) (r float)", math.Log2).
	Func("logb(x float) (r float)", math.Logb).
	Func("max(x, y float) (r float)", math.Max).
	Func("min(x, y float) (r float)", math.Min).
	Func("mod(x, y float) (r float)", math.Mod).
	Func("nan() (r float)", math.NaN).
	Func("nextafter(x, y float) (r float)", math.Nextafter).
	Func("pow(x, y float) (r float)", math.Pow).
	Func("pow10(n int) (r float)", math.Pow10).
	Func("remainder(x, y float) (r float)", math.Remainder).
	Func("signbit(x float) (r bool)", math.Signbit).
	Func("sin(x float) (r float)", math.Sin).
	Func("sinh(x float) (r float)", math.Sinh).
	Func("sqrt(x float) (r float)", math.Sqrt).
	Func("tan(x float) (r float)", math.Tan).
	Func("tanh(x float) (r float)", math.Tanh).
	Func("trunc(x float) (r float)", math.Trunc).
	Func("y0(x float) (r float)", math.Y0).
	Func("y1(x float) (r float)", math.Y1).
	Func("yn(n int, x float) (r float)", math.Yn)
