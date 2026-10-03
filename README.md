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
and select a recipe to proceed to its parameter prompts. Ctrl+C cancels. After
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
program, with inherited stdin/stdout/stderr and its exit code. Recipe and CLI
errors return 1; cancellation returns 130. No shell expansion is performed.

See [docs/schema.md](docs/schema.md) and [examples/commands](examples/commands).
Handlers, workflows, shell pipelines, usage-frequency ranking, and favorites are
not implemented. The yt-dlp example is declarative: subtitle network failures
follow yt-dlp's exit behavior. A workflow that preserves successful video downloads
when subtitle downloads fail is not implemented or verified against live YouTube.
