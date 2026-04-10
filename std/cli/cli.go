package cli

import (
	"context"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
	flag "github.com/spf13/pflag"
)

func init() {
	Module = module.NewBuiltin().
		Func("new(config map) (app *App)								creates a new CLI application from the given configuration map", cliNew).
		Func("command(config map) (command map)							creates a command configuration", cliCommand).
		Func("string_flag(config map) (flag map)						creates a string flag configuration", cliStringFlag).
		Func("bool_flag(config map) (flag map)							creates a bool flag configuration", cliBoolFlag).
		Func("int_flag(config map) (flag map)							creates an int flag configuration", cliIntFlag).
		Func("float_flag(config map) (flag map)							creates a float flag configuration", cliFloatFlag).
		Func("string_slice_flag(config map) (flag map)					creates a string slice flag configuration", cliStringSliceFlag)
}

// ---------------------------------------------------------------------------
// Config-map helpers
// ---------------------------------------------------------------------------

func cfgMap(obj vm.Object) map[string]vm.Object {
	switch m := obj.(type) {
	case *vm.Map:
		return m.Value
	case *vm.ImmutableMap:
		return m.Value
	}
	return nil
}

func cfgStr(m map[string]vm.Object, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := vm.ToString(v); ok {
			return s
		}
	}
	return ""
}

func cfgBool(m map[string]vm.Object, key string) bool {
	if v, ok := m[key]; ok {
		return !v.IsFalsy()
	}
	return false
}

func cfgArray(m map[string]vm.Object, key string) []vm.Object {
	if v, ok := m[key]; ok {
		switch a := v.(type) {
		case *vm.Array:
			return a.Value
		case *vm.ImmutableArray:
			return a.Value
		}
	}
	return nil
}

func cfgStrArray(m map[string]vm.Object, key string) []string {
	arr := cfgArray(m, key)
	var result []string
	for _, elem := range arr {
		if s, ok := vm.ToString(elem); ok {
			result = append(result, s)
		}
	}
	return result
}

func cfgFunc(m map[string]vm.Object, key string) vm.Object {
	if v, ok := m[key]; ok && v.CanCall() {
		return v
	}
	return nil
}

func cfgInt(m map[string]vm.Object, key string) (int, bool) {
	if v, ok := m[key]; ok {
		if i, ok := vm.ToInt(v); ok {
			return i, true
		}
	}
	return 0, false
}

func cfgFloat(m map[string]vm.Object, key string) (float64, bool) {
	if v, ok := m[key]; ok {
		if f, ok := vm.ToFloat64(v); ok {
			return f, true
		}
	}
	return 0, false
}

// ---------------------------------------------------------------------------
// Flag descriptor helpers
// ---------------------------------------------------------------------------

func makeFlagMap(flagType string, cfg map[string]vm.Object) *vm.ImmutableMap {
	m := make(map[string]vm.Object, len(cfg)+1)
	m["__flag_type"] = &vm.String{Value: flagType}
	for k, v := range cfg {
		m[k] = v
	}
	return &vm.ImmutableMap{Value: m}
}

// getShorthand returns the first single-character alias (for pflag shorthand).
func getShorthand(aliases []string) string {
	for _, a := range aliases {
		if len(a) == 1 {
			return a
		}
	}
	return ""
}

// boolObj returns vm.TrueValue or vm.FalseValue.
func boolObj(b bool) vm.Object {
	if b {
		return vm.TrueValue
	}
	return vm.FalseValue
}

// ---------------------------------------------------------------------------
// Module entry points
// ---------------------------------------------------------------------------

