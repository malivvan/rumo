package cli_test

import (
	"context"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

// ---------------------------------------------------------------------------
// Flag builder tests (via require.Module)
// ---------------------------------------------------------------------------

func TestStringFlag(t *testing.T) {
	res := require.Module(t, "cli").Call("string_flag", require.MAP{
		"name":    "config",
		"aliases": require.ARR{"c"},
		"usage":   "config file path",
		"value":   "default.yaml",
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	// verify wrong arg count
	require.Module(t, "cli").Call("string_flag").ExpectError()

	// verify wrong arg type
	require.Module(t, "cli").Call("string_flag", "not-a-map").ExpectError()
}

func TestBoolFlag(t *testing.T) {
	res := require.Module(t, "cli").Call("bool_flag", require.MAP{
		"name":    "verbose",
		"aliases": require.ARR{"v"},
		"usage":   "enable verbose output",
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	require.Module(t, "cli").Call("bool_flag").ExpectError()
	require.Module(t, "cli").Call("bool_flag", 42).ExpectError()
}

func TestIntFlag(t *testing.T) {
	res := require.Module(t, "cli").Call("int_flag", require.MAP{
		"name":    "port",
		"aliases": require.ARR{"p"},
		"usage":   "port number",
		"value":   8080,
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	require.Module(t, "cli").Call("int_flag").ExpectError()
	require.Module(t, "cli").Call("int_flag", "bad").ExpectError()
}

func TestFloatFlag(t *testing.T) {
	res := require.Module(t, "cli").Call("float_flag", require.MAP{
		"name":  "rate",
		"usage": "rate limit",
		"value": 1.5,
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	require.Module(t, "cli").Call("float_flag").ExpectError()
	require.Module(t, "cli").Call("float_flag", true).ExpectError()
}

func TestStringSliceFlag(t *testing.T) {
	res := require.Module(t, "cli").Call("string_slice_flag", require.MAP{
		"name":  "tags",
		"usage": "tags to apply",
		"value": require.ARR{"a", "b"},
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	require.Module(t, "cli").Call("string_slice_flag").ExpectError()
	require.Module(t, "cli").Call("string_slice_flag", 99).ExpectError()
}

// ---------------------------------------------------------------------------
// Command builder tests (via require.Module)
// ---------------------------------------------------------------------------

func TestCommand(t *testing.T) {
	res := require.Module(t, "cli").Call("command", require.MAP{
		"name":  "serve",
		"usage": "start the server",
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	require.Module(t, "cli").Call("command").ExpectError()
	require.Module(t, "cli").Call("command", "bad").ExpectError()
}

// ---------------------------------------------------------------------------
// App builder tests (via require.Module, context.Background)
// ---------------------------------------------------------------------------

func TestNewMinimal(t *testing.T) {
	res := require.Module(t, "cli").Call("new", require.MAP{
		"name":  "testapp",
		"usage": "a test app",
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)

	require.Module(t, "cli").Call("new").ExpectError()
	require.Module(t, "cli").Call("new", "bad").ExpectError()
}

// ---------------------------------------------------------------------------
// Integration tests (via require.Expect — runs full rumo scripts with VM)
// ---------------------------------------------------------------------------

func TestAppRunAction(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	action: func(ctx) {
		result = "action_called"
	}
})

app.run(["test"])
out := result
`, "action_called")
}

func TestAppRunWithStringFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_flag({name: "name", aliases: ["n"], value: "world"})
	],
	action: func(ctx) {
		result = ctx.string("name")
	}
})

app.run(["test", "--name", "rumo"])
out := result
`, "rumo")
}

func TestAppRunWithStringFlagDefault(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_flag({name: "name", value: "default_val"})
	],
	action: func(ctx) {
		result = ctx.string("name")
	}
})

app.run(["test"])
out := result
`, "default_val")
}

func TestAppRunWithBoolFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := false
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.bool_flag({name: "verbose", aliases: ["v"]})
	],
	action: func(ctx) {
		result = ctx.bool("verbose")
	}
})

app.run(["test", "--verbose"])
out := result
`, true)
}

func TestAppRunWithIntFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := 0
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.int_flag({name: "port", aliases: ["p"], value: 3000})
	],
	action: func(ctx) {
		result = ctx.int("port")
	}
})

app.run(["test", "-p", "8080"])
out := result
`, int64(8080))
}

func TestAppRunWithFloatFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := 0.0
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.float_flag({name: "rate", value: 1.0})
	],
	action: func(ctx) {
		result = ctx.float("rate")
	}
})

app.run(["test", "--rate", "2.5"])
out := result
`, 2.5)
}

func TestAppRunWithCommand(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	commands: [
		cli.command({
			name: "greet",
			flags: [
				cli.string_flag({name: "name", value: "world"})
			],
			action: func(ctx) {
				result = "hello " + ctx.string("name")
			}
		})
	]
})

app.run(["test", "greet", "--name", "rumo"])
out := result
`, "hello rumo")
}

func TestAppRunWithInlineCommand(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	commands: [
		{
			name: "ping",
			action: func(ctx) {
				result = "pong"
			}
		}
	]
})

app.run(["test", "ping"])
out := result
`, "pong")
}

