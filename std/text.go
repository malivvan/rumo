package std

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/malivvan/rumo/vm"
)

var textModule = map[string]vm.Object{
	"re_match": &vm.BuiltinFunction{
		Name:  "re_match",
		Value: textREMatch,
	}, // re_match(pattern, text) => bool/error
	"re_find": &vm.BuiltinFunction{
		Name:  "re_find",
		Value: textREFind,
	}, // re_find(pattern, text, count) => [[{text:,begin:,end:}]]/undefined
	"re_replace": &vm.BuiltinFunction{
		Name:  "re_replace",
		Value: textREReplace,
	}, // re_replace(pattern, text, repl) => string/error
	"re_split": &vm.BuiltinFunction{
		Name:  "re_split",
		Value: textRESplit,
	}, // re_split(pattern, text, count) => [string]/error
	"re_compile": &vm.BuiltinFunction{
		Name:  "re_compile",
		Value: textRECompile,
	}, // re_compile(pattern) => Regexp/error
	"compare": &vm.BuiltinFunction{
		Name:  "compare",
		Value: FuncASSRI(strings.Compare),
	}, // compare(a, b) => int
	"contains": &vm.BuiltinFunction{
		Name:  "contains",
		Value: FuncASSRB(strings.Contains),
	}, // contains(s, substr) => bool
	"contains_any": &vm.BuiltinFunction{
		Name:  "contains_any",
		Value: FuncASSRB(strings.ContainsAny),
	}, // contains_any(s, chars) => bool
	"count": &vm.BuiltinFunction{
		Name:  "count",
		Value: FuncASSRI(strings.Count),
	}, // count(s, substr) => int
	"equal_fold": &vm.BuiltinFunction{
		Name:  "equal_fold",
		Value: FuncASSRB(strings.EqualFold),
	}, // "equal_fold(s, t) => bool
	"fields": &vm.BuiltinFunction{
		Name:  "fields",
		Value: FuncASRSs(strings.Fields),
	}, // fields(s) => [string]
	"has_prefix": &vm.BuiltinFunction{
		Name:  "has_prefix",
		Value: FuncASSRB(strings.HasPrefix),
	}, // has_prefix(s, prefix) => bool
	"has_suffix": &vm.BuiltinFunction{
		Name:  "has_suffix",
		Value: FuncASSRB(strings.HasSuffix),
	}, // has_suffix(s, suffix) => bool
	"index": &vm.BuiltinFunction{
		Name:  "index",
		Value: FuncASSRI(strings.Index),
	}, // index(s, substr) => int
	"index_any": &vm.BuiltinFunction{
		Name:  "index_any",
		Value: FuncASSRI(strings.IndexAny),
	}, // index_any(s, chars) => int
	"join": &vm.BuiltinFunction{
		Name:  "join",
		Value: textJoin,
	}, // join(arr, sep) => string
	"last_index": &vm.BuiltinFunction{
		Name:  "last_index",
		Value: FuncASSRI(strings.LastIndex),
	}, // last_index(s, substr) => int
	"last_index_any": &vm.BuiltinFunction{
		Name:  "last_index_any",
		Value: FuncASSRI(strings.LastIndexAny),
	}, // last_index_any(s, chars) => int
	"repeat": &vm.BuiltinFunction{
		Name:  "repeat",
		Value: textRepeat,
	}, // repeat(s, count) => string
	"replace": &vm.BuiltinFunction{
		Name:  "replace",
		Value: textReplace,
	}, // replace(s, old, new, n) => string
	"substr": &vm.BuiltinFunction{
		Name:  "substr",
		Value: textSubstring,
	}, // substr(s, lower, upper) => string
	"split": &vm.BuiltinFunction{
		Name:  "split",
		Value: FuncASSRSs(strings.Split),
	}, // split(s, sep) => [string]
	"split_after": &vm.BuiltinFunction{
		Name:  "split_after",
		Value: FuncASSRSs(strings.SplitAfter),
	}, // split_after(s, sep) => [string]
	"split_after_n": &vm.BuiltinFunction{
		Name:  "split_after_n",
		Value: FuncASSIRSs(strings.SplitAfterN),
	}, // split_after_n(s, sep, n) => [string]
	"split_n": &vm.BuiltinFunction{
		Name:  "split_n",
		Value: FuncASSIRSs(strings.SplitN),
	}, // split_n(s, sep, n) => [string]
	"title": &vm.BuiltinFunction{
		Name:  "title",
		Value: FuncASRS(strings.Title),
	}, // title(s) => string
	"to_lower": &vm.BuiltinFunction{
		Name:  "to_lower",
		Value: FuncASRS(strings.ToLower),
	}, // to_lower(s) => string
	"to_title": &vm.BuiltinFunction{
		Name:  "to_title",
		Value: FuncASRS(strings.ToTitle),
	}, // to_title(s) => string
	"to_upper": &vm.BuiltinFunction{
		Name:  "to_upper",
		Value: FuncASRS(strings.ToUpper),
	}, // to_upper(s) => string
	"pad_left": &vm.BuiltinFunction{
		Name:  "pad_left",
		Value: textPadLeft,
	}, // pad_left(s, pad_len, pad_with) => string
	"pad_right": &vm.BuiltinFunction{
		Name:  "pad_right",
		Value: textPadRight,
	}, // pad_right(s, pad_len, pad_with) => string
	"trim": &vm.BuiltinFunction{
		Name:  "trim",
		Value: FuncASSRS(strings.Trim),
	}, // trim(s, cutset) => string
	"trim_left": &vm.BuiltinFunction{
		Name:  "trim_left",
		Value: FuncASSRS(strings.TrimLeft),
	}, // trim_left(s, cutset) => string
	"trim_prefix": &vm.BuiltinFunction{
		Name:  "trim_prefix",
		Value: FuncASSRS(strings.TrimPrefix),
	}, // trim_prefix(s, prefix) => string
	"trim_right": &vm.BuiltinFunction{
		Name:  "trim_right",
		Value: FuncASSRS(strings.TrimRight),
	}, // trim_right(s, cutset) => string
	"trim_space": &vm.BuiltinFunction{
		Name:  "trim_space",
		Value: FuncASRS(strings.TrimSpace),
	}, // trim_space(s) => string
	"trim_suffix": &vm.BuiltinFunction{
		Name:  "trim_suffix",
		Value: FuncASSRS(strings.TrimSuffix),
	}, // trim_suffix(s, suffix) => string
	"atoi": &vm.BuiltinFunction{
		Name:  "atoi",
		Value: FuncASRIE(strconv.Atoi),
	}, // atoi(str) => int/error
	"format_bool": &vm.BuiltinFunction{
		Name:  "format_bool",
		Value: textFormatBool,
	}, // format_bool(b) => string
	"format_float": &vm.BuiltinFunction{
		Name:  "format_float",
		Value: textFormatFloat,
	}, // format_float(f, fmt, prec, bits) => string
	"format_int": &vm.BuiltinFunction{
		Name:  "format_int",
		Value: textFormatInt,
	}, // format_int(i, base) => string
	"itoa": &vm.BuiltinFunction{
		Name:  "itoa",
		Value: FuncAIRS(strconv.Itoa),
	}, // itoa(i) => string
	"parse_bool": &vm.BuiltinFunction{
		Name:  "parse_bool",
		Value: textParseBool,
	}, // parse_bool(str) => bool/error
	"parse_float": &vm.BuiltinFunction{
		Name:  "parse_float",
		Value: textParseFloat,
	}, // parse_float(str, bits) => float/error
	"parse_int": &vm.BuiltinFunction{
		Name:  "parse_int",
		Value: textParseInt,
	}, // parse_int(str, base, bits) => int/error
	"quote": &vm.BuiltinFunction{
		Name:  "quote",
		Value: FuncASRS(strconv.Quote),
	}, // quote(str) => string
	"unquote": &vm.BuiltinFunction{
		Name:  "unquote",
		Value: FuncASRSE(strconv.Unquote),
	}, // unquote(str) => string/error
}

