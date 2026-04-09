package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

// Module provides a declarative CLI framework.
var Module = module.NewBuiltin().
	Func("new(config map) (app *App)                                creates a new CLI app from the given configuration map", cliNew).
	Func("command(config map) (cmd map)                             creates a command configuration", cliCommand).
	Func("string_flag(config map) (flag map)                        creates a string flag", cliStringFlag).
	Func("bool_flag(config map) (flag map)                          creates a bool flag", cliBoolFlag).
	Func("int_flag(config map) (flag map)                           creates an int flag", cliIntFlag).
	Func("float_flag(config map) (flag map)                         creates a float flag", cliFloatFlag).
	Func("string_slice_flag(config map) (flag map)                  creates a string slice flag", cliStringSliceFlag)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// fn creates a BuiltinFunction.
func fn(name string, f vm.CallableFunc) vm.Object {
	return &vm.BuiltinFunction{Name: name, Value: f}
}

// boolVal converts a Go bool to the appropriate vm.Object.
func boolVal(b bool) vm.Object {
	if b {
		return vm.TrueValue
	}
	return vm.FalseValue
}

// callFunc calls a vm.Object function, handling CompiledFunction by creating
// a shallow-cloned VM (same as the start builtin). For BuiltinFunction and
// other callable objects it falls back to Object.Call().
func callFunc(ctx context.Context, fn vm.Object, args ...vm.Object) (vm.Object, error) {
	if cfn, ok := fn.(*vm.CompiledFunction); ok {
		if vmVal := ctx.Value(vm.ContextKey("vm")); vmVal != nil {
			parentVM := vmVal.(*vm.VM)
			clone := parentVM.ShallowClone()
			return clone.RunCompiled(cfn, args...)
		}
		return nil, fmt.Errorf("no VM in context to run compiled function")
	}
	return fn.Call(ctx, args...)
}

// flagPtr wraps a cli.Flag so it can be stored in an ImmutableMap.
type flagPtr struct {
	vm.ObjectImpl
	f Flag
}

func (o *flagPtr) TypeName() string { return "cli-flag-ptr" }
func (o *flagPtr) String() string   { return "<cli-flag-ptr>" }
func (o *flagPtr) Copy() vm.Object  { return o }

// commandPtr wraps a *Command so it can be stored in an ImmutableMap.
type commandPtr struct {
	vm.ObjectImpl
	c *Command
}

func (o *commandPtr) TypeName() string { return "cli-command-ptr" }
func (o *commandPtr) String() string   { return "<cli-command-ptr>" }
func (o *commandPtr) Copy() vm.Object  { return o }

// getMap extracts a map[string]vm.Object from a vm.Object (supports Map and ImmutableMap).
func getMap(obj vm.Object) (map[string]vm.Object, bool) {
	switch m := obj.(type) {
	case *vm.Map:
		return m.Value, true
	case *vm.ImmutableMap:
		return m.Value, true
	}
	return nil, false
}

// getStringSlice converts a vm.Object (Array or ImmutableArray) to []string.
func getStringSlice(obj vm.Object) ([]string, bool) {
	var elems []vm.Object
	switch a := obj.(type) {
	case *vm.Array:
		elems = a.Value
	case *vm.ImmutableArray:
		elems = a.Value
	default:
		return nil, false
	}
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		s, ok := vm.ToString(e)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// mapStr returns a string from a map entry, or empty string if missing.
func mapStr(m map[string]vm.Object, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := vm.ToString(v)
	return s
}

// mapBool returns a bool from a map entry.
func mapBool(m map[string]vm.Object, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	return !v.IsFalsy()
}

// mapStringSlice returns a []string from a map entry.
func mapStringSlice(m map[string]vm.Object, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	ss, _ := getStringSlice(v)
	return ss
}

// mapInt returns an int from a map entry.
func mapInt(m map[string]vm.Object, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	return vm.ToInt(v)
}

// mapFloat returns a float64 from a map entry.
func mapFloat(m map[string]vm.Object, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	return vm.ToFloat64(v)
}

// ---------------------------------------------------------------------------
// Flag builders
// ---------------------------------------------------------------------------

func cliStringFlag(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}
	f := &StringFlag{
		Name:     mapStr(m, "name"),
		Aliases:  mapStringSlice(m, "aliases"),
		Usage:    mapStr(m, "usage"),
		EnvVars:  mapStringSlice(m, "env_vars"),
		Value:    mapStr(m, "value"),
		Required: mapBool(m, "required"),
		Hidden:   mapBool(m, "hidden"),
	}
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"__flag": &flagPtr{f: f},
	}}, nil
}