func cliNew(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}

	// Build the root command (persistent = true so root flags/hooks
	// propagate to subcommands).
	cmd := buildCommand(ctx, cfg, true)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	hideVersion := cfgBool(cfg, "hide_version")
	hideHelp := cfgBool(cfg, "hide_help")
	hideHelpCommand := cfgBool(cfg, "hide_help_command")

	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"run": &vm.BuiltinFunction{
			Name: "run",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) > 1 {
					return nil, vm.ErrWrongNumArguments
				}

				if len(fnArgs) == 1 {
					var runArgs []string
					switch a := fnArgs[0].(type) {
					case *vm.Array:
						for _, elem := range a.Value {
							if s, ok := vm.ToString(elem); ok {
								runArgs = append(runArgs, s)
							}
						}
					case *vm.ImmutableArray:
						for _, elem := range a.Value {
							if s, ok := vm.ToString(elem); ok {
								runArgs = append(runArgs, s)
							}
						}
					default:
						return nil, vm.ErrInvalidArgumentType{
							Name:     "args",
							Expected: "array",
							Found:    fnArgs[0].TypeName(),
						}
					}
					// First element is the program name — skip it,
					// mimicking os.Args[1:].
					if len(runArgs) > 1 {
						cmd.SetArgs(runArgs[1:])
					} else {
						cmd.SetArgs([]string{})
					}
				}
				// If no args supplied, cobra defaults to os.Args[1:].

				// Handle hide_version / hide_help before execution.
				if hideVersion && cmd.Version != "" {
					cmd.InitDefaultVersionFlag()
					if f := cmd.Flags().Lookup("version"); f != nil {
						f.Hidden = true
					}
				}
				if hideHelp {
					cmd.InitDefaultHelpFlag()
					if f := cmd.Flags().Lookup("help"); f != nil {
						f.Hidden = true
					}
				}
				if hideHelpCommand {
					cmd.SetHelpCommand(&Command{Hidden: true, Use: "no-help"})
				}

				if err := cmd.Execute(); err != nil {
					return module.WrapError(err), nil
				}
				return vm.UndefinedValue, nil
			},
		},
	}}, nil
}

func cliCommand(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}
	// Return a copy of the config — it will be consumed by buildCommand
	// when the parent command processes its "commands" array.
	m := make(map[string]vm.Object, len(cfg)+1)
	for k, v := range cfg {
		m[k] = v
	}
	m["__is_command"] = vm.TrueValue
	return &vm.ImmutableMap{Value: m}, nil
}

func cliStringFlag(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}
	return makeFlagMap("string", cfg), nil
}

func cliBoolFlag(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}
	return makeFlagMap("bool", cfg), nil
}

func cliIntFlag(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}
	return makeFlagMap("int", cfg), nil
}

func cliFloatFlag(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}
	return makeFlagMap("float", cfg), nil
}

func cliStringSliceFlag(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	cfg := cfgMap(args[0])
	if cfg == nil {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "config",
			Expected: "map",
			Found:    args[0].TypeName(),
		}
	}
	return makeFlagMap("string_slice", cfg), nil
}

// ---------------------------------------------------------------------------
// Command builder
// ---------------------------------------------------------------------------

