package shell

import (
	"context"
	"io"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Func("new(prompt string) (instance *Instance)					creates a new shell instance with the given prompt", shellNew).
	Func("new_from_config(config map) (instance *Instance)			creates a new shell instance from a config map", shellNewFromConfig)

func shellNew(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	prompt, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "prompt",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	inst, err := New(prompt)
	if err != nil {
		return module.WrapError(err), nil
	}
	return shellInstance(inst), nil
}

func shellNewFromConfig(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}

	var cfgMap map[string]vm.Object
	switch m := args[0].(type) {
	case *vm.Map:
		cfgMap = m.Value
	case *vm.ImmutableMap:
		cfgMap = m.Value
	default:
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}

	cfg := &Config{}

	if v, ok := cfgMap["prompt"]; ok {
		if s, ok := vm.ToString(v); ok {
			cfg.Prompt = s
		}
	}
	if v, ok := cfgMap["history_file"]; ok {
		if s, ok := vm.ToString(v); ok {
			cfg.HistoryFile = s
		}
	}
	if v, ok := cfgMap["history_limit"]; ok {
		if i, ok := vm.ToInt(v); ok {
			cfg.HistoryLimit = i
		}
	}
	if v, ok := cfgMap["disable_auto_save_history"]; ok {
		cfg.DisableAutoSaveHistory = !v.IsFalsy()
	}
	if v, ok := cfgMap["history_search_fold"]; ok {
		cfg.HistorySearchFold = !v.IsFalsy()
	}
	if v, ok := cfgMap["vim_mode"]; ok {
		cfg.VimMode = !v.IsFalsy()
	}
	if v, ok := cfgMap["interrupt_prompt"]; ok {
		if s, ok := vm.ToString(v); ok {
			cfg.InterruptPrompt = s
		}
	}
	if v, ok := cfgMap["eof_prompt"]; ok {
		if s, ok := vm.ToString(v); ok {
			cfg.EOFPrompt = s
		}
	}
	if v, ok := cfgMap["enable_mask"]; ok {
		cfg.EnableMask = !v.IsFalsy()
	}
	if v, ok := cfgMap["mask_rune"]; ok {
		if ch, ok := v.(*vm.Char); ok {
			cfg.MaskRune = ch.Value
		} else if s, ok := vm.ToString(v); ok && len(s) > 0 {
			cfg.MaskRune = []rune(s)[0]
		}
	}
	if v, ok := cfgMap["undo"]; ok {
		cfg.Undo = !v.IsFalsy()
	}

	inst, err := NewFromConfig(cfg)
	if err != nil {
		return module.WrapError(err), nil
	}
	return shellInstance(inst), nil
}

func shellInstance(inst *Instance) *vm.ImmutableMap {
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"readline": &vm.BuiltinFunction{
			Name: "readline",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 0 {
					return nil, vm.ErrWrongNumArguments
				}
				line, err := inst.ReadLine()
				if err != nil {
					if err == io.EOF {
						return vm.UndefinedValue, nil
					}
					if err == ErrInterrupt {
						return &vm.Error{Value: &vm.String{Value: "interrupt"}}, nil
					}
					return module.WrapError(err), nil
				}
				return &vm.String{Value: line}, nil
			},
		},
		"read_password": &vm.BuiltinFunction{
			Name: "read_password",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				prompt, ok := vm.ToString(args[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name:     "prompt",
						Expected: "string(compatible)",
						Found:    args[0].TypeName(),
					}
				}
				pw, err := inst.ReadPassword(prompt)
				if err != nil {
					if err == io.EOF {
						return vm.UndefinedValue, nil
					}
					if err == ErrInterrupt {
						return &vm.Error{Value: &vm.String{Value: "interrupt"}}, nil
					}
					return module.WrapError(err), nil
				}
				return &vm.Bytes{Value: pw}, nil
			},
		},
		"read_line_with_default": &vm.BuiltinFunction{
			Name: "read_line_with_default",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				def, ok := vm.ToString(args[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name:     "default",
						Expected: "string(compatible)",
						Found:    args[0].TypeName(),
					}
				}
				line, err := inst.ReadLineWithDefault(def)
				if err != nil {
					if err == io.EOF {
						return vm.UndefinedValue, nil
					}
					if err == ErrInterrupt {
						return &vm.Error{Value: &vm.String{Value: "interrupt"}}, nil
					}
					return module.WrapError(err), nil
				}
				return &vm.String{Value: line}, nil
			},
		},
		"set_prompt": &vm.BuiltinFunction{
			Name: "set_prompt",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				s, ok := vm.ToString(args[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name:     "prompt",
						Expected: "string(compatible)",
						Found:    args[0].TypeName(),
					}
				}
				inst.SetPrompt(s)
				return vm.UndefinedValue, nil
			},
		},
		"set_default": &vm.BuiltinFunction{
			Name: "set_default",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				s, ok := vm.ToString(args[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name:     "default",
						Expected: "string(compatible)",
						Found:    args[0].TypeName(),
					}
				}
				inst.SetDefault(s)
				return vm.UndefinedValue, nil
			},
		},
		"save_to_history": &vm.BuiltinFunction{
			Name:  "save_to_history",
			Value: module.Func(func(s string) error { return inst.SaveToHistory(s) }),
		},
		"set_vim_mode": &vm.BuiltinFunction{
			Name: "set_vim_mode",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				inst.SetVimMode(!args[0].IsFalsy())
				return vm.UndefinedValue, nil
			},
		},
		"is_vim_mode": &vm.BuiltinFunction{
			Name:  "is_vim_mode",
			Value: module.Func(inst.IsVimMode),
		},
		"close": &vm.BuiltinFunction{
			Name:  "close",
			Value: module.Func(func() error { return inst.Close() }),
		},
		"capture_exit_signal": &vm.BuiltinFunction{
			Name:  "capture_exit_signal",
			Value: module.Func(inst.CaptureExitSignal),
		},
		"write": &vm.BuiltinFunction{
			Name: "write",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				if len(args) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				var data []byte
				switch v := args[0].(type) {
				case *vm.Bytes:
					data = v.Value
				default:
					s, ok := vm.ToString(args[0])
					if !ok {
						return nil, vm.ErrInvalidArgumentType{
							Name:     "data",
							Expected: "string/bytes",
							Found:    args[0].TypeName(),
						}
					}
					data = []byte(s)
				}
				n, err := inst.Write(data)
				if err != nil {
					return module.WrapError(err), nil
				}
				return &vm.Int{Value: int64(n)}, nil
			},
		},
		"refresh": &vm.BuiltinFunction{
			Name:  "refresh",
			Value: module.Func(inst.Refresh),
		},
		"clear_screen": &vm.BuiltinFunction{
			Name:  "clear_screen",
			Value: module.Func(inst.ClearScreen),
		},
		"disable_history": &vm.BuiltinFunction{
			Name:  "disable_history",
			Value: module.Func(inst.DisableHistory),
		},
		"enable_history": &vm.BuiltinFunction{
			Name:  "enable_history",
			Value: module.Func(inst.EnableHistory),
		},
		"reset_history": &vm.BuiltinFunction{
			Name:  "reset_history",
			Value: module.Func(inst.ResetHistory),
		},
	}}
}