func cliBoolFlag(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}
	f := &BoolFlag{
		Name:     mapStr(m, "name"),
		Aliases:  mapStringSlice(m, "aliases"),
		Usage:    mapStr(m, "usage"),
		EnvVars:  mapStringSlice(m, "env_vars"),
		Value:    mapBool(m, "value"),
		Required: mapBool(m, "required"),
		Hidden:   mapBool(m, "hidden"),
	}
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"__flag": &flagPtr{f: f},
	}}, nil
}

func cliIntFlag(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}
	f := &IntFlag{
		Name:     mapStr(m, "name"),
		Aliases:  mapStringSlice(m, "aliases"),
		Usage:    mapStr(m, "usage"),
		EnvVars:  mapStringSlice(m, "env_vars"),
		Required: mapBool(m, "required"),
		Hidden:   mapBool(m, "hidden"),
	}
	if v, ok := mapInt(m, "value"); ok {
		f.Value = v
	}
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"__flag": &flagPtr{f: f},
	}}, nil
}

func cliFloatFlag(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}
	f := &Float64Flag{
		Name:     mapStr(m, "name"),
		Aliases:  mapStringSlice(m, "aliases"),
		Usage:    mapStr(m, "usage"),
		EnvVars:  mapStringSlice(m, "env_vars"),
		Required: mapBool(m, "required"),
		Hidden:   mapBool(m, "hidden"),
	}
	if v, ok := mapFloat(m, "value"); ok {
		f.Value = v
	}
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"__flag": &flagPtr{f: f},
	}}, nil
}

func cliStringSliceFlag(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}
	f := &StringSliceFlag{
		Name:     mapStr(m, "name"),
		Aliases:  mapStringSlice(m, "aliases"),
		Usage:    mapStr(m, "usage"),
		EnvVars:  mapStringSlice(m, "env_vars"),
		Required: mapBool(m, "required"),
		Hidden:   mapBool(m, "hidden"),
	}
	if defaults := mapStringSlice(m, "value"); len(defaults) > 0 {
		f.Value = NewStringSlice(defaults...)
	}
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"__flag": &flagPtr{f: f},
	}}, nil
}

// ---------------------------------------------------------------------------
// Flag extraction
// ---------------------------------------------------------------------------

// extractFlag retrieves a cli.Flag from an ImmutableMap produced by a flag builder.
func extractFlag(obj vm.Object) (Flag, bool) {
	m, ok := obj.(*vm.ImmutableMap)
	if !ok {
		return nil, false
	}
	fp, ok := m.Value["__flag"]
	if !ok {
		return nil, false
	}
	ptr, ok := fp.(*flagPtr)
	if !ok {
		return nil, false
	}
	return ptr.f, true
}

// extractFlags converts a vm.Array/ImmutableArray of flag maps into []Flag.
func extractFlags(obj vm.Object) ([]Flag, error) {
	var elems []vm.Object
	switch a := obj.(type) {
	case *vm.Array:
		elems = a.Value
	case *vm.ImmutableArray:
		elems = a.Value
	default:
		return nil, fmt.Errorf("flags must be an array, got %s", obj.TypeName())
	}
	flags := make([]Flag, 0, len(elems))
	for _, e := range elems {
		f, ok := extractFlag(e)
		if !ok {
			return nil, fmt.Errorf("invalid flag entry: %s", e.TypeName())
		}
		flags = append(flags, f)
	}
	return flags, nil
}

