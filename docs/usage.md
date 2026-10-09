# Usage

qrr runs commands saved as YAML recipes. Install any programs your recipes use
separately.

## Install

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/harryxu/qrr/master/install.sh | sh
```

The installer writes qrr to `~/.local/bin`. Add
`export PATH="$HOME/.local/bin:$PATH"` to your shell profile if needed, then open
a new terminal.

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/harryxu/qrr/master/install.ps1 | iex
```

The installer adds qrr to your user PATH. Reopen your terminal after installation.
You can also download qrr from [GitHub Releases](https://github.com/harryxu/qrr/releases).

Run `qrr --version` to check the installed version.

## Quick start

Create a greeting recipe, preview its command, then run it:

```sh
qrr add hello
qrr list
qrr hello --message "Hello, Test!" --non-interactive --dry-run --json
# Expected stdout: ["echo","Hello, Test!"]
qrr hello --message "Hello, Test!" --non-interactive
```

The preview shows the command arguments. The last command prints `Hello, Test!`.

To choose a recipe or answer its prompts interactively:

```sh
qrr
qrr hello
```

`qrr add hello` creates a template with a `message` parameter. Use `qrr edit hello`
to customize it. The [example recipes](../examples/commands) are available to copy
into your recipe directory; they are not installed automatically.

## Upgrade qrr

```sh
qrr upgrade
```

qrr checks GitHub for the latest stable release and updates itself if a newer
version is available. A development version (`dev`) is replaced with the latest
stable release. You need write access to the directory containing qrr.

Press Ctrl+C to cancel the check or download. This command does not support
`--dry-run` or `--json`.

## Configuration

By default, recipes are stored in `~/.config/qrr/commands/`:

```text
~/.config/qrr/
└── commands/
    ├── hello.yaml
    └── myytdl.yaml
```

If `XDG_CONFIG_HOME` is set, qrr uses `$XDG_CONFIG_HOME/qrr/commands/` instead.
Run `qrr schema` to see the resolved configuration and recipe directories.

Save one YAML file per recipe, with its `name` matching the filename. See the
[recipe schema](schema.md) for the format. Changes take effect the next time you
run qrr. Use `qrr validate` to check for errors; invalid recipes are skipped.

## Choose a recipe interactively

Run `qrr` in a terminal to search your recipes. After you select one, qrr asks
for its parameters, runs the command, and returns to the shell.

| Key | Action |
| --- | --- |
| Type keywords | Filter recipe names and descriptions |
| Arrow keys | Move through matching recipes |
| Enter | Choose the highlighted recipe, then fill in parameters |
| Ctrl+R | Rename the highlighted recipe |
| Ctrl+E | Edit the highlighted recipe in `$EDITOR` (or `vi`) |
| Ctrl+C | Cancel and return to the shell |

When renaming, Enter saves and Esc cancels. Renaming or editing a recipe does
not run it; press Enter in the list when you are ready to execute it.

## Manage recipes

| Command | Purpose |
| --- | --- |
| `qrr list` | List valid recipe names and descriptions |
| `qrr show <name>` | Print the file, including invalid YAML |
| `qrr add <name>` | Create a template in the configuration directory |
| `qrr edit <name>` | Open the file in an editor; requires a terminal |
| `qrr rename <old-name> <new-name>` | Rename a valid recipe and its file |
| `qrr remove <name> [--yes]` | Delete the file, including invalid YAML |
| `qrr validate [name]` | Validate all recipes or one named recipe |

`add` creates a template without overwriting existing files. `edit` uses `$EDITOR`
or falls back to `vi`. Removal asks for confirmation; use `--yes` in scripts.

### Rename constraints

Renaming updates both the recipe's `name` and its filename. Choose a new, unused
name that follows the [naming rules](schema.md#file-layout-and-names). The source
recipe must pass validation.

## Create recipes

### With an AI agent

Pass a command to `qrr prompt`:

```sh
qrr prompt flutter upgrade
```

Copy the printed instruction to your AI agent:

```text
Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.
```

To get the full recipe-writing instructions yourself:

```sh
qrr prompt -v "flutter upgrade"
```

Place `-v` (or `--verbose`) before the target command. Flags after the target
executable belong to that command. Quote URLs and shell-sensitive values.

`qrr prompt` only generates instructions. Your agent creates the recipe; qrr
does not execute the supplied command or save a recipe itself.

To save the full instructions to a file:

```sh
qrr prompt -v flutter upgrade > recipe-prompt.txt
```

### With an AI chat

Paste the output into your AI chat:

```sh
qrr prompt --chat "flutter upgrade"
# Separate target arguments also work:
qrr prompt --chat yt-dlp --write-subs 'https://example.com/video'
```

The AI reads the docs and examples in the [qrr GitHub repository](https://github.com/harryxu/qrr)
and returns a complete YAML recipe with a filename, parameter and dependency
notes, and local save, validation, and dry-run steps. Save the YAML in your
recipe directory and run the suggested checks.

### Manually

Create a YAML file in `~/.config/qrr/commands/`, creating the directory if needed.
Use one file per recipe, with its `name` matching the filename. If you use
`XDG_CONFIG_HOME`, run `qrr schema` to check your recipe directory.

For example, save this as `~/.config/qrr/commands/greet.yaml`:

```yaml
version: 1
name: greet
type: command
command:
  - echo
  - Hello from qrr
```

Check the file and run the recipe:

```sh
qrr validate greet
qrr greet
```

This prints `Hello from qrr`. See the [recipe schema](schema.md) for the complete
file format, including parameters, prompts, defaults, and optional arguments.
You can also view the format with `qrr schema`.

## Shell completion

Add completion to your shell:

```sh
# Zsh, after compinit (add to ~/.zshrc):
source <(qrr completion zsh)

# Bash (add to ~/.bashrc; requires bash-completion):
source <(qrr completion bash)

# Fish:
qrr completion fish > ~/.config/fish/completions/qrr.fish
```

Try `qrr <Tab>` to complete recipe names, or `qrr hello --<Tab>` to complete flags.
New YAML recipes and their flags appear without regenerating completion scripts.

## Parameters and execution

### Flags, defaults, and prompts

| Flag | Effect when running a recipe |
| --- | --- |
| `--config-dir <dir>` | Load recipes from `<dir>/commands/` instead of the default directory. Pass the parent of `commands/`; relative paths start at your current directory. Place this flag before the command when using completion or `prompt`. |
| `--non-interactive` | Use supplied values and defaults without qrr prompts or option queries |
| `--dry-run` | Preview the command without running it |
| `--json` | Show preview arguments as a JSON array; requires `--dry-run` |

Flags override recipe defaults. In a terminal, qrr prompts for values you have
not supplied. Use `--non-interactive` in scripts: omitted values use defaults,
and missing required values cause an error. Specify the recipe name in scripts.
The program run by a recipe may still ask for input.

Use `--flag=false` to turn off a boolean parameter. Separate multiple selections
with commas, or use `--subtitles=` for an empty selection.

### Selection menus and dynamic options

Type to search a single-select menu. In a multiselect menu, press `/` to filter
and Space to toggle an option.

Recipes can load current choices, such as running Docker containers, using
[dynamic options](schema.md#dynamic-options). Supplying the parameter flag or
using `--non-interactive` skips the option query.

Save the [Docker shell example](../examples/commands/docker-shell.yaml) as
`docker-shell.yaml` in your recipe directory. It queries running
containers, displays their names and images, and prompts for `sh` or `bash` to
open in the selected container. It requires Bash, Docker, and jq on the host,
plus the selected shell inside the container:

```sh
qrr docker-shell
# Preview without querying Docker or entering a container:
qrr docker-shell --container example-id --shell sh --non-interactive --dry-run --json
```

### Preview a command

Save the [yt-dlp example](../examples/commands/myytdl.yaml) as `myytdl.yaml` in
your recipe directory, then preview its command:

```sh
qrr myytdl --url 'https://example.com/video' --subtitles= --non-interactive --dry-run --json
```

This uses the configured resolution default and clears the default subtitle
selection, so subtitle arguments are omitted. It does not download anything.

Interactive previews can still load dynamic options. Use `--non-interactive` to
preview without querying them.

### Cancel a command

Press Ctrl+C to cancel a prompt or running command. In scripts, qrr returns the
command's exit code; qrr errors return 1 and cancellation returns 130.

## Troubleshooting

| Symptom | Next step |
| --- | --- |
| No commands configured | Use `qrr add <name>` or copy recipes into your recipe directory |
| Recipe missing from list or completion | Run `qrr schema` to check the recipe directory, then `qrr validate`; check that the YAML name matches the filename |
| A value is required | Supply the parameter flag in scripts, or set a suitable default in YAML |
| Option query fails or a default is stale | Inspect the provider output and update the default; see [Dynamic options](schema.md#dynamic-options) |
| Executable not found | Install the program named by the recipe or provider and check `PATH` |
| JSON output requested without dry-run | Use both `--dry-run --json` |

## Current limits

qrr supports command recipes. Workflows, handlers, favorites, and usage-based
ranking are not available. The yt-dlp example follows yt-dlp's behavior and does
not add recovery for subtitle download failures.
