# qrr (Quick Recipe Runner)

qrr turns complex commands into interactive recipes, with prompts for values
and menus for options. Save commands you run often, then choose a recipe and
fill in the values that change each time.

[demo1.webm](https://github.com/user-attachments/assets/d2d08d2a-4ed9-48bb-81f6-e150f1108aea)

## Run saved commands

Search for a recipe, answer its prompts, and run it—all from your terminal.

You can also run recipes by name with flags, preview commands with `--dry-run`,
or use `--non-interactive` in scripts.

Selection parameters can load searchable choices from external commands with
`options_command`. The [Docker shell example](examples/commands/docker-shell.yaml)
lists running containers and opens a shell in the one you select.

## Turn an existing command into a recipe

Pass a command to `qrr prompt` and send the generated instructions to your AI
agent. The agent can use qrr's recipe schema to turn that command into a saved
shortcut with prompts and defaults. Recipes are YAML files that you can edit
and share.

## Recipe location

Store recipes in `~/.config/qrr/commands/`, one file per recipe, such as
`hello.yaml`. The recipe's `name` must match the filename. `qrr add hello`
creates the file in this directory.

## Basic usage

After installation, use these commands with your saved recipes:

- `qrr`: search for a recipe and run it interactively; press Ctrl+R to rename or Ctrl+E to edit the highlighted recipe.
- `qrr hello`: run the `hello` recipe and answer its parameter prompts.
- `qrr hello --name Harry --non-interactive`: run with supplied values and defaults without prompts.
- `qrr hello --name Harry --non-interactive --dry-run`: preview the command without executing it.
- `qrr list`: list saved recipes.
- `qrr show hello`: view the recipe definition.
- `qrr add hello`: create a recipe template to edit.
- `qrr edit hello`: open the recipe in your editor.
- `qrr rename hello greet`: rename a recipe and its file.
- `qrr remove hello`: delete the recipe after confirmation.
- `qrr validate`: check recipe files for errors.
- `qrr prompt "flutter upgrade"`: generate instructions to send to an AI agent that can create a recipe.
- `qrr --help`: show available commands and options.

Build a smaller standalone executable with `make release`; use `make build` for
a build with debugging information.

See the [documentation index](docs/README.md) for user and maintainer guides, the
[usage guide](docs/usage.md) for installation and commands, or the
[example recipes](examples/commands).