// ---------------------------------------------------------------------------
// Command builder
// ---------------------------------------------------------------------------

func cliCommand(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}
	cmd, err := buildCommand(ctx, m)
	if err != nil {
		return nil, err
	}
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"__command": &commandPtr{c: cmd},
	}}, nil
}

// buildCommand constructs a *Command from a config map.
func buildCommand(ctx context.Context, m map[string]vm.Object) (*Command, error) {
	cmd := &Command{
		Name:                   mapStr(m, "name"),
		Aliases:                mapStringSlice(m, "aliases"),
		Usage:                  mapStr(m, "usage"),
		UsageText:              mapStr(m, "usage_text"),
		Description:            mapStr(m, "description"),
		ArgsUsage:              mapStr(m, "args_usage"),
		Category:               mapStr(m, "category"),
		Hidden:                 mapBool(m, "hidden"),
		HideHelp:               mapBool(m, "hide_help"),
		HideHelpCommand:        mapBool(m, "hide_help_command"),
		SkipFlagParsing:        mapBool(m, "skip_flag_parsing"),
		UseShortOptionHandling: mapBool(m, "use_short_option_handling"),
		Args:                   mapBool(m, "args"),
	}

	// Flags
	if v, ok := m["flags"]; ok {
		flags, err := extractFlags(v)
		if err != nil {
			return nil, err
		}
		cmd.Flags = flags
	}

	// Subcommands
	if v, ok := m["commands"]; ok {
		subs, err := extractCommands(ctx, v)
		if err != nil {
			return nil, err
		}
		cmd.Subcommands = subs
	}

	// Action callback
	if v, ok := m["action"]; ok && v.CanCall() {
		cb := v
		cmd.Action = func(cCtx *Context) error {
			ret, err := callFunc(ctx, cb, wrapContext(ctx, cCtx))
			if err != nil {
				return err
			}
			if ret != nil {
				if e, ok := ret.(*vm.Error); ok {
					return fmt.Errorf("%s", e.Value)
				}
			}
			return nil
		}
	}

	// Before callback
	if v, ok := m["before"]; ok && v.CanCall() {
		cb := v
		cmd.Before = func(cCtx *Context) error {
			ret, err := callFunc(ctx, cb, wrapContext(ctx, cCtx))
			if err != nil {
				return err
			}
			if ret != nil {
				if e, ok := ret.(*vm.Error); ok {
					return fmt.Errorf("%s", e.Value)
				}
			}
			return nil
		}
	}

	// After callback
	if v, ok := m["after"]; ok && v.CanCall() {
		cb := v
		cmd.After = func(cCtx *Context) error {
			ret, err := callFunc(ctx, cb, wrapContext(ctx, cCtx))
			if err != nil {
				return err
			}
			if ret != nil {
				if e, ok := ret.(*vm.Error); ok {
					return fmt.Errorf("%s", e.Value)
				}
			}
			return nil
		}
	}

	return cmd, nil
}