func textREMatch(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	matched, err := regexp.MatchString(s1, s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	if matched {
		ret = vm.TrueValue
	} else {
		ret = vm.FalseValue
	}

	return
}

func textREFind(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if numArgs < 3 {
		m := re.FindStringSubmatchIndex(s2)
		if m == nil {
			ret = vm.UndefinedValue
			return
		}

		arr := &vm.Array{}
		for i := 0; i < len(m); i += 2 {
			arr.Value = append(arr.Value,
				&vm.ImmutableMap{Value: map[string]vm.Object{
					"text":  &vm.String{Value: s2[m[i]:m[i+1]]},
					"begin": &vm.Int{Value: int64(m[i])},
					"end":   &vm.Int{Value: int64(m[i+1])},
				}})
		}

		ret = &vm.Array{Value: []vm.Object{arr}}

		return
	}

	i3, ok := vm.ToInt(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}
	m := re.FindAllStringSubmatchIndex(s2, i3)
	if m == nil {
		ret = vm.UndefinedValue
		return
	}

	arr := &vm.Array{}
	for _, m := range m {
		subMatch := &vm.Array{}
		for i := 0; i < len(m); i += 2 {
			subMatch.Value = append(subMatch.Value,
				&vm.ImmutableMap{Value: map[string]vm.Object{
					"text":  &vm.String{Value: s2[m[i]:m[i+1]]},
					"begin": &vm.Int{Value: int64(m[i])},
					"end":   &vm.Int{Value: int64(m[i+1])},
				}})
		}

		arr.Value = append(arr.Value, subMatch)
	}

	ret = arr

	return
}

func textREReplace(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	s3, ok := vm.ToString(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
	} else {
		s, ok := doTextRegexpReplace(re, s2, s3)
		if !ok {
			return nil, vm.ErrStringLimit
		}

		ret = &vm.String{Value: s}
	}

	return
}

func textRESplit(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	var i3 = -1
	if numArgs > 2 {
		i3, ok = vm.ToInt(args[2])
		if !ok {
			err = vm.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	arr := &vm.Array{}
	for _, s := range re.Split(s2, i3) {
		arr.Value = append(arr.Value, &vm.String{Value: s})
	}

	ret = arr

	return
}

func textRECompile(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
	} else {
		ret = makeTextRegexp(re)
	}

	return
}

func textReplace(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 4 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	s3, ok := vm.ToString(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	i4, ok := vm.ToInt(args[3])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	s, ok := doTextReplace(s1, s2, s3, i4)
	if !ok {
		err = vm.ErrStringLimit
		return
	}

	ret = &vm.String{Value: s}

	return
}

func textSubstring(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	strlen := len(s1)
	i3 := strlen
	if argslen == 3 {
		i3, ok = vm.ToInt(args[2])
		if !ok {
			err = vm.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	if i2 > i3 {
		err = vm.ErrInvalidIndexType
		return
	}

	if i2 < 0 {
		i2 = 0
	} else if i2 > strlen {
		i2 = strlen
	}

	if i3 < 0 {
		i3 = 0
	} else if i3 > strlen {
		i3 = strlen
	}

	ret = &vm.String{Value: s1[i2:i3]}

	return
}

func textPadLeft(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if i2 > vm.MaxStringLen {
		return nil, vm.ErrStringLimit
	}

	sLen := len(s1)
	if sLen >= i2 {
		ret = &vm.String{Value: s1}
		return
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = vm.ToString(args[2])
		if !ok {
			err = vm.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "string(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		ret = &vm.String{Value: s1}
		return
	}

	padCount := ((i2 - padStrLen) / padStrLen) + 1
	retStr := strings.Repeat(s3, padCount) + s1
	ret = &vm.String{Value: retStr[len(retStr)-i2:]}

	return
}

func textPadRight(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if i2 > vm.MaxStringLen {
		return nil, vm.ErrStringLimit
	}

	sLen := len(s1)
	if sLen >= i2 {
		ret = &vm.String{Value: s1}
		return
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = vm.ToString(args[2])
		if !ok {
			err = vm.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "string(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		ret = &vm.String{Value: s1}
		return
	}

	padCount := ((i2 - padStrLen) / padStrLen) + 1
	retStr := s1 + strings.Repeat(s3, padCount)
	ret = &vm.String{Value: retStr[:i2]}

	return
}

func textRepeat(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}

	if len(s1)*i2 > vm.MaxStringLen {
		return nil, vm.ErrStringLimit
	}

	return &vm.String{Value: strings.Repeat(s1, i2)}, nil
}

func textJoin(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}

	var slen int
	var ss1 []string
	switch arg0 := args[0].(type) {
	case *vm.Array:
		for idx, a := range arg0.Value {
			as, ok := vm.ToString(a)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{
					Name:     fmt.Sprintf("first[%d]", idx),
					Expected: "string(compatible)",
					Found:    a.TypeName(),
				}
			}
			slen += len(as)
			ss1 = append(ss1, as)
		}
	case *vm.ImmutableArray:
		for idx, a := range arg0.Value {
			as, ok := vm.ToString(a)
			if !ok {
				return nil, vm.ErrInvalidArgumentType{
					Name:     fmt.Sprintf("first[%d]", idx),
					Expected: "string(compatible)",
					Found:    a.TypeName(),
				}
			}
			slen += len(as)
			ss1 = append(ss1, as)
		}
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "array",
			Found:    args[0].TypeName(),
		}
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}

	// make sure output length does not exceed the limit
	if slen+len(s2)*(len(ss1)-1) > vm.MaxStringLen {
		return nil, vm.ErrStringLimit
	}

	return &vm.String{Value: strings.Join(ss1, s2)}, nil
}

func textFormatBool(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	b1, ok := args[0].(*vm.Bool)
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bool",
			Found:    args[0].TypeName(),
		}
		return
	}

	if b1 == vm.TrueValue {
		ret = &vm.String{Value: "true"}
	} else {
		ret = &vm.String{Value: "false"}
	}

	return
}

func textFormatFloat(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 4 {
		err = vm.ErrWrongNumArguments
		return
	}

	f1, ok := args[0].(*vm.Float)
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	i3, ok := vm.ToInt(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	i4, ok := vm.ToInt(args[3])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	ret = &vm.String{Value: strconv.FormatFloat(f1.Value, s2[0], i3, i4)}

	return
}

func textFormatInt(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].(*vm.Int)
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &vm.String{Value: strconv.FormatInt(i1.Value, i2)}

	return
}

func textParseBool(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*vm.String)
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
		return
	}

	parsed, err := strconv.ParseBool(s1.Value)
	if err != nil {
		ret = wrapError(err)
		return
	}

	if parsed {
		ret = vm.TrueValue
	} else {
		ret = vm.FalseValue
	}

	return
}