// buildCommand translates a rumo config map into a *Command.
// If persistent is true the hooks and flags are registered as persistent
// (i.e. inherited by subcommands) — used for the root app command.
func buildCommand(ctx context.Context, cfg map[string]vm.Object, persistent bool) *Command {
	cmd := &Command{}

	// name / use
	if name := cfgStr(cfg, "name"); name != "" {
		cmd.Use = name
	}
	if usageText := cfgStr(cfg, "usage_text"); usageText != "" {
		cmd.Use = usageText
	}
	if argsUsage := cfgStr(cfg, "args_usage"); argsUsage != "" && cmd.Use != "" {
		cmd.Use = cmd.Use + " " + argsUsage
	}

	// descriptions
	if usage := cfgStr(cfg, "usage"); usage != "" {
		cmd.Short = usage
	}
	if desc := cfgStr(cfg, "description"); desc != "" {
		cmd.Long = desc
	}

	// version
	if version := cfgStr(cfg, "version"); version != "" {
		cmd.Version = version
	}

	// aliases
	if aliases := cfgStrArray(cfg, "aliases"); len(aliases) > 0 {
		cmd.Aliases = aliases
	}

	// category → GroupID
	if category := cfgStr(cfg, "category"); category != "" {
		cmd.GroupID = category
	}

	// boolean options
	if cfgBool(cfg, "hidden") {
		cmd.Hidden = true
	}
	if cfgBool(cfg, "skip_flag_parsing") {
		cmd.DisableFlagParsing = true
	}

	// --- flags ---
	addFlags(cfg, cmd, persistent)

	// --- action / before / after hooks ---
	if actionFn := cfgFunc(cfg, "action"); actionFn != nil {
		fn := actionFn
		cmd.RunE = func(c *Command, a []string) error {
			_, err := vm.CallFunc(ctx, fn, wrapContext(c, a))
			return err
		}
	}

	if beforeFn := cfgFunc(cfg, "before"); beforeFn != nil {
		fn := beforeFn
		if persistent {
			cmd.PersistentPreRunE = func(c *Command, a []string) error {
				_, err := vm.CallFunc(ctx, fn, wrapContext(c, a))
				return err
			}
		} else {
			cmd.PreRunE = func(c *Command, a []string) error {
				_, err := vm.CallFunc(ctx, fn, wrapContext(c, a))
				return err
			}
		}
	}

	if afterFn := cfgFunc(cfg, "after"); afterFn != nil {
		fn := afterFn
		if persistent {
			cmd.PersistentPostRunE = func(c *Command, a []string) error {
				_, err := vm.CallFunc(ctx, fn, wrapContext(c, a))
				return err
			}
		} else {
			cmd.PostRunE = func(c *Command, a []string) error {
				_, err := vm.CallFunc(ctx, fn, wrapContext(c, a))
				return err
			}
		}
	}

	// --- subcommands ---
	for _, cmdObj := range cfgArray(cfg, "commands") {
		subCfg := cfgMap(cmdObj)
		if subCfg != nil {
			subCmd := buildCommand(ctx, subCfg, false)
			cmd.AddCommand(subCmd)
		}
	}

	return cmd
}

