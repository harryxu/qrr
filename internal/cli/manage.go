package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"qrr/internal/prompt"
	"qrr/internal/recipe"
	"qrr/internal/runner"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

func addManagement(root *cobra.Command, store *recipe.Store, recipes []*recipe.Recipe, problems []error, nonInteractive func() bool) {
	complete := func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		var result []string
		if len(args) == 0 {
			for _, r := range recipes {
				if strings.HasPrefix(r.Name, prefix) {
					result = append(result, r.Name+"\t"+r.Description)
				}
			}
		}
		return result, cobra.ShellCompDirectiveNoFileComp
	}
	root.AddCommand(&cobra.Command{Use: "list", Short: "List configured recipes", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, args []string) {
		for _, r := range recipes {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", r.Name, r.Description)
		}
	}})
	root.AddCommand(&cobra.Command{Use: "show <name>", Short: "Show a recipe definition", Args: cobra.ExactArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		data, err := store.Read(args[0])
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "validate [name]", Short: "Validate recipe configuration without running commands", Args: cobra.MaximumNArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			r, err := store.Validate(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Valid:", r.Name)
			return nil
		}
		if len(problems) > 0 {
			for _, err := range problems {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
			}
			return fmt.Errorf("%d invalid configuration(s)", len(problems))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Validated %d command(s)\n", len(recipes))
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "add <name>", Short: "Create a recipe template", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		path, err := store.Create(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Created:", path)
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "edit <name>", Short: "Edit a recipe using $EDITOR", Args: cobra.ExactArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		if !prompt.IsTerminal() {
			return fmt.Errorf("editing requires an interactive terminal")
		}
		path, err := store.Path(args[0])
		if err != nil {
			return err
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		argv, err := shlex.Split(editor)
		if err != nil || len(argv) == 0 {
			return fmt.Errorf("invalid EDITOR value")
		}
		if _, err := exec.LookPath(argv[0]); err != nil {
			return err
		}
		return runner.Run(cmd.Context(), append(argv, path))
	}})
	var yes bool
	remove := &cobra.Command{Use: "remove <name>", Short: "Delete a recipe", Args: cobra.ExactArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		_, err := store.Path(name)
		if err != nil {
			return err
		}
		if !yes {
			if nonInteractive() || !prompt.IsTerminal() {
				return fmt.Errorf("use --yes to confirm removal in non-interactive mode")
			}
			value, err := prompt.Ask(cmd.Context(), recipe.Param{Name: "remove", Type: "confirm", Prompt: "Remove " + name + "?"}, false)
			if err != nil {
				return err
			}
			if !value.(bool) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return nil
			}
		}
		if err := store.Remove(name); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Removed:", name)
		return nil
	}}
	remove.Flags().BoolVar(&yes, "yes", false, "Confirm deletion without prompting")
	root.AddCommand(remove)
}