func TestContextArgs(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
text := import("text")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	args: true,
	action: func(ctx) {
		result = text.join(ctx.args(), ",")
	}
})

app.run(["test", "foo", "bar", "baz"])
out := result
`, "foo,bar,baz")
}

func TestContextNArg(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := 0
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	args: true,
	action: func(ctx) {
		result = ctx.narg()
	}
})

app.run(["test", "a", "b"])
out := result
`, int64(2))
}

func TestContextIsSet(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

was_set := false
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_flag({name: "foo"}),
		cli.string_flag({name: "bar"})
	],
	action: func(ctx) {
		was_set = ctx.is_set("foo")
	}
})

app.run(["test", "--foo", "x"])
out := was_set
`, true)
}

func TestContextIsSetFalse(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

was_set := true
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_flag({name: "foo"}),
		cli.string_flag({name: "bar"})
	],
	action: func(ctx) {
		was_set = ctx.is_set("bar")
	}
})

app.run(["test", "--foo", "x"])
out := was_set
`, false)
}

func TestBeforeCallback(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

order := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	before: func(ctx) {
		order = order + "before,"
	},
	action: func(ctx) {
		order = order + "action"
	}
})

app.run(["test"])
out := order
`, "before,action")
}

func TestAfterCallback(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

order := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	action: func(ctx) {
		order = order + "action,"
	},
	after: func(ctx) {
		order = order + "after"
	}
})

app.run(["test"])
out := order
`, "action,after")
}

func TestSubcommands(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	commands: [
		cli.command({
			name: "parent",
			commands: [
				cli.command({
					name: "child",
					action: func(ctx) {
						result = "child_action"
					}
				})
			]
		})
	]
})

app.run(["test", "parent", "child"])
out := result
`, "child_action")
}

func TestCommandAliases(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	commands: [
		cli.command({
			name: "serve",
			aliases: ["s"],
			action: func(ctx) {
				result = "served"
			}
		})
	]
})

app.run(["test", "s"])
out := result
`, "served")
}

func TestFlagAlias(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_flag({name: "output", aliases: ["o"]})
	],
	action: func(ctx) {
		result = ctx.string("output")
	}
})

app.run(["test", "-o", "file.txt"])
out := result
`, "file.txt")
}

func TestContextFlagNames(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := 0
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_flag({name: "aaa"}),
		cli.int_flag({name: "bbb"})
	],
	action: func(ctx) {
		result = ctx.num_flags()
	}
})

app.run(["test", "--aaa", "x", "--bbb", "3"])
out := result
`, int64(2))
}

// ---------------------------------------------------------------------------
// Direct Go-level tests for internal helpers
// ---------------------------------------------------------------------------

