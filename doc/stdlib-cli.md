---
title: Standard Library - cli
---

```golang
cli := import("cli")
```

## Functions

- `new(config map) => app`: Creates a new CLI application from the given
  configuration map and returns an app object.
- `command(config map) => command`: Creates a command configuration that can be
  passed to the `commands` array of an app or parent command.
- `string_flag(config map) => flag`: Creates a string flag configuration.
- `bool_flag(config map) => flag`: Creates a bool flag configuration.
- `int_flag(config map) => flag`: Creates an int flag configuration.
- `float_flag(config map) => flag`: Creates a float flag configuration.
- `string_slice_flag(config map) => flag`: Creates a string slice flag
  configuration (allows multiple values).

## App Configuration

The `new` function accepts a map with the following keys:

| Key                        | Type     | Description                              |
|----------------------------|----------|------------------------------------------|
| `name`                     | string   | Application name                         |
| `usage`                    | string   | Short usage description                  |
| `usage_text`               | string   | Override for the usage text              |
| `version`                  | string   | Application version                      |
| `description`              | string   | Longer description                       |
| `args_usage`               | string   | Describes positional arguments           |
| `copyright`                | string   | Copyright text                           |
| `default_command`          | string   | Default command name                     |
| `hide_help`                | bool     | Hide the built-in help flag              |
| `hide_help_command`        | bool     | Hide the built-in help command           |
| `hide_version`             | bool     | Hide the built-in version flag           |
| `enable_bash_completion`   | bool     | Enable bash completion                   |
| `use_short_option_handling` | bool    | Enable short option combining            |
| `suggest`                  | bool     | Enable command suggestions               |
| `skip_flag_parsing`        | bool     | Skip flag parsing                        |
| `args`                     | bool     | Allow positional arguments               |
| `flags`                    | array    | Array of flag objects                    |
| `commands`                 | array    | Array of command objects or config maps  |
| `authors`                  | array    | Array of author maps (`{name, email}`)   |
| `action`                   | func     | Action callback `func(ctx)`              |
| `before`                   | func     | Before callback `func(ctx)`              |
| `after`                    | func     | After callback `func(ctx)`               |

## Command Configuration

The `command` function (and inline command maps) accept:

| Key                        | Type     | Description                              |
|----------------------------|----------|------------------------------------------|
| `name`                     | string   | Command name                             |
| `aliases`                  | array    | Alternative names for the command        |
| `usage`                    | string   | Short usage description                  |
| `usage_text`               | string   | Override for the usage text              |
| `description`              | string   | Longer description                       |
| `args_usage`               | string   | Describes positional arguments           |
| `category`                 | string   | Category for grouping                    |
| `hidden`                   | bool     | Hide from help output                    |
| `hide_help`                | bool     | Hide the built-in help flag              |
| `hide_help_command`        | bool     | Hide the built-in help command           |
| `skip_flag_parsing`        | bool     | Skip flag parsing                        |
| `use_short_option_handling` | bool    | Enable short option combining            |
| `args`                     | bool     | Allow positional arguments               |
| `flags`                    | array    | Array of flag objects                    |
| `commands`                 | array    | Array of sub-command objects             |
| `action`                   | func     | Action callback `func(ctx)`              |
| `before`                   | func     | Before callback `func(ctx)`              |
| `after`                    | func     | After callback `func(ctx)`               |

## Flag Configuration

All flag builders accept a map with the following common keys:

| Key        | Type     | Description                                     |
|------------|----------|-------------------------------------------------|
| `name`     | string   | Flag name (used as `--name`)                    |
| `aliases`  | array    | Short or alternative names (e.g. `["n"]` for `-n`) |
| `usage`    | string   | Help text for the flag                          |
| `env_vars` | array    | Environment variable names to read from         |
| `value`    | varies   | Default value (type depends on the flag type)   |
| `required` | bool     | Whether the flag is required                    |
| `hidden`   | bool     | Hide from help output                           |

## App Object

The app object returned by `new` has the following methods:

- `run(args array)`: Runs the app with the given argument array. If no arguments
  are provided, `os.Args` is used.

## Context Object

The context object passed to `action`, `before`, and `after` callbacks provides:

- `args() => array`: Returns positional arguments as a string array.
- `narg() => int`: Returns the number of positional arguments.
- `string(name) => string`: Returns the string value of the named flag.
- `bool(name) => bool`: Returns the bool value of the named flag.
- `int(name) => int`: Returns the int value of the named flag.
- `float(name) => float`: Returns the float value of the named flag.
- `string_slice(name) => array`: Returns the string slice value of the named flag.
- `is_set(name) => bool`: Returns true if the named flag was explicitly set.
- `count(name) => int`: Returns how many times the flag was set.
- `num_flags() => int`: Returns the number of flags that were set.
- `flag_names() => array`: Returns the names of all defined flags.

## Examples

### Simple app with flags

```golang
cli := import("cli")
fmt := import("fmt")

app := cli.new({
    name: "greet",
    usage: "a greeting application",
    flags: [
        cli.string_flag({name: "name", aliases: ["n"], value: "world"}),
        cli.bool_flag({name: "loud", aliases: ["l"]})
    ],
    action: func(ctx) {
        greeting := "Hello, " + ctx.string("name") + "!"
        if ctx.bool("loud") {
            greeting = text.to_upper(greeting)
        }
        fmt.println(greeting)
    }
})

app.run(["greet", "--name", "rumo", "--loud"])
```

### App with subcommands

```golang
cli := import("cli")
fmt := import("fmt")

app := cli.new({
    name: "tool",
    usage: "a multi-command tool",
    commands: [
        cli.command({
            name: "serve",
            aliases: ["s"],
            usage: "start the server",
            flags: [
                cli.int_flag({name: "port", aliases: ["p"], value: 8080})
            ],
            action: func(ctx) {
                fmt.println("serving on port", ctx.int("port"))
            }
        }),
        cli.command({
            name: "build",
            usage: "build the project",
            action: func(ctx) {
                fmt.println("building...")
            }
        })
    ]
})

app.run(["tool", "serve", "-p", "3000"])
```

### Before and after hooks

```golang
cli := import("cli")
fmt := import("fmt")

app := cli.new({
    name: "app",
    before: func(ctx) {
        fmt.println("initializing...")
    },
    action: func(ctx) {
        fmt.println("running action")
    },
    after: func(ctx) {
        fmt.println("cleaning up...")
    }
})

app.run(["app"])
// Output:
// initializing...
// running action
// cleaning up...
```