// addFlags reads the "flags" array from cfg and registers each flag on cmd.
func addFlags(cfg map[string]vm.Object, cmd *Command, persistent bool) {
	for _, flagObj := range cfgArray(cfg, "flags") {
		flagCfg := cfgMap(flagObj)
		if flagCfg == nil {
			continue
		}

		flagType := cfgStr(flagCfg, "__flag_type")
		name := cfgStr(flagCfg, "name")
		if name == "" {
			continue
		}

		aliases := cfgStrArray(flagCfg, "aliases")
		shorthand := getShorthand(aliases)
		usage := cfgStr(flagCfg, "usage")
		required := cfgBool(flagCfg, "required")
		hidden := cfgBool(flagCfg, "hidden")

		var flags *flag.FlagSet
		if persistent {
			flags = cmd.PersistentFlags()
		} else {
			flags = cmd.Flags()
		}

		switch flagType {
		case "string":
			defVal := cfgStr(flagCfg, "value")
			flags.StringP(name, shorthand, defVal, usage)
		case "bool":
			defVal := cfgBool(flagCfg, "value")
			flags.BoolP(name, shorthand, defVal, usage)
		case "int":
			defVal := 0
			if v, ok := cfgInt(flagCfg, "value"); ok {
				defVal = v
			}
			flags.IntP(name, shorthand, defVal, usage)
		case "float":
			defVal := 0.0
			if v, ok := cfgFloat(flagCfg, "value"); ok {
				defVal = v
			}
			flags.Float64P(name, shorthand, defVal, usage)
		case "string_slice":
			defVal := cfgStrArray(flagCfg, "value")
			flags.StringArrayP(name, shorthand, defVal, usage)
		default:
			continue
		}

		if required {
			_ = cmd.MarkFlagRequired(name)
		}
		if hidden {
			_ = flags.MarkHidden(name)
		}

		// env_vars: set default from environment if the flag hasn't been
		// explicitly provided on the command-line.
		for _, envVar := range cfgStrArray(flagCfg, "env_vars") {
			if val := EnvLookupFunc(envVar); val != "" {
				if f := flags.Lookup(name); f != nil {
					_ = f.Value.Set(val)
					f.DefValue = val
				}
				break
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Context wrapper — the object passed to action/before/after callbacks
// ---------------------------------------------------------------------------

func wrapContext(cmd *Command, args []string) *vm.ImmutableMap {
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"args": &vm.BuiltinFunction{
			Name: "args",
			Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
				arr := &vm.Array{}
				for _, a := range args {
					arr.Value = append(arr.Value, &vm.String{Value: a})
				}
				return arr, nil
			},
		},
		"narg": &vm.BuiltinFunction{
			Name: "narg",
			Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
				return &vm.Int{Value: int64(len(args))}, nil
			},
		},
		"string": &vm.BuiltinFunction{
			Name: "string",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				val, err := cmd.Flags().GetString(name)
				if err != nil {
					return &vm.String{Value: ""}, nil
				}
				return &vm.String{Value: val}, nil
			},
		},
		"bool": &vm.BuiltinFunction{
			Name: "bool",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				val, err := cmd.Flags().GetBool(name)
				if err != nil {
					return vm.FalseValue, nil
				}
				return boolObj(val), nil
			},
		},
		"int": &vm.BuiltinFunction{
			Name: "int",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				val, err := cmd.Flags().GetInt(name)
				if err != nil {
					return &vm.Int{Value: 0}, nil
				}
				return &vm.Int{Value: int64(val)}, nil
			},
		},
		"float": &vm.BuiltinFunction{
			Name: "float",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				val, err := cmd.Flags().GetFloat64(name)
				if err != nil {
					return &vm.Float{Value: 0}, nil
				}
				return &vm.Float{Value: val}, nil
			},
		},
		"string_slice": &vm.BuiltinFunction{
			Name: "string_slice",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				val, err := cmd.Flags().GetStringArray(name)
				if err != nil {
					return &vm.Array{}, nil
				}
				arr := &vm.Array{}
				for _, s := range val {
					arr.Value = append(arr.Value, &vm.String{Value: s})
				}
				return arr, nil
			},
		},
		"is_set": &vm.BuiltinFunction{
			Name: "is_set",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				f := cmd.Flags().Lookup(name)
				return boolObj(f != nil && f.Changed), nil
			},
		},
		"count": &vm.BuiltinFunction{
			Name: "count",
			Value: func(_ context.Context, fnArgs ...vm.Object) (vm.Object, error) {
				if len(fnArgs) != 1 {
					return nil, vm.ErrWrongNumArguments
				}
				name, ok := vm.ToString(fnArgs[0])
				if !ok {
					return nil, vm.ErrInvalidArgumentType{
						Name: "name", Expected: "string", Found: fnArgs[0].TypeName(),
					}
				}
				f := cmd.Flags().Lookup(name)
				if f != nil && f.Changed {
					return &vm.Int{Value: 1}, nil
				}
				return &vm.Int{Value: 0}, nil
			},
		},
		"num_flags": &vm.BuiltinFunction{
			Name: "num_flags",
			Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
				n := 0
				cmd.Flags().Visit(func(_ *flag.Flag) { n++ })
				return &vm.Int{Value: int64(n)}, nil
			},
		},
		"flag_names": &vm.BuiltinFunction{
			Name: "flag_names",
			Value: func(_ context.Context, _ ...vm.Object) (vm.Object, error) {
				arr := &vm.Array{}
				cmd.Flags().VisitAll(func(f *flag.Flag) {
					arr.Value = append(arr.Value, &vm.String{Value: f.Name})
				})
				return arr, nil
			},
		},
	}}
}