func TestAppRunReturnsError(t *testing.T) {
	// Calling run with a non-array argument should produce an error
	appRes := require.Module(t, "cli").Call("new", require.MAP{
		"name": "test",
	})
	require.NoError(t, appRes.E)

	runRes := appRes.Call("run", "not-an-array")
	require.Error(t, runRes.E)
}

func TestAppRunWrongArgCount(t *testing.T) {
	appRes := require.Module(t, "cli").Call("new", require.MAP{
		"name": "test",
	})
	require.NoError(t, appRes.E)

	// run with too many args
	runRes := appRes.Call("run", require.ARR{"a"}, require.ARR{"b"})
	require.Error(t, runRes.E)
}

func TestNewWithAuthors(t *testing.T) {
	res := require.Module(t, "cli").Call("new", require.MAP{
		"name": "test",
		"authors": require.ARR{
			require.MAP{"name": "Alice", "email": "alice@example.com"},
			require.MAP{"name": "Bob"},
		},
	})
	require.NoError(t, res.E)
	require.NotNil(t, res.O)
}

func TestNewWithInvalidFlags(t *testing.T) {
	// flags must be an array — passing a string should error
	res := require.Module(t, "cli").Call("new", require.MAP{
		"name":  "test",
		"flags": "bad",
	})
	require.Error(t, res.E)
}

func TestNewWithInvalidCommands(t *testing.T) {
	// commands must be an array — passing a string should error
	res := require.Module(t, "cli").Call("new", require.MAP{
		"name":     "test",
		"commands": "bad",
	})
	require.Error(t, res.E)
}

func TestNewWithInvalidAuthors(t *testing.T) {
	// authors must be an array — passing a string should error
	res := require.Module(t, "cli").Call("new", require.MAP{
		"name":    "test",
		"authors": "bad",
	})
	require.Error(t, res.E)
}

func TestCommandWithFlags(t *testing.T) {
	// Build a flag, then a command containing it
	flagRes := require.Module(t, "cli").Call("string_flag", require.MAP{
		"name": "out",
	})
	require.NoError(t, flagRes.E)

	cmdRes := require.Module(t, "cli").Call("command", require.MAP{
		"name":  "build",
		"flags": require.ARR{flagRes.O.(vm.Object)},
	})
	require.NoError(t, cmdRes.E)
	require.NotNil(t, cmdRes.O)
}

func TestAppRunNoArgs(t *testing.T) {
	// run() without arguments uses os.Args — just make sure it doesn't panic
	// We can't easily test the full behavior here, but we can at least
	// verify the function path by calling with an explicit empty program arg
	appRes := require.Module(t, "cli").Call("new", require.MAP{
		"name":         "test",
		"hide_help":    true,
		"hide_version": true,
		"action": &vm.BuiltinFunction{
			Name: "noop",
			Value: func(ctx context.Context, args ...vm.Object) (vm.Object, error) {
				return vm.UndefinedValue, nil
			},
		},
	})
	require.NoError(t, appRes.E)

	// Call run with a single-element array (just the program name)
	runRes := appRes.Call("run", require.ARR{"test"})
	require.NoError(t, runRes.E)
}

func TestStringSliceFlagInScript(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
text := import("text")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	flags: [
		cli.string_slice_flag({name: "tag", aliases: ["t"]})
	],
	action: func(ctx) {
		result = text.join(ctx.string_slice("tag"), ",")
	}
})

app.run(["test", "--tag", "a", "--tag", "b"])
out := result
`, "a,b")
}

func TestMultipleCommands(t *testing.T) {
	require.Expect(t, `
cli := import("cli")

result := ""
app := cli.new({
	name: "test",
	hide_help: true,
	hide_version: true,
	commands: [
		cli.command({
			name: "alpha",
			action: func(ctx) { result = "alpha" }
		}),
		cli.command({
			name: "beta",
			action: func(ctx) { result = "beta" }
		})
	]
})

app.run(["test", "beta"])
out := result
`, "beta")
}

