package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qrr/internal/prompt"
	"qrr/internal/recipe"
	"qrr/internal/runner"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

func addManagement(root *cobra.Command, dir string, recipes []*recipe.Recipe, problems []error, nonInteractive func() bool) {
	// Management commands can inspect and repair invalid recipes as well.
	find := func(name string) (*recipe.Recipe, error) {
		if !recipe.ValidName(name) {
			return nil, fmt.Errorf("invalid command name %q", name)
		}
		for _, ext := range []string{".yaml", ".yml"} {
			path := filepath.Join(dir, "commands", name+ext)
			if _, err := os.Stat(path); err == nil {
				return &recipe.Recipe{Name: name, Path: path}, nil
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
		return nil, fmt.Errorf("unknown command %q", name)
	}
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
		r, err := find(args[0])
		if err != nil {
			return err
		}
		data, err := os.ReadFile(r.Path)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "validate [name]", Short: "Validate recipe configuration without running commands", Args: cobra.MaximumNArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			for _, ext := range []string{".yaml", ".yml"} {
				path := filepath.Join(dir, "commands", args[0]+ext)
				if !recipe.ValidName(args[0]) {
					return fmt.Errorf("invalid command name")
				}
				data, err := os.ReadFile(path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return err
				}
				r, err := recipe.Parse(data)
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				if r.Name != args[0] {
					return fmt.Errorf("name must match filename")
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Valid:", r.Name)
				return nil
			}
			return fmt.Errorf("unknown command %q", args[0])
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
		name := args[0]
		if !recipe.ValidName(name) {
			return fmt.Errorf("invalid or reserved command name %q", name)
		}
		commandsDir := filepath.Join(dir, "commands")
		if err := os.MkdirAll(commandsDir, 0700); err != nil {
			return err
		}
		for _, ext := range []string{".yaml", ".yml"} {
			if _, err := os.Lstat(filepath.Join(commandsDir, name+ext)); err == nil {
				return fmt.Errorf("command %q already exists", name)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		path := filepath.Join(commandsDir, name+".yaml")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := fmt.Fprintf(f, "version: 1\nname: %s\ndescription: An example command\ntype: command\nparams:\n  - name: message\n    type: input\n    prompt: Message\n    default: Hello from qrr\ncommand:\n  - echo\n  - '{{ .message }}'\n", name)
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Created:", path)
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "edit <name>", Short: "Edit a recipe using $EDITOR", Args: cobra.ExactArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		if !prompt.IsTerminal() {
			return fmt.Errorf("editing requires an interactive terminal")
		}
		r, err := find(args[0])
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
		return runner.Run(cmd.Context(), append(argv, r.Path))
	}})
	var yes bool
	remove := &cobra.Command{Use: "remove <name>", Short: "Delete a recipe", Args: cobra.ExactArgs(1), ValidArgsFunction: complete, RunE: func(cmd *cobra.Command, args []string) error {
		r, err := find(args[0])
		if err != nil {
			return err
		}
		if !yes {
			if nonInteractive() || !prompt.IsTerminal() {
				return fmt.Errorf("use --yes to confirm removal in non-interactive mode")
			}
			value, err := prompt.Ask(cmd.Context(), recipe.Param{Name: "remove", Type: "confirm", Prompt: "Remove " + r.Name + "?"}, false)
			if err != nil {
				return err
			}
			if !value.(bool) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return nil
			}
		}
		if err := os.Remove(r.Path); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Removed:", r.Name)
		return nil
	}}
	remove.Flags().BoolVar(&yes, "yes", false, "Confirm deletion without prompting")
	root.AddCommand(remove)
}
