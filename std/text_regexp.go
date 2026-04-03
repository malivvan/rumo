package std

import (
	"context"
	"regexp"

	"github.com/malivvan/vv/vm"
)

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
