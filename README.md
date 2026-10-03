# qrr — Quick Recipe Runner

An interactive Go CLI for running shortcuts defined in YAML.

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
Type keywords to filter names and descriptions, use the arrow keys to navigate,
and select a recipe to proceed to its parameter prompts. Arrow keys move the
highlight within a stable list; the list scrolls only when the selected item moves
outside the visible rows. Long descriptions wrap to fit the terminal width. Ctrl+C cancels. After
execution, qrr returns to the shell. Press Enter once to submit the current selection.
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
qrr remove hello --yes
```

The `add` command creates an editable example template. `edit` uses `$EDITOR`
(with quoted arguments supported), falling back to `vi`. Removal asks for
confirmation unless `--yes` is supplied.

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

The default output is only an entry prompt, for example:

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
uses highlighted text with no background, border, padding, or inserted line
breaks. The terminal handles visual wrapping, keeping the prompt on one logical
line for copying. The copy reminder uses the terminal's default text color.
Redirected output stays plain text; `NO_COLOR` disables styling. Only the default
mode prints a copy reminder to stderr. `qrr schema` prints the configuration path, embedded YAML schema, and
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

See [docs/schema.md](docs/schema.md) and [examples/commands](examples/commands).
Handlers, workflows, shell pipelines, usage-frequency ranking, and favorites are
not implemented. The yt-dlp example is declarative: subtitle network failures
follow yt-dlp's exit behavior. A workflow that preserves successful video downloads
when subtitle downloads fail is not implemented or verified against live YouTube.

## Dependencies

The interactive CLI uses [Huh v2](https://github.com/charmbracelet/huh),
[Bubble Tea v2](https://github.com/charmbracelet/bubbletea),
[Bubbles v2](https://github.com/charmbracelet/bubbles), and
[Lip Gloss v2](https://github.com/charmbracelet/lipgloss). Huh manages selection
and scrolls only enough to keep the active option visible. qrr adds a small
adapter for immediate keyword filtering and empty-result protection.
Dependency versions are pinned in `go.mod` and `go.sum`. The indirect pins also
cover upstream test and tool modules so `go list -m -u all` reports no available
updates. `go mod tidy` can prune unused graph pins; recheck the complete module
graph after changing dependencies.
