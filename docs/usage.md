# Usage

qrr runs YAML command recipes on Linux and macOS. The executable needs no Node
or Python runtime; programs used by a recipe must be installed separately.

Start with [Quick start](#quick-start), then see [Configuration](#configuration),
[Manage recipes](#manage-recipes), or [Parameters and execution](#parameters-and-execution).
The [recipe schema](schema.md) defines the YAML format.

Run commands using `./bin/qrr` from the project root. Commands using `qrr` assume
the executable is on `PATH`.

## Build and install

Install the latest prebuilt release without a Go toolchain:

```sh
# macOS and Linux (amd64 or arm64):
curl -fsSL https://raw.githubusercontent.com/harryxu/qrr/master/install.sh | sh
```

The installer checks SHA-256 before installing into `~/.local/bin`. Add
`export PATH="$HOME/.local/bin:$PATH"` to your shell profile if needed, then open
a new terminal. To choose another directory:

```sh
curl -fsSL https://raw.githubusercontent.com/harryxu/qrr/master/install.sh | QRR_INSTALL_DIR="$HOME/bin" sh
```

For the Windows amd64 release, run in PowerShell:

```powershell
# Optional: set the destination before running the installer.
# $env:QRR_INSTALL_DIR = "$HOME\bin"
irm https://raw.githubusercontent.com/harryxu/qrr/master/install.ps1 | iex
```

The PowerShell installer checks SHA-256, writes `qrr.exe` to
`%LOCALAPPDATA%\Programs\qrr\bin`, and updates the user PATH without administrator
permissions. Reopen other terminals to pick up the PATH change.

Run `qrr --version` to check the installed version. Rerun either installer to
update to the latest release. You can also download archives and `checksums.txt` from
[GitHub Releases](https://github.com/harryxu/qrr/releases).

To build from source, use the Go version specified in `go.mod` or newer:

```sh
make release
# Equivalent command without Make:
go build -trimpath -ldflags='-s -w' -o bin/qrr ./cmd/qrr
# Or install a release build into your Go binary directory:
go install -trimpath -ldflags='-s -w' ./cmd/qrr
```

Add your Go binary directory to `PATH` when using `go install`.

Release builds strip the symbol table and DWARF debugging information to reduce
executable size, and remove absolute build paths with `-trimpath`. Normal Go
panic stack traces remain available. For debugging with symbols and DWARF, use
`make build` or `go build -o bin/qrr ./cmd/qrr`. Both Make targets write to
`bin/qrr`.

GitHub release builds embed the release tag, so `qrr --version` prints a value
such as `qrr version v1.2.3`. Development builds include the first seven characters
of the Git commit recorded by Go, for example `qrr version dev (abc1234)`.
Without Git build metadata (such as when using `-buildvcs=false`), they report
`qrr version dev (unknown)`. Published versions do not display the commit hash.
To embed a version when building locally, run `make release VERSION=v1.2.3`, or:

```sh
go build -trimpath -ldflags='-s -w -X qrr/internal/cli.version=v1.2.3' -o bin/qrr ./cmd/qrr
```

## Quick start

Preview the included greeting, then run it with the system's `echo` program:

```sh
./bin/qrr --config-dir ./examples list
./bin/qrr --config-dir ./examples hello --name Test --non-interactive --dry-run --json
# Expected stdout: ["echo","Hello, Test!"]
./bin/qrr --config-dir ./examples hello --name Test --non-interactive
```

The dry-run prints argv without running `echo`. The last command executes it and
prints `Hello, Test!`; qrr also prints the command to stderr before execution.

In an interactive terminal, open the recipe selector or prompt for a name:

```sh
./bin/qrr --config-dir ./examples
./bin/qrr --config-dir ./examples hello
```

The examples are loaded only when you select `./examples` as the configuration
directory. They are not installed into your personal configuration automatically.

## Configuration

The configuration directory contains a `commands/` subdirectory:

```text
<config-dir>/
└── commands/
    ├── hello.yaml
    └── myytdl.yaml
```

By default, recipes are stored in `~/.config/qrr/commands/`. If `XDG_CONFIG_HOME`
is set, qrr uses `$XDG_CONFIG_HOME/qrr/commands/` instead.

Use `--config-dir <dir>` to load recipes from `<dir>/commands/`. For example,
`--config-dir ./examples` reads `./examples/commands`; pass the parent directory,
not the `commands/` directory itself. Relative paths are resolved from your
current working directory. Run `qrr schema` to see the resolved configuration
and recipe directories.

Files are reloaded on each invocation, including completion. Invalid recipes are
skipped so valid ones remain usable. Normal commands report warnings;
`qrr validate` reports configuration errors and fails if any are found. Completion
omits invalid recipes without warnings.

## Choose a recipe interactively

With no command, qrr opens a searchable recipe list in an interactive terminal.
Type keywords to filter names and descriptions. Use the arrow keys to navigate
and press Enter to select a recipe and fill in its parameters. The list scrolls
to keep the selection visible, and long descriptions wrap to the terminal width.
Ctrl+C cancels. After execution, qrr returns to the shell.
No recipe can be selected when the filter has no matches. With no terminal or
with `--non-interactive`, a command name is required.

| Key | Action |
| --- | --- |
| Type keywords | Filter recipe names and descriptions |
| Arrow keys | Move through matching recipes |
| Enter | Choose the highlighted recipe, then fill in parameters |
| Ctrl+R | Rename the highlighted recipe |
| Ctrl+E | Edit the highlighted recipe in `$EDITOR` (or `vi`) |
| Ctrl+C | Cancel and return to the shell |

Ctrl+R opens an inline input prefilled with the current name. Enter saves; Esc
returns to the same filtered list without changes. Invalid, reserved, unchanged,
or occupied names show an error and allow correction. After saving, the search
is cleared and the renamed recipe is highlighted; press Enter separately to run
it. Ctrl+C cancels from the rename input too.

Ctrl+E restores the terminal before opening the editor. After it exits, qrr
reloads recipes, clears the search, and retains the highlighted name if it is
still valid. Editor failures and invalid configurations are reported; invalid
recipes are skipped, and an empty list exits with an error. Editing never runs
the recipe. Ctrl+R and Ctrl+E do nothing when no recipes match.

## Manage recipes

Create your first personal recipe, inspect it, and preview its command:

```sh
qrr add hello
qrr edit hello
qrr validate
qrr list
qrr show hello
qrr hello --message "Hello from qrr" --non-interactive --dry-run --json
qrr run hello --message "Hello from qrr" --non-interactive
qrr rename hello greet
qrr remove greet --yes
```

The `add` command creates a template with a `message` parameter, without
overwriting an existing recipe. This differs from the included `hello` example,
which has a `name` parameter. Customize the new file before running it.

| Command | Purpose |
| --- | --- |
| `qrr list` | List valid recipe names and descriptions |
| `qrr show <name>` | Print the file, including invalid YAML |
| `qrr add <name>` | Create a template in the configuration directory |
| `qrr edit <name>` | Open the file in an editor; requires a terminal |
| `qrr rename <old-name> <new-name>` | Rename a valid recipe and its file |
| `qrr remove <name> [--yes]` | Delete the file, including invalid YAML |
| `qrr validate [name]` | Validate all recipes or one named recipe |

`edit` uses `$EDITOR` (with quoted arguments supported), falling back to `vi`. Removal asks for
confirmation unless `--yes` is supplied. In non-interactive mode or without a
terminal, removal requires `--yes`.

### Rename constraints

`qrr rename <old-name> <new-name>` updates the YAML `name` and filename without
prompting or executing the recipe. It preserves the `.yaml` or `.yml` extension,
comments, file permissions, and command behavior; YAML formatting may change.
The new name must be valid and unreserved, and neither target extension may
already exist. Renaming to the same name is an error. The source must be a valid
recipe in a regular file, with no duplicate file under the other extension.
Recipes whose other values depend on an anchored `name` must be edited to remove
that dependency before renaming. The new command and its flags appear in completion
on the next invocation.

## Create recipes with an AI agent

Pass an existing command to `qrr prompt` to generate a short prompt, then copy the
prompt and send it to your AI agent:

```sh
qrr prompt yt-dlp \
  -f "bestvideo[height<=2160]+bestaudio/best[height<=2160]" \
  --write-subs \
  --write-auto-subs \
  --sub-langs "zh-Hans,ja,en" \
  --embed-subs \
  --sleep-subtitles 60 \
  'https://www.youtube.com/watch?v=cD-Z2A2zuiY'
```

For a simpler command, the default output looks like this:

```sh
qrr prompt flutter upgrade
```

```text
Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.
```

The agent runs the suggested command to retrieve detailed instructions:

```sh
qrr prompt -v "flutter upgrade"
qrr prompt --verbose "flutter upgrade"
qrr prompt --help
qrr schema
```

Use `-v` or `--verbose` before the target command to select detailed output. In
verbose mode, pass a complete command as one quoted argument or use separate
command arguments. After the target executable, all flags belong to that command,
including `-v`, `--help`, and flags that overlap qrr flags. Simple command tokens
are displayed naturally; values containing spaces or shell syntax remain quoted
to preserve argument boundaries.

Quote URLs and shell-sensitive values so your shell passes them unchanged.
Default entry prompts omit configuration paths. For a custom configuration,
retrieve detailed instructions with qrr's global flags before `prompt`:

```sh
qrr --config-dir ./recipes prompt -v "flutter upgrade"
```

Both modes generate text only; neither executes the target command nor creates a
recipe. Prompts go to stdout. In a color-capable terminal, the default entry
uses ANSI cyan and stays on one logical line for copying; the terminal handles
visual wrapping. The copy reminder uses the terminal's default text color.
Redirected output stays plain text; `NO_COLOR` disables styling. Only the default
mode prints a copy reminder to stderr when stdout is a terminal. Pipes and file
redirection omit the reminder, including when `NO_COLOR` is set. For example:

```sh
qrr prompt flutter upgrade | cat
qrr prompt -v flutter upgrade > recipe-prompt.txt
```

Use `-v` to send the complete authoring instructions directly to an agent that
reads stdin. Without `-v`, the agent receives instructions to run
`qrr prompt -v` first.

`qrr schema` prints the configuration path, embedded YAML schema, and
adding steps, so agents can learn the format when only the binary is installed.
`prompt` and `schema` are reserved command names.

## Shell completion

After installing qrr on PATH, load completion once for your shell:

```sh
# Zsh, after compinit (add to ~/.zshrc):
source <(qrr completion zsh)

# Bash (add to ~/.bashrc; requires bash-completion):
source <(qrr completion bash)

# Fish:
qrr completion fish > ~/.config/fish/completions/qrr.fish
```

Try `qrr <Tab>`, `qrr run <Tab>`, or `qrr myytdl --resolution <Tab>`.
New YAML recipes and their flags appear without regenerating completion scripts.
To use a custom configuration directory, put `--config-dir` before the command
name being completed. Completion reads configuration only and never executes a
recipe or opens a prompt.

## Parameters and execution

### Flags, defaults, and prompts

| Flag | Effect when running a recipe |
| --- | --- |
| `--config-dir <dir>` | Read recipes from `<dir>/commands` |
| `--non-interactive` | Use supplied values and defaults without qrr prompts or option queries |
| `--dry-run` | Render argv without executing the recipe command |
| `--json` | Print dry-run argv as a JSON array; requires `--dry-run` |

Explicit flags override defaults. In interactive mode, parameters not supplied as
flags are prompted. In non-interactive mode they use defaults, and missing required
values fail. Boolean flags accept `--flag=false`; multi-select flags accept comma
separated values, with `--subtitles=` representing an explicit empty selection.
When stdin or stdout is not a terminal, recipe parameters also use defaults
without prompting. Parameters are processed in YAML declaration order.
`--non-interactive` controls qrr's prompts and option queries; the invoked program
still receives stdin and may have its own interactive behavior.

### Selection menus and dynamic options

Selection parameters can use `options_command` to retrieve their choices from an
external command or an explicitly invoked shell script. The command returns a JSON
array of `{ "label": "...", "value": "..." }` objects. Single-select menus support
direct keyword search; multiselect menus use `/` to filter and Space to toggle.
An empty result, failed query, invalid output, or stale default stops execution
with an error. Queries time out after 30 seconds and support Ctrl+C cancellation.
Explicit flags and non-interactive mode skip queries. Validation and completion
also never run them. Interactive dry-runs may query options before showing argv.

The [Docker shell example](../examples/commands/docker-shell.yaml) queries running
containers, displays their names and images, and prompts for `sh` or `bash` to
open in the selected container. It requires Bash, Docker, and jq on the host,
plus the selected shell inside the container:

```sh
./bin/qrr --config-dir ./examples docker-shell
# Inspect argv without querying Docker or entering a container:
./bin/qrr --config-dir ./examples docker-shell --container example-id --shell sh --non-interactive --dry-run --json
```

### Preview argv

```sh
./bin/qrr --config-dir ./examples myytdl --url 'https://example.com/video' --subtitles= --non-interactive --dry-run --json
```

This uses the configured resolution default and clears the default subtitle
selection, so subtitle arguments are omitted. It does not download anything.

`--dry-run` renders argv without executing; `--dry-run --json` emits a JSON array.
Shell-quoted output is for inspection only. Interactive dry-runs can still run
`options_command`; supply explicit values or use `--non-interactive` to avoid
those queries.

### Streams, cancellation, and exit codes

Execution passes argv directly to the program, with inherited stdin/stdout/stderr and its exit code. Before starting a
recipe, qrr prints the complete rendered command to stderr with a `Running:`
prefix, including all selected parameter values. Recipe and CLI
errors return 1; cancellation returns 130. Ctrl+C is treated as user cancellation:
qrr avoids sending a duplicate interrupt to a child already in the terminal's
foreground group and suppresses subsequent stderr cleanup diagnostics. Normal
stderr output and genuine failures remain visible; stdout stays unchanged.
Terminal stderr is relayed through a pseudo-terminal to retain terminal-aware
progress output and resizing. No shell expansion is performed.

## Troubleshooting

| Symptom | Next step |
| --- | --- |
| No commands configured | Use `qrr add <name>` or choose a configuration directory containing recipes |
| Recipe missing from list or completion | Use the same `--config-dir` with `validate` and `schema`; check the directory and that the YAML name matches the filename |
| A value is required | Supply the parameter flag in scripts, or set a suitable default in YAML |
| Option query fails or a default is stale | Inspect the provider output and update the default; see [Dynamic options](schema.md#dynamic-options) |
| Executable not found | Install the program named by the recipe or provider and check `PATH` |
| JSON output requested without dry-run | Use both `--dry-run --json` |

When using a custom directory, diagnose that directory explicitly:

```sh
qrr --config-dir ./recipes schema
qrr --config-dir ./recipes validate
```

## Current limits

Handlers, workflows, implicit shell pipelines, usage-frequency ranking, and
favorites are not implemented. Explicit shell scripts can be used in
`options_command`. The yt-dlp example is declarative: subtitle network failures
follow yt-dlp's exit behavior. A workflow that preserves successful video downloads
when subtitle downloads fail is not implemented or verified against live YouTube.
