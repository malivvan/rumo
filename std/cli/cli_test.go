package cli_test

import (
	"testing"

	"github.com/malivvan/rumo/vm/require"
)

// ---------------------------------------------------------------------------
// Flag descriptor tests (via require.Module().Call())
// ---------------------------------------------------------------------------

func TestCliStringFlag(t *testing.T) {
	require.Module(t, "cli").Call("string_flag",
		require.MAP{"name": "host", "value": "localhost"},
	).Expect(require.IMAP{
		"__flag_type": "string",
		"name":        "host",
		"value":       "localhost",
	})
}

func TestCliBoolFlag(t *testing.T) {
	require.Module(t, "cli").Call("bool_flag",
		require.MAP{"name": "verbose"},
	).Expect(require.IMAP{
		"__flag_type": "bool",
		"name":        "verbose",
	})
}

func TestCliIntFlag(t *testing.T) {
	require.Module(t, "cli").Call("int_flag",
		require.MAP{"name": "port", "value": 8080},
	).Expect(require.IMAP{
		"__flag_type": "int",
		"name":        "port",
		"value":       8080,
	})
}

func TestCliFloatFlag(t *testing.T) {
	require.Module(t, "cli").Call("float_flag",
		require.MAP{"name": "rate", "value": 0.5},
	).Expect(require.IMAP{
		"__flag_type": "float",
		"name":        "rate",
		"value":       0.5,
	})
}

func TestCliStringSliceFlag(t *testing.T) {
	require.Module(t, "cli").Call("string_slice_flag",
		require.MAP{"name": "tag"},
	).Expect(require.IMAP{
		"__flag_type": "string_slice",
		"name":        "tag",
	})
}

func TestCliCommand(t *testing.T) {
	require.Module(t, "cli").Call("command",
		require.MAP{"name": "serve", "usage": "start server"},
	).Expect(require.IMAP{
		"__is_command": true,
		"name":         "serve",
		"usage":        "start server",
	})
}

// ---------------------------------------------------------------------------
// Flag descriptor — wrong argument type / count
// ---------------------------------------------------------------------------

func TestCliFlagErrors(t *testing.T) {
	require.Module(t, "cli").Call("string_flag").ExpectError()
	require.Module(t, "cli").Call("string_flag", "not-a-map").ExpectError()
	require.Module(t, "cli").Call("bool_flag").ExpectError()
	require.Module(t, "cli").Call("int_flag").ExpectError()
	require.Module(t, "cli").Call("float_flag").ExpectError()
	require.Module(t, "cli").Call("string_slice_flag").ExpectError()
	require.Module(t, "cli").Call("command").ExpectError()
	require.Module(t, "cli").Call("new").ExpectError()
}

// ---------------------------------------------------------------------------
// Integration tests (via require.Expect — runs full VM)
// ---------------------------------------------------------------------------

func TestCliAppAction(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	action: func(ctx) {
		out = "executed"
	}
})
app.run(["test"])
`, "executed")
}

func TestCliAppStringFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "name", aliases: ["n"], value: "world"})
	],
	action: func(ctx) {
		out = ctx.string("name")
	}
})
app.run(["test", "--name", "rumo"])
`, "rumo")
}

func TestCliAppStringFlagDefault(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "name", value: "default_val"})
	],
	action: func(ctx) {
		out = ctx.string("name")
	}
})
app.run(["test"])
`, "default_val")
}

func TestCliAppStringFlagShorthand(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "name", aliases: ["n"], value: "world"})
	],
	action: func(ctx) {
		out = ctx.string("name")
	}
})
app.run(["test", "-n", "short"])
`, "short")
}

func TestCliAppBoolFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := false
app := cli.new({
	name: "test",
	flags: [
		cli.bool_flag({name: "verbose", aliases: ["v"]})
	],
	action: func(ctx) {
		out = ctx.bool("verbose")
	}
})
app.run(["test", "-v"])
`, true)
}

func TestCliAppBoolFlagDefault(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := true
app := cli.new({
	name: "test",
	flags: [
		cli.bool_flag({name: "verbose"})
	],
	action: func(ctx) {
		out = ctx.bool("verbose")
	}
})
app.run(["test"])
`, false)
}

func TestCliAppIntFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := 0
app := cli.new({
	name: "test",
	flags: [
		cli.int_flag({name: "port", aliases: ["p"], value: 8080})
	],
	action: func(ctx) {
		out = ctx.int("port")
	}
})
app.run(["test", "-p", "3000"])
`, int64(3000))
}

func TestCliAppIntFlagDefault(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := 0
app := cli.new({
	name: "test",
	flags: [
		cli.int_flag({name: "port", value: 8080})
	],
	action: func(ctx) {
		out = ctx.int("port")
	}
})
app.run(["test"])
`, int64(8080))
}

func TestCliAppFloatFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := 0.0
app := cli.new({
	name: "test",
	flags: [
		cli.float_flag({name: "rate", value: 0.5})
	],
	action: func(ctx) {
		out = ctx.float("rate")
	}
})
app.run(["test", "--rate", "0.85"])
`, 0.85)
}

func TestCliAppStringSliceFlag(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
fmt := import("fmt")
out := ""
app := cli.new({
	name: "test",
	flags: [
		cli.string_slice_flag({name: "tag", aliases: ["t"]})
	],
	action: func(ctx) {
		tags := ctx.string_slice("tag")
		out = fmt.sprintf("%v", tags)
	}
})
app.run(["test", "-t", "alpha", "-t", "beta"])
`, `["alpha", "beta"]`)
}

func TestCliAppSubcommand(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "tool",
	commands: [
		cli.command({
			name: "greet",
			flags: [
				cli.string_flag({name: "name", value: "world"})
			],
			action: func(ctx) {
				out = "hello " + ctx.string("name")
			}
		})
	]
})
app.run(["tool", "greet", "--name", "rumo"])
`, "hello rumo")
}

func TestCliAppSubcommandAlias(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "tool",
	commands: [
		cli.command({
			name: "greet",
			aliases: ["g"],
			action: func(ctx) {
				out = "greeted"
			}
		})
	]
})
app.run(["tool", "g"])
`, "greeted")
}

func TestCliAppBeforeAfter(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	before: func(ctx) {
		out = out + "before:"
	},
	action: func(ctx) {
		out = out + "action:"
	},
	after: func(ctx) {
		out = out + "after"
	}
})
app.run(["test"])
`, "before:action:after")
}

func TestCliAppBeforeWithSubcommand(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	before: func(ctx) {
		out = out + "root-before:"
	},
	after: func(ctx) {
		out = out + ":root-after"
	},
	commands: [
		cli.command({
			name: "sub",
			action: func(ctx) {
				out = out + "sub-action"
			}
		})
	]
})
app.run(["test", "sub"])
`, "root-before:sub-action:root-after")
}

func TestCliContextArgs(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
fmt := import("fmt")
out := ""
app := cli.new({
	name: "test",
	action: func(ctx) {
		args := ctx.args()
		out = fmt.sprintf("%d:%v", ctx.narg(), args)
	}
})
app.run(["test", "foo", "bar"])
`, `2:["foo", "bar"]`)
}

func TestCliContextIsSet(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
fmt := import("fmt")
out := ""
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "a", value: "default"}),
		cli.string_flag({name: "b", value: "default"})
	],
	action: func(ctx) {
		out = fmt.sprintf("%v:%v", ctx.is_set("a"), ctx.is_set("b"))
	}
})
app.run(["test", "--a", "set"])
`, "true:false")
}

func TestCliContextNumFlags(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := 0
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "a"}),
		cli.string_flag({name: "b"}),
		cli.string_flag({name: "c"})
	],
	action: func(ctx) {
		out = ctx.num_flags()
	}
})
app.run(["test", "--a", "1", "--c", "3"])
`, int64(2))
}

func TestCliContextFlagNames(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := 0
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "alpha"}),
		cli.int_flag({name: "beta"})
	],
	action: func(ctx) {
		names := ctx.flag_names()
		// flag_names includes persistent flags (alpha, beta) plus auto-added help/version etc.
		// Just check that our flags are present.
		count := 0
		for n in names {
			if n == "alpha" || n == "beta" {
				count = count + 1
			}
		}
		out = count
	}
})
app.run(["test"])
`, int64(2))
}

func TestCliAppPersistentFlags(t *testing.T) {
	// Root-level flags should be accessible from subcommands.
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	flags: [
		cli.string_flag({name: "env", value: "prod"})
	],
	commands: [
		cli.command({
			name: "deploy",
			action: func(ctx) {
				out = ctx.string("env")
			}
		})
	]
})
app.run(["test", "--env", "staging", "deploy"])
`, "staging")
}

func TestCliAppMultipleSubcommands(t *testing.T) {
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "tool",
	commands: [
		cli.command({
			name: "build",
			action: func(ctx) { out = "built" }
		}),
		cli.command({
			name: "test",
			action: func(ctx) { out = "tested" }
		}),
		cli.command({
			name: "deploy",
			action: func(ctx) { out = "deployed" }
		})
	]
})
app.run(["tool", "deploy"])
`, "deployed")
}

func TestCliAppNoArgs(t *testing.T) {
	// When no args are given to run, cobra uses os.Args[1:].
	// We pass an explicit empty-ish array so the root action fires.
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	action: func(ctx) {
		out = "root"
	}
})
app.run(["test"])
`, "root")
}

func TestCliAppHideVersion(t *testing.T) {
	// Ensure hide_version doesn't break execution.
	require.Expect(t, `
cli := import("cli")
out := ""
app := cli.new({
	name: "test",
	version: "1.0.0",
	hide_version: true,
	action: func(ctx) {
		out = "ok"
	}
})
app.run(["test"])
`, "ok")
}
