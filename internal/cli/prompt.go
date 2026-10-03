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
		Use:                   "prompt [-v] <command> [arguments...]",
		Short:                 "Generate an AI prompt for adding a recipe",
		Long:                  "Generate a short entry prompt to copy to an AI agent.\nUse -v or --verbose before the target command to show detailed authoring instructions.\nVerbose mode accepts a whole command as one quoted argument.\nFlags after the target executable belong to the target command.\nThe target command is not executed. Place qrr global flags before prompt.",
		Example:               "  qrr prompt flutter upgrade\n  qrr prompt -v 'flutter upgrade'\n  qrr prompt yt-dlp --write-subs 'https://www.youtube.com/watch?v=example'",
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose := false
			if len(args) > 0 && (args[0] == "-v" || args[0] == "--verbose") {
				verbose = true
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
			if verbose && len(args) == 1 {
				command = args[0]
			}
			var text string
			if verbose {
				configDir, err := filepath.Abs(dir)
				if err != nil {
					return err
				}
				prefix := "qrr --config-dir " + Display([]string{configDir})
				text = fmt.Sprintf("Analyze this command and add a reusable qrr YAML recipe with suitable interactive parameters:\n\n%s\n\nFirst run qrr --help, %s schema, and qrr add --help to learn the format and authoring workflow. Choose a unique name, preserve the command's behavior, and create the recipe using the configuration path shown by schema. Validate it with %s validate <name>, then inspect its argv with %s <name> --non-interactive --dry-run --json, supplying required parameter values.\n", command, prefix, prefix, prefix)
			} else {
				text = fmt.Sprintf("Run qrr prompt -v %s and follow its instructions to add a qrr YAML recipe.\n", Display([]string{command}))
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Copy the following prompt and send it to your AI agent:"); err != nil {
					return err
				}
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), text)
			return err
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed recipe authoring instructions")
	return cmd
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
