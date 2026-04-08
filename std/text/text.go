package text

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin("text").
	Func("re_match(pattern string, text string) (matched bool, err error)												returns whether the text matches the regular expression pattern", textREMatch).
	Func("re_find(pattern string, text string, count int) (matches [[{text string, begin int, end int}]], err error)	returns the matches of the regular expression pattern in the text. If count is not provided, it returns the first match.", textREFind).
	Func("re_replace(pattern string, text string, repl string) (result string, err error)								returns a copy of the text with all matches of the regular expression pattern replaced by the replacement string repl", textREReplace).
	Func("re_split(pattern string, text string, count int) (result [string], err error)									returns a slice of strings split by the regular expression pattern. If count is not provided, it splits all occurrences.", textRESplit).
	Func("re_compile(pattern string) (re Regexp, err error)																compiles the regular expression pattern and returns a Regexp object", textRECompile).
	Func("compare(a string, b string) (ret int)																			returns an integer comparing two strings lexicographically", strings.Compare).
	Func("contains(s string, substr string) (ret bool)																	returns true if substr is within s", strings.Contains).
	Func("contains_any(s string, chars string) (ret bool)																returns true if any Unicode code point in chars is within s", strings.ContainsAny).
	Func("count(s string, substr string) (ret int)																		returns the number of non-overlapping instances of substr in s", strings.Count).
	Func("equal_fold(s string, t string) (ret bool)																		returns true if s and t are equal under Unicode case-folding", strings.EqualFold).
	Func("fields(s string) (ret string)																					returns a slice of strings split from s by white space", strings.Fields).
	Func("has_prefix(s string, prefix string) (ret bool)																returns true if s begins with prefix", strings.HasPrefix).
	Func("has_suffix(s string, suffix string) (ret bool)																returns true if s ends with suffix", strings.HasSuffix).
	Func("index(s string, substr string) (ret int)																		returns the index of the first instance of substr in s, or -1 if substr is not present in s", strings.Index).
	Func("index_any(s string, chars string) (ret int)																	returns the index of the first instance of any Unicode code point in chars in s, or -1 if no Unicode code point in chars is present in s", strings.IndexAny).
	Func("join(arr [string], sep string) (ret string)																	returns the concatenation of the elements of arr separated by the separator sep", textJoin).
	Func("last_index(s string, substr string) (ret int)																	returns the index of the last instance of substr in s, or -1 if substr is not present in s", strings.LastIndex).
	Func("last_index_any(s string, chars string) (ret int)																returns the index of the last instance of any Unicode code point in chars in s, or -1 if no Unicode code point in chars is present in s", strings.LastIndexAny).
	Func("repeat(s string, count int) (ret string)																		returns a new string consisting of count copies of the string s", textRepeat).
	Func("replace(s string, old string, new string, n int) (ret string)													returns a copy of the string s with the first n non-overlapping instances of old replaced by new. If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string. If n < 0, there is no limit on the number of replacements.", textReplace).
	Func("substr(s string, lower int, upper int) (ret string)															returns the substring of s from index lower to upper. If upper is not provided, it returns the substring from lower to the end of s", textSubstring).
	Func("split(s string, sep string) (ret string)																		returns a slice of strings split from s by the separator sep", strings.Split).
	Func("split_after(s string, sep string) (ret string)																returns a slice of strings split from s by the separator sep, including the separator in the resulting strings", strings.SplitAfter).
	Func("split_after_n(s string, sep string, n int) (ret string)														returns a slice of strings split from s by the separator sep, including the separator in the resulting strings, with a maximum of n splits", strings.SplitAfterN).
	Func("split_n(s string, sep string, n int) (ret string)																returns a slice of strings split from s by the separator sep, with a maximum of n splits", strings.SplitN).
	Func("title(s string) (ret string)																					returns a copy of the string s with all Unicode letters that begin words mapped to their title case", strings.Title).
	Func("to_lower(s string) (ret string)																				returns a copy of the string s with all Unicode letters mapped to their lower case", strings.ToLower).
	Func("to_title(s string) (ret string)																				returns a copy of the string s with all Unicode letters mapped to their title case", strings.ToTitle).
	Func("to_upper(s string) (ret string)																				returns a copy of the string s with all Unicode letters mapped to their upper case", strings.ToUpper).
	Func("pad_left(s string, pad_len int, pad_with string) (ret string)													returns a copy of the string s left-padded with the pad_with string to a total length of pad_len. If pad_with is not provided, it defaults to a single space", textPadLeft).
	Func("pad_right(s string, pad_len int, pad_with string) (ret string)												returns a copy of the string s right-padded with the pad_with string to a total length of pad_len. If pad_with is not provided, it defaults to a single space", textPadRight).
	Func("trim(s string, cutset string) (ret string)																	returns a copy of the string s with all leading and trailing Unicode code points contained in cutset removed", strings.Trim).
	Func("trim_left(s string, cutset string) (ret string)																returns a copy of the string s with all leading Unicode code points contained in cutset removed", strings.TrimLeft).
	Func("trim_prefix(s string, prefix string) (ret string)																returns s without the provided leading prefix string. If s doesn't start with prefix, s is returned unchanged", strings.TrimPrefix).
	Func("trim_right(s string, cutset string) (ret string)																returns a copy of the string s with all trailing Unicode code points contained in cutset removed", strings.TrimRight).
	Func("trim_space(s string) (ret string)																				returns a copy of the string s with all leading and trailing white space removed, as defined by Unicode", strings.TrimSpace).
	Func("trim_suffix(s string, suffix string) (ret string)																returns s without the provided trailing suffix string. If s doesn't end with suffix, s is returned unchanged", strings.TrimSuffix).
	Func("atoi(str string) (i int, err error)																			returns the integer represented by the string str", strconv.Atoi).
	Func("format_bool(b bool) (ret string)																				returns the string representation of the boolean value b", textFormatBool).
	Func("format_float(f float, fmt byte, prec int, bits int) (ret string)												returns the string representation of the floating-point number f formatted according to the format fmt and precision prec", textFormatFloat).
	Func("format_int(i int, base int) (ret string)																		returns the string representation of the integer i in the specified base", textFormatInt).
	Func("itoa(i int) (ret string)																						returns the string representation of the integer i", strconv.Itoa).
	Func("parse_bool(str string) (b bool, err error)																	returns the boolean value represented by the string str", textParseBool).
	Func("parse_float(str string, bits int) (f float, err error)														returns the floating-point number represented by the string str", textParseFloat).
	Func("parse_int(str string, base int, bits int) (i int, err error)													returns the integer represented by the string str in the specified base and bit size", textParseInt).
	Func("quote(str string) (ret string)																				returns a double-quoted Go string literal representing str", strconv.Quote).
	Func("unquote(str string) (result string, err error)																returns the string represented by the Go string literal str", strconv.Unquote)

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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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
		ret = module.WrapError(err)
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

