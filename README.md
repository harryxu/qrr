# qrr (Quick Recipe Runner)

qrr turns complex commands into interactive recipes, with prompts for values
and menus for options. Save commands you run often, then choose a recipe and
fill in the values that change each time.

## Run saved commands

Open qrr, search for a recipe, and fill in its parameters. For the included
video-download recipe, you enter a URL, choose a maximum resolution, and select
subtitle languages. qrr runs the command and returns you to the shell.

When you know the recipe name, run it directly and supply values as flags. You
can preview the resulting command before running it or use non-interactive mode
in scripts.

## Turn an existing command into a recipe

Pass a command to `qrr prompt` and send the generated instructions to your AI
agent. The agent can use qrr's recipe schema to turn that command into a saved
shortcut with prompts and defaults. Recipes are YAML files that you can edit
and share.

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

See the [usage guide](docs/usage.md) for installation and commands, or browse the
[example recipes](examples/commands).