// extractCommands converts a vm.Array/ImmutableArray of command maps into []*Command.
func extractCommands(ctx context.Context, obj vm.Object) ([]*Command, error) {
	var elems []vm.Object
	switch a := obj.(type) {
	case *vm.Array:
		elems = a.Value
	case *vm.ImmutableArray:
		elems = a.Value
	default:
		return nil, fmt.Errorf("commands must be an array, got %s", obj.TypeName())
	}
	cmds := make([]*Command, 0, len(elems))
	for _, e := range elems {
		// Try extracting a pre-built command (from cli.command())
		if m, ok := e.(*vm.ImmutableMap); ok {
			if cp, ok := m.Value["__command"]; ok {
				if ptr, ok := cp.(*commandPtr); ok {
					cmds = append(cmds, ptr.c)
					continue
				}
			}
		}
		// Otherwise, treat as inline config map
		em, ok := getMap(e)
		if !ok {
			return nil, fmt.Errorf("invalid command entry: %s", e.TypeName())
		}
		cmd, err := buildCommand(ctx, em)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

// ---------------------------------------------------------------------------
// App builder
// ---------------------------------------------------------------------------

func cliNew(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := getMap(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "config", Expected: "map", Found: args[0].TypeName()}
	}

	app := NewApp()

	if v := mapStr(m, "name"); v != "" {
		app.Name = v
	}
	if v := mapStr(m, "usage"); v != "" {
		app.Usage = v
	}
	if v := mapStr(m, "usage_text"); v != "" {
		app.UsageText = v
	}
	if v := mapStr(m, "version"); v != "" {
		app.Version = v
	}
	if v := mapStr(m, "description"); v != "" {
		app.Description = v
	}
	if v := mapStr(m, "args_usage"); v != "" {
		app.ArgsUsage = v
	}
	if v := mapStr(m, "copyright"); v != "" {
		app.Copyright = v
	}
	if v := mapStr(m, "default_command"); v != "" {
		app.DefaultCommand = v
	}
	app.HideHelp = mapBool(m, "hide_help")
	app.HideHelpCommand = mapBool(m, "hide_help_command")
	app.HideVersion = mapBool(m, "hide_version")
	app.EnableBashCompletion = mapBool(m, "enable_bash_completion")
	app.UseShortOptionHandling = mapBool(m, "use_short_option_handling")
	app.Suggest = mapBool(m, "suggest")
	app.SkipFlagParsing = mapBool(m, "skip_flag_parsing")
	app.Args = mapBool(m, "args")

	// Flags
	if v, ok := m["flags"]; ok {
		flags, err := extractFlags(v)
		if err != nil {
			return nil, err
		}
		app.Flags = flags
	}

	// Commands
	if v, ok := m["commands"]; ok {
		cmds, err := extractCommands(ctx, v)
		if err != nil {
			return nil, err
		}
		app.Commands = cmds
	}

	// Authors
	if v, ok := m["authors"]; ok {
		authors, err := extractAuthors(v)
		if err != nil {
			return nil, err
		}
		app.Authors = authors
	}

	// Action callback
	if v, ok := m["action"]; ok && v.CanCall() {
		cb := v
		app.Action = func(cCtx *Context) error {
			ret, err := callFunc(ctx, cb, wrapContext(ctx, cCtx))
			if err != nil {
				return err
			}
			if ret != nil {
				if e, ok := ret.(*vm.Error); ok {
					return fmt.Errorf("%s", e.Value)
				}
			}
			return nil
		}
	}

	// Before callback
	if v, ok := m["before"]; ok && v.CanCall() {
		cb := v
		app.Before = func(cCtx *Context) error {
			ret, err := callFunc(ctx, cb, wrapContext(ctx, cCtx))
			if err != nil {
				return err
			}
			if ret != nil {
				if e, ok := ret.(*vm.Error); ok {
					return fmt.Errorf("%s", e.Value)
				}
			}
			return nil
		}
	}

	// After callback
	if v, ok := m["after"]; ok && v.CanCall() {
		cb := v
		app.After = func(cCtx *Context) error {
			ret, err := callFunc(ctx, cb, wrapContext(ctx, cCtx))
			if err != nil {
				return err
			}
			if ret != nil {
				if e, ok := ret.(*vm.Error); ok {
					return fmt.Errorf("%s", e.Value)
				}
			}
			return nil
		}
	}

	// Wire VM stdout/stderr to the app if available
	if vmVal := ctx.Value(vm.ContextKey("vm")); vmVal != nil {
		if v, ok := vmVal.(*vm.VM); ok {
			app.Writer = v.Out
		}
	}

	return wrapApp(app, ctx), nil
}

// extractAuthors extracts []*Author from a vm array of maps.
func extractAuthors(obj vm.Object) ([]*Author, error) {
	var elems []vm.Object
	switch a := obj.(type) {
	case *vm.Array:
		elems = a.Value
	case *vm.ImmutableArray:
		elems = a.Value
	default:
		return nil, fmt.Errorf("authors must be an array, got %s", obj.TypeName())
	}
	authors := make([]*Author, 0, len(elems))
	for _, e := range elems {
		m, ok := getMap(e)
		if !ok {
			return nil, fmt.Errorf("author entry must be a map, got %s", e.TypeName())
		}
		authors = append(authors, &Author{
			Name:  mapStr(m, "name"),
			Email: mapStr(m, "email"),
		})
	}
	return authors, nil
}

// ---------------------------------------------------------------------------
// App wrapper
// ---------------------------------------------------------------------------

func wrapApp(app *App, vmCtx context.Context) *vm.ImmutableMap {
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"run": fn("run", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			var arguments []string
			switch len(args) {
			case 0:
				arguments = os.Args
			case 1:
				var elems []vm.Object
				switch a := args[0].(type) {
				case *vm.Array:
					elems = a.Value
				case *vm.ImmutableArray:
					elems = a.Value
				default:
					return nil, vm.ErrInvalidArgumentType{Name: "args", Expected: "array", Found: args[0].TypeName()}
				}
				arguments = make([]string, 0, len(elems))
				for _, e := range elems {
					s, ok := vm.ToString(e)
					if !ok {
						return nil, vm.ErrInvalidArgumentType{Name: "args[]", Expected: "string", Found: e.TypeName()}
					}
					arguments = append(arguments, s)
				}
			default:
				return nil, vm.ErrWrongNumArguments
			}
			if err := app.RunContext(vmCtx, arguments); err != nil {
				return module.WrapError(err), nil
			}
			return vm.UndefinedValue, nil
		}),
	}}
}