// regexp:

func makeTextRegexp(re *regexp.Regexp) *vm.ImmutableMap {
	return &vm.ImmutableMap{
		Value: map[string]vm.Object{
			// match(text) => bool
			"match": &vm.BuiltinFunction{
				Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
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

					if re.MatchString(s1) {
						ret = vm.TrueValue
					} else {
						ret = vm.FalseValue
					}

					return
				},
			},

			// find(text) 			=> array(array({text:,begin:,end:}))/undefined
			// find(text, maxCount) => array(array({text:,begin:,end:}))/undefined
			"find": &vm.BuiltinFunction{
				Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
					numArgs := len(args)
					if numArgs != 1 && numArgs != 2 {
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

					if numArgs == 1 {
						m := re.FindStringSubmatchIndex(s1)
						if m == nil {
							ret = vm.UndefinedValue
							return
						}

						arr := &vm.Array{}
						for i := 0; i < len(m); i += 2 {
							arr.Value = append(arr.Value,
								&vm.ImmutableMap{
									Value: map[string]vm.Object{
										"text": &vm.String{
											Value: s1[m[i]:m[i+1]],
										},
										"begin": &vm.Int{
											Value: int64(m[i]),
										},
										"end": &vm.Int{
											Value: int64(m[i+1]),
										},
									}})
						}

						ret = &vm.Array{Value: []vm.Object{arr}}

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
					m := re.FindAllStringSubmatchIndex(s1, i2)
					if m == nil {
						ret = vm.UndefinedValue
						return
					}

					arr := &vm.Array{}
					for _, m := range m {
						subMatch := &vm.Array{}
						for i := 0; i < len(m); i += 2 {
							subMatch.Value = append(subMatch.Value,
								&vm.ImmutableMap{
									Value: map[string]vm.Object{
										"text": &vm.String{
											Value: s1[m[i]:m[i+1]],
										},
										"begin": &vm.Int{
											Value: int64(m[i]),
										},
										"end": &vm.Int{
											Value: int64(m[i+1]),
										},
									}})
						}

						arr.Value = append(arr.Value, subMatch)
					}

					ret = arr

					return
				},
			},

			// replace(src, repl) => string
			"replace": &vm.BuiltinFunction{
				Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
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

					s, ok := doTextRegexpReplace(re, s1, s2)
					if !ok {
						return nil, vm.ErrStringLimit
					}

					ret = &vm.String{Value: s}

					return
				},
			},

			// split(text) 			 => array(string)
			// split(text, maxCount) => array(string)
			"split": &vm.BuiltinFunction{
				Value: func(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
					numArgs := len(args)
					if numArgs != 1 && numArgs != 2 {
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

					var i2 = -1
					if numArgs > 1 {
						i2, ok = vm.ToInt(args[1])
						if !ok {
							err = vm.ErrInvalidArgumentType{
								Name:     "second",
								Expected: "int(compatible)",
								Found:    args[1].TypeName(),
							}
							return
						}
					}

					arr := &vm.Array{}
					for _, s := range re.Split(s1, i2) {
						arr.Value = append(arr.Value,
							&vm.String{Value: s})
					}

					ret = arr

					return
				},
			},
		},
	}
}

// Size-limit checking implementation of regexp.ReplaceAllString.
func doTextRegexpReplace(re *regexp.Regexp, src, repl string) (string, bool) {
	idx := 0
	out := ""
	for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
		var exp []byte
		exp = re.ExpandString(exp, repl, src, m)
		if len(out)+m[0]-idx+len(exp) > vm.MaxStringLen {
			return "", false
		}
		out += src[idx:m[0]] + string(exp)
		idx = m[1]
	}
	if idx < len(src) {
		if len(out)+len(src)-idx > vm.MaxStringLen {
			return "", false
		}
		out += src[idx:]
	}
	return out, true
}
