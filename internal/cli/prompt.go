package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"qrr/docs"

	"github.com/spf13/cobra"
)

func newPromptCommand(dir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "prompt [-v] [--chat] <command> [arguments...]",
		Short:                 "Generate an AI prompt for adding a recipe",
		Long:                  "Generate a short entry prompt to copy to an AI agent.\nUse -v or --verbose before the target command to show detailed local authoring instructions.\nUse --chat to generate complete instructions for an AI chat to read GitHub docs and return recipe YAML.\nVerbose and chat modes accept a whole command as one quoted argument.\nFlags after the target executable belong to the target command.\nThe target command is not executed. Place qrr global flags before prompt.",
		Example:               "  qrr prompt flutter upgrade\n  qrr prompt -v 'flutter upgrade'\n  qrr prompt --chat 'flutter upgrade'\n  qrr prompt yt-dlp --write-subs 'https://www.youtube.com/watch?v=example'",
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose := false
			chat := false
		options:
			for len(args) > 0 {
				switch args[0] {
				case "-v", "--verbose":
					verbose = true
				case "--chat":
					chat = true
				default:
					break options
				}
				args = args[1:]
			}
			if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
				return cmd.Help()
			}
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}
			if len(args) == 0 || strings.TrimSpace(args[0]) == "" || strings.HasPrefix(args[0], "-") {
				return fmt.Errorf("a command is required; use qrr prompt --help")
			}
			command := commandText(args)
			if (verbose || chat) && len(args) == 1 {
				command = args[0]
			}
			var text string
			if verbose || chat {
				configDir, err := filepath.Abs(dir)
				if err != nil {
					return err
				}
				prefix := "qrr --config-dir " + Display([]string{configDir})
				if chat {
					text = chatRecipePrompt(command, prefix)
				} else {
					text = fmt.Sprintf("Analyze this command and add a reusable qrr YAML recipe with suitable interactive parameters:\n\n%s\n\nFirst run qrr --help, %s schema, and qrr add --help to learn the format and authoring workflow. Choose a unique name, preserve the command's behavior, and create the recipe using the configuration path shown by schema. Validate it with %s validate <name>, then inspect its argv with %s <name> --non-interactive --dry-run --json, supplying required parameter values.\n", command, prefix, prefix, prefix)
				}
			} else {
				text = fmt.Sprintf("Run qrr prompt -v %s and follow its instructions to add a qrr YAML recipe.\n", Display([]string{command}))
				return writeEntryPrompt(cmd.OutOrStdout(), cmd.ErrOrStderr(), text)
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), text)
			return err
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed recipe authoring instructions")
	cmd.Flags().Bool("chat", false, "Generate instructions for an AI chat using GitHub documentation")
	return cmd
}

func chatRecipePrompt(command, prefix string) string {
	return fmt.Sprintf(`Analyze this command and write a reusable qrr YAML recipe with suitable interactive parameters:

%s

First browse the qrr GitHub repository at https://github.com/harryxu/qrr and read these sources:
- README: https://github.com/harryxu/qrr/blob/master/README.md
- Implemented recipe schema: https://github.com/harryxu/qrr/blob/master/docs/schema.md
- Usage and authoring workflow: https://github.com/harryxu/qrr/blob/master/docs/usage.md
- Recipe examples: https://github.com/harryxu/qrr/tree/master/examples/commands

Use docs/schema.md as the configuration contract; design plans may describe features that are not implemented. If you cannot access these sources, ask me to paste the relevant documents before writing YAML.

Return the complete recipe in a YAML code block with its suggested filename. Use version: 1 and type: command, and choose a valid name matching the filename. Preserve the command's behavior and argv boundaries. Declare suitable parameter types, prompts, defaults, options, and required values. Follow the documented command, optional_args, and args_tail ordering. Do not interpolate user input into shell source or include credentials. Explain the parameters, assumptions, and required external programs briefly.

I will save the file locally. Tell me to run %s schema to find my recipe directory, save the YAML as <name>.yaml there without overwriting an existing recipe, and run %s validate <name>. Provide a concrete %s <name> --non-interactive --dry-run --json example with all required parameter flags so I can inspect the argv without executing the target command. Do not claim to have written files or run local validation; return the YAML and steps in this chat.
`, command, prefix, prefix, prefix)
}

var plainCommandToken = regexp.MustCompile(`^[a-zA-Z0-9_./:@%+=,-]+$`)

// commandText shows simple tokens naturally while preserving argv boundaries.
func commandText(args []string) string {
	tokens := make([]string, len(args))
	for i, arg := range args {
		if plainCommandToken.MatchString(arg) {
			tokens[i] = arg
		} else {
			tokens[i] = Display([]string{arg})
		}
	}
	return strings.Join(tokens, " ")
}

func newSchemaCommand(dir string) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Show the recipe format and how to add a command",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			prefix := "qrr --config-dir " + Display([]string{configDir})
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Configuration directory: %s\nRecipe directory: %s\n\n%s\n## Adding a recipe\n\n1. Choose a unique name and run `%s add <name>` to create a template.\n2. Edit the created file using the schema above; do not overwrite existing recipes.\n3. Run `%s validate <name>`.\n4. Inspect argv with `%s <name> --non-interactive --dry-run --json`, supplying required parameter flags.\n", configDir, filepath.Join(configDir, "commands"), docs.Schema, prefix, prefix, prefix)
			return err
		},
	}
}

// promptCommandIndex identifies the pass-through boundary before reading config flags.
func promptCommandIndex(args []string) int {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--config-dir" {
			i++
			continue
		}
		if arg == "__complete" || arg == "__completeNoDesc" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if arg == "prompt" {
			return i
		}
		return -1
	}
	return -1
}