// ---------------------------------------------------------------------------
// Context wrapper
// ---------------------------------------------------------------------------

func wrapContext(vmCtx context.Context, cCtx *Context) *vm.ImmutableMap {
	return &vm.ImmutableMap{Value: map[string]vm.Object{
		"args": fn("args", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			cliArgs := cCtx.Args().Slice()
			arr := &vm.Array{Value: make([]vm.Object, 0, len(cliArgs))}
			for _, s := range cliArgs {
				arr.Value = append(arr.Value, &vm.String{Value: s})
			}
			return arr, nil
		}),
		"narg": fn("narg", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			return &vm.Int{Value: int64(cCtx.NArg())}, nil
		}),
		"string": fn("string", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			return &vm.String{Value: cCtx.String(name)}, nil
		}),
		"bool": fn("bool", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			return boolVal(cCtx.Bool(name)), nil
		}),
		"int": fn("int", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			return &vm.Int{Value: int64(cCtx.Int(name))}, nil
		}),
		"float": fn("float", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			return &vm.Float{Value: cCtx.Float64(name)}, nil
		}),
		"string_slice": fn("string_slice", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			ss := cCtx.StringSlice(name)
			arr := &vm.Array{Value: make([]vm.Object, 0, len(ss))}
			for _, s := range ss {
				arr.Value = append(arr.Value, &vm.String{Value: s})
			}
			return arr, nil
		}),
		"is_set": fn("is_set", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			return boolVal(cCtx.IsSet(name)), nil
		}),
		"count": fn("count", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			name, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[0].TypeName()}
			}
			return &vm.Int{Value: int64(cCtx.Count(name))}, nil
		}),
		"num_flags": fn("num_flags", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			return &vm.Int{Value: int64(cCtx.NumFlags())}, nil
		}),
		"flag_names": fn("flag_names", func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			names := cCtx.FlagNames()
			arr := &vm.Array{Value: make([]vm.Object, 0, len(names))}
			for _, n := range names {
				arr.Value = append(arr.Value, &vm.String{Value: n})
			}
			return arr, nil
		}),
	}}
}

