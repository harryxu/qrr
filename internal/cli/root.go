// Package cli builds the dynamic qrr command tree.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"qrr/internal/prompt"
	"qrr/internal/recipe"
	"qrr/internal/runner"

	"github.com/spf13/cobra"
)

// ConfigDir resolves the configuration path before dynamic flags are registered.
func ConfigDir(args []string) (string, error) {
	dir, err := recipe.DefaultDir()
	if err != nil {
		return "", err
	}
	promptIndex := promptCommandIndex(args)
	for i := 0; i < len(args); i++ {
		if i == promptIndex {
			break
		}
		if args[i] == "--" {
			break
		}
		if strings.HasPrefix(args[i], "--config-dir=") {
			dir = strings.TrimPrefix(args[i], "--config-dir=")
		} else if args[i] == "--config-dir" && i+1 < len(args) {
			i++
			dir = args[i]
		}
	}
	if dir == "" {
		return "", fmt.Errorf("config-dir must not be empty")
	}
	return dir, nil
}
func New(args []string) (*cobra.Command, error) {
	dir, err := ConfigDir(args)
	if err != nil {
		return nil, err
	}
	recipes, problems := recipe.Load(dir)
	var nonInteractive, dryRun, jsonOutput bool
	root := &cobra.Command{Use: "qrr", Short: "Run your command recipes", SilenceUsage: true, SilenceErrors: true, TraverseChildren: true, Args: cobra.NoArgs}
	root.PersistentFlags().StringVar(&dir, "config-dir", dir, "Configuration directory")
	root.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "Use defaults without prompting")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Render argv without executing")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Print dry-run argv as JSON")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "__complete" || cmd.Name() == "__completeNoDesc" || cmd.Name() == "validate" || cmd.Name() == "prompt" || cmd.Name() == "schema" {
			return
		}
		for _, problem := range problems {
			fmt.Fprintln(cmd.ErrOrStderr(), "Warning:", problem)
		}
	}
	execute := func(cmd *cobra.Command, r *recipe.Recipe, selected bool) error {
		values := map[string]any{}
		interactive := !nonInteractive && prompt.IsTerminal()
		for _, p := range r.Params {
			value := recipe.DefaultValue(p)
			provided := !selected && cmd.Flags().Changed(p.Name)
			if provided {
				switch p.Type {
				case "confirm":
					value, _ = cmd.Flags().GetBool(p.Name)
				case "multiselect":
					s, _ := cmd.Flags().GetString(p.Name)
					if s == "" {
						value = []string{}
					} else {
						value = strings.Split(s, ",")
					}
				default:
					value, _ = cmd.Flags().GetString(p.Name)
				}
			}
			if !provided && interactive {
				var err error
				value, err = prompt.Ask(cmd.Context(), p, value)
				if err != nil {
					return err
				}
			}
			if err := recipe.ValidateValue(p, value); err != nil {
				return err
			}
			values[p.Name] = value
		}
		argv, err := r.Render(values)
		if err != nil {
			return err
		}
		if dryRun {
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(argv)
			}
			fmt.Fprintln(cmd.OutOrStdout(), Display(argv))
			return nil
		}
		if jsonOutput {
			return fmt.Errorf("--json requires --dry-run")
		}
		return runner.Run(cmd.Context(), argv)
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if nonInteractive || !prompt.IsTerminal() {
			return fmt.Errorf("a command name is required outside interactive mode; use qrr list or qrr --help")
		}
		r, err := prompt.Choose(cmd.Context(), recipes)
		if err != nil {
			return err
		}
		return execute(cmd, r, true)
	}
	run := &cobra.Command{Use: "run <name>", Short: "Run a named recipe", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error { return fmt.Errorf("a command name is required") }}
	root.AddCommand(run, newPromptCommand(dir), newSchemaCommand(dir))
	for _, r := range recipes {
		root.AddCommand(recipeCommand(r, execute))
		run.AddCommand(recipeCommand(r, execute))
	}
	addManagement(root, dir, recipes, problems, func() bool { return nonInteractive })
	return root, nil
}
func recipeCommand(r *recipe.Recipe, execute func(*cobra.Command, *recipe.Recipe, bool) error) *cobra.Command {
	cmd := &cobra.Command{Use: r.Name, Short: r.Description, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error { return execute(cmd, r, false) }}
	for _, p := range r.Params {
		value := recipe.DefaultValue(p)
		description := p.Prompt
		if description == "" {
			description = p.Name
		}
		if p.Required {
			description += " (required)"
		}
		switch p.Type {
		case "confirm":
			cmd.Flags().Bool(p.Name, value.(bool), description)
		case "multiselect":
			cmd.Flags().String(p.Name, strings.Join(value.([]string), ","), description+" (comma-separated; empty value clears selection)")
		default:
			cmd.Flags().String(p.Name, value.(string), description)
		}
		if p.Type == "select" || p.Type == "multiselect" {
			param := p
			_ = cmd.RegisterFlagCompletionFunc(p.Name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				prefix := ""
				fragment := toComplete
				used := map[string]bool{}
				if param.Type == "multiselect" {
					if i := strings.LastIndex(toComplete, ","); i >= 0 {
						prefix = toComplete[:i+1]
						fragment = toComplete[i+1:]
						for _, s := range strings.Split(strings.TrimSuffix(prefix, ","), ",") {
							used[s] = true
						}
					}
				}
				var candidates []string
				for _, o := range param.Options {
					if strings.HasPrefix(o.Value, fragment) && !used[o.Value] {
						candidates = append(candidates, prefix+o.Value+"\t"+o.Label)
					}
				}
				return candidates, cobra.ShellCompDirectiveNoFileComp
			})
		} else if p.Type == "input" {
			_ = cmd.RegisterFlagCompletionFunc(p.Name, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
				return nil, cobra.ShellCompDirectiveNoFileComp
			})
		}
	}
	return cmd
}
func Display(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
	}
	return strings.Join(quoted, " ")
}
func Execute(ctx context.Context, args []string, out, stderr io.Writer) error {
	root, err := New(args)
	if err != nil {
		return err
	}
	root.SetArgs(args)
	root.SetOut(out)
	root.SetErr(stderr)
	return root.ExecuteContext(ctx)
}
