# Usage

Run the commands below from the project root when using `./bin/qrr`.

## Build and install

Use the Go version specified in `go.mod` or newer:

```sh
go build -o bin/qrr ./cmd/qrr
# Or install into your Go binary directory:
go install ./cmd/qrr
```

Add your Go binary directory to `PATH` when using `go install`.

## Quick start

```sh
./bin/qrr --config-dir ./examples
./bin/qrr --config-dir ./examples hello
./bin/qrr --config-dir ./examples hello --name Harry --non-interactive
./bin/qrr --config-dir ./examples myytdl --url 'https://example.com/video' --dry-run --json --non-interactive
```

With no command, qrr opens a searchable recipe list in an interactive terminal.
Type keywords to filter names and descriptions. Use the arrow keys to navigate
and press Enter to select a recipe and fill in its parameters. The list scrolls
to keep the selection visible, and long descriptions wrap to the terminal width.
Ctrl+C cancels. After execution, qrr returns to the shell.
Press Ctrl+R to rename the highlighted recipe in an inline input prefilled with
its current name. Enter saves; Esc discards the edit and returns to the same
filtered list. Invalid, reserved, unchanged, or occupied names show an error and
allow correction. After saving, the search is cleared and the renamed recipe is
highlighted; press Enter separately to run it. Ctrl+R does nothing when no recipes
match the search. Ctrl+C cancels from either the list or the rename input.
Press Ctrl+E to open the highlighted recipe's configuration file in `$EDITOR`
(or `vi` when unset). The terminal is restored before the editor starts. After the
editor exits, qrr reloads the recipes and returns to the list with the search
cleared, retaining the highlighted name when it is still valid. Editor failures
and invalid configurations are reported; invalid recipes are skipped, and an
empty list exits with an error. Editing never runs the recipe. Ctrl+E does
nothing when the search has no matches.
With no terminal or with `--non-interactive`, a command name is required.

Configuration is loaded from `$XDG_CONFIG_HOME/qrr/commands`, or
`~/.config/qrr/commands` when XDG_CONFIG_HOME is unset. Use `--config-dir` to override
the qrr directory. Invalid files are skipped with warnings; `qrr validate` reports
configuration errors. Files are reloaded on each invocation, including completion.

```sh
qrr add hello
qrr edit hello
qrr validate
qrr list
qrr show hello
qrr hello --name Harry
qrr run hello --name Harry
qrr rename hello greet
qrr remove greet --yes
```

The `add` command creates an editable example template. `edit` uses `$EDITOR`
(with quoted arguments supported), falling back to `vi`. Removal asks for
confirmation unless `--yes` is supplied.

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

The default output tells the agent which command to run:

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
uses the terminal theme's ANSI cyan color with no background, border, padding, or inserted line
breaks. The terminal handles visual wrapping, keeping the prompt on one logical
line for copying. The copy reminder uses the terminal's default text color.
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

Explicit flags override defaults. In interactive mode, parameters not supplied as
flags are prompted. In non-interactive mode they use defaults, and missing required
values fail. Boolean flags accept `--flag=false`; multi-select flags accept comma
separated values, with `--subtitles=` representing an explicit empty selection.

Selection parameters can use `options_command` to retrieve their choices from an
external command or an explicitly invoked shell script. The command returns a JSON
array of `{ "label": "...", "value": "..." }` objects. Single-select menus support
direct keyword search; multiselect menus use `/` to filter and Space to toggle.
An empty result, failed query, invalid output, or stale default stops execution
with an error. Queries time out after 30 seconds and support Ctrl+C cancellation.
Explicit flags and non-interactive mode skip queries. Validation and completion
also never run them. Interactive dry-runs may query options before showing argv.

The [Docker shell example](../examples/commands/docker-shell.yaml) queries running
containers, displays their names and images, and opens `sh` in the selected
container. It requires Bash, Docker, jq, and `sh` inside the container:

```sh
./bin/qrr --config-dir ./examples docker-shell
# Inspect argv without querying Docker or entering a container:
./bin/qrr --config-dir ./examples docker-shell --container example-id --non-interactive --dry-run --json
```

`--dry-run` renders argv without executing; `--dry-run --json` emits a JSON array.
Shell-quoted output is for inspection only. Execution passes argv directly to the
program, with inherited stdin/stdout/stderr and its exit code. Before starting a
recipe, qrr prints the complete rendered command to stderr with a `Running:`
prefix, including all selected parameter values. Recipe and CLI
errors return 1; cancellation returns 130. Ctrl+C is treated as user cancellation:
qrr avoids sending a duplicate interrupt to a child already in the terminal's
foreground group and suppresses subsequent stderr cleanup diagnostics. Normal
stderr output and genuine failures remain visible; stdout stays unchanged.
Terminal stderr is relayed through a pseudo-terminal to retain terminal-aware
progress output and resizing. No shell expansion is performed.

See the [recipe schema](schema.md) and [example recipes](../examples/commands).
Handlers, workflows, implicit shell pipelines, usage-frequency ranking, and
favorites are not implemented. Explicit shell scripts can be used in
`options_command`. The yt-dlp example is declarative: subtitle network failures
follow yt-dlp's exit behavior. A workflow that preserves successful video downloads
when subtitle downloads fail is not implemented or verified against live YouTube.