func textParseFloat(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*vm.String)
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	parsed, err := strconv.ParseFloat(s1.Value, i2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &vm.Float{Value: parsed}

	return
}

func textParseInt(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 3 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*vm.String)
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	i3, ok := vm.ToInt(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	parsed, err := strconv.ParseInt(s1.Value, i2, i3)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &vm.Int{Value: parsed}

	return
}

// Modified implementation of strings.Replace
// to limit the maximum length of output string.
func doTextReplace(s, old, new string, n int) (string, bool) {
	if old == new || n == 0 {
		return s, true // avoid allocation
	}

	// Compute number of replacements.
	if m := strings.Count(s, old); m == 0 {
		return s, true // avoid allocation
	} else if n < 0 || m < n {
		n = m
	}

	// Apply replacements to buffer.
	t := make([]byte, len(s)+n*(len(new)-len(old)))
	w := 0
	start := 0
	for i := 0; i < n; i++ {
		j := start
		if len(old) == 0 {
			if i > 0 {
				_, wid := utf8.DecodeRuneInString(s[start:])
				j += wid
			}
		} else {
			j += strings.Index(s[start:], old)
		}

		ssj := s[start:j]
		if w+len(ssj)+len(new) > vm.MaxStringLen {
			return "", false
		}

		w += copy(t[w:], ssj)
		w += copy(t[w:], new)
		start = j + len(old)
	}

	ss := s[start:]
	if w+len(ss) > vm.MaxStringLen {
		return "", false
	}

	w += copy(t[w:], ss)

	return string(t[0:w]), true
}
