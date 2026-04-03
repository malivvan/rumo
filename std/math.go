package std

import (
	"math"

	"github.com/malivvan/vv/vm"
)

var mathModule = map[string]vm.Object{
	"e":       &vm.Float{Value: math.E},
	"pi":      &vm.Float{Value: math.Pi},
	"phi":     &vm.Float{Value: math.Phi},
	"sqrt2":   &vm.Float{Value: math.Sqrt2},
	"sqrtE":   &vm.Float{Value: math.SqrtE},
	"sqrtPi":  &vm.Float{Value: math.SqrtPi},
	"sqrtPhi": &vm.Float{Value: math.SqrtPhi},
	"ln2":     &vm.Float{Value: math.Ln2},
	"log2E":   &vm.Float{Value: math.Log2E},
	"ln10":    &vm.Float{Value: math.Ln10},
	"log10E":  &vm.Float{Value: math.Log10E},
	"abs": &vm.BuiltinFunction{
		Name:  "abs",
		Value: FuncAFRF(math.Abs),
	},
	"acos": &vm.BuiltinFunction{
		Name:  "acos",
		Value: FuncAFRF(math.Acos),
	},
	"acosh": &vm.BuiltinFunction{
		Name:  "acosh",
		Value: FuncAFRF(math.Acosh),
	},
	"asin": &vm.BuiltinFunction{
		Name:  "asin",
		Value: FuncAFRF(math.Asin),
	},
	"asinh": &vm.BuiltinFunction{
		Name:  "asinh",
		Value: FuncAFRF(math.Asinh),
	},
	"atan": &vm.BuiltinFunction{
		Name:  "atan",
		Value: FuncAFRF(math.Atan),
	},
	"atan2": &vm.BuiltinFunction{
		Name:  "atan2",
		Value: FuncAFFRF(math.Atan2),
	},
	"atanh": &vm.BuiltinFunction{
		Name:  "atanh",
		Value: FuncAFRF(math.Atanh),
	},
	"cbrt": &vm.BuiltinFunction{
		Name:  "cbrt",
		Value: FuncAFRF(math.Cbrt),
	},
	"ceil": &vm.BuiltinFunction{
		Name:  "ceil",
		Value: FuncAFRF(math.Ceil),
	},
	"copysign": &vm.BuiltinFunction{
		Name:  "copysign",
		Value: FuncAFFRF(math.Copysign),
	},
	"cos": &vm.BuiltinFunction{
		Name:  "cos",
		Value: FuncAFRF(math.Cos),
	},
	"cosh": &vm.BuiltinFunction{
		Name:  "cosh",
		Value: FuncAFRF(math.Cosh),
	},
	"dim": &vm.BuiltinFunction{
		Name:  "dim",
		Value: FuncAFFRF(math.Dim),
	},
	"erf": &vm.BuiltinFunction{
		Name:  "erf",
		Value: FuncAFRF(math.Erf),
	},
	"erfc": &vm.BuiltinFunction{
		Name:  "erfc",
		Value: FuncAFRF(math.Erfc),
	},
	"exp": &vm.BuiltinFunction{
		Name:  "exp",
		Value: FuncAFRF(math.Exp),
	},
	"exp2": &vm.BuiltinFunction{
		Name:  "exp2",
		Value: FuncAFRF(math.Exp2),
	},
	"expm1": &vm.BuiltinFunction{
		Name:  "expm1",
		Value: FuncAFRF(math.Expm1),
	},
	"floor": &vm.BuiltinFunction{
		Name:  "floor",
		Value: FuncAFRF(math.Floor),
	},
	"gamma": &vm.BuiltinFunction{
		Name:  "gamma",
		Value: FuncAFRF(math.Gamma),
	},
	"hypot": &vm.BuiltinFunction{
		Name:  "hypot",
		Value: FuncAFFRF(math.Hypot),
	},
	"ilogb": &vm.BuiltinFunction{
		Name:  "ilogb",
		Value: FuncAFRI(math.Ilogb),
	},
	"inf": &vm.BuiltinFunction{
		Name:  "inf",
		Value: FuncAIRF(math.Inf),
	},
	"is_inf": &vm.BuiltinFunction{
		Name:  "is_inf",
		Value: FuncAFIRB(math.IsInf),
	},
	"is_nan": &vm.BuiltinFunction{
		Name:  "is_nan",
		Value: FuncAFRB(math.IsNaN),
	},
	"j0": &vm.BuiltinFunction{
		Name:  "j0",
		Value: FuncAFRF(math.J0),
	},
	"j1": &vm.BuiltinFunction{
		Name:  "j1",
		Value: FuncAFRF(math.J1),
	},
	"jn": &vm.BuiltinFunction{
		Name:  "jn",
		Value: FuncAIFRF(math.Jn),
	},
	"ldexp": &vm.BuiltinFunction{
		Name:  "ldexp",
		Value: FuncAFIRF(math.Ldexp),
	},
	"log": &vm.BuiltinFunction{
		Name:  "log",
		Value: FuncAFRF(math.Log),
	},
	"log10": &vm.BuiltinFunction{
		Name:  "log10",
		Value: FuncAFRF(math.Log10),
	},
	"log1p": &vm.BuiltinFunction{
		Name:  "log1p",
		Value: FuncAFRF(math.Log1p),
	},
	"log2": &vm.BuiltinFunction{
		Name:  "log2",
		Value: FuncAFRF(math.Log2),
	},
	"logb": &vm.BuiltinFunction{
		Name:  "logb",
		Value: FuncAFRF(math.Logb),
	},
	"max": &vm.BuiltinFunction{
		Name:  "max",
		Value: FuncAFFRF(math.Max),
	},
	"min": &vm.BuiltinFunction{
		Name:  "min",
		Value: FuncAFFRF(math.Min),
	},
	"mod": &vm.BuiltinFunction{
		Name:  "mod",
		Value: FuncAFFRF(math.Mod),
	},
	"nan": &vm.BuiltinFunction{
		Name:  "nan",
		Value: FuncARF(math.NaN),
	},
	"nextafter": &vm.BuiltinFunction{
		Name:  "nextafter",
		Value: FuncAFFRF(math.Nextafter),
	},
	"pow": &vm.BuiltinFunction{
		Name:  "pow",
		Value: FuncAFFRF(math.Pow),
	},
	"pow10": &vm.BuiltinFunction{
		Name:  "pow10",
		Value: FuncAIRF(math.Pow10),
	},
	"remainder": &vm.BuiltinFunction{
		Name:  "remainder",
		Value: FuncAFFRF(math.Remainder),
	},
	"signbit": &vm.BuiltinFunction{
		Name:  "signbit",
		Value: FuncAFRB(math.Signbit),
	},
	"sin": &vm.BuiltinFunction{
		Name:  "sin",
		Value: FuncAFRF(math.Sin),
	},
	"sinh": &vm.BuiltinFunction{
		Name:  "sinh",
		Value: FuncAFRF(math.Sinh),
	},
	"sqrt": &vm.BuiltinFunction{
		Name:  "sqrt",
		Value: FuncAFRF(math.Sqrt),
	},
	"tan": &vm.BuiltinFunction{
		Name:  "tan",
		Value: FuncAFRF(math.Tan),
	},
	"tanh": &vm.BuiltinFunction{
		Name:  "tanh",
		Value: FuncAFRF(math.Tanh),
	},
	"trunc": &vm.BuiltinFunction{
		Name:  "trunc",
		Value: FuncAFRF(math.Trunc),
	},
	"y0": &vm.BuiltinFunction{
		Name:  "y0",
		Value: FuncAFRF(math.Y0),
	},
	"y1": &vm.BuiltinFunction{
		Name:  "y1",
		Value: FuncAFRF(math.Y1),
	},
	"yn": &vm.BuiltinFunction{
		Name:  "yn",
		Value: FuncAIFRF(math.Yn),
	},
}
