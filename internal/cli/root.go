// Package cli builds the dynamic qrr command tree.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"qrr/internal/prompt"
	"qrr/internal/recipe"
	"qrr/internal/runner"

	"github.com/spf13/cobra"
)

// version is set from the release tag with the Go linker's -X option.
var version = "dev"

func versionString(buildInfo *debug.BuildInfo) string {
	if version != "dev" {
		return version
	}
	if buildInfo != nil {
		for _, setting := range buildInfo.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				return "dev (" + setting.Value[:min(7, len(setting.Value))] + ")"
			}
		}
	}
	return "dev (unknown)"
}

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
	store := recipe.NewStore(dir)
	recipes, problems := store.Load()
	var nonInteractive, dryRun, jsonOutput bool
	buildInfo, _ := debug.ReadBuildInfo()
	root := &cobra.Command{Use: "qrr", Short: "Run your command recipes", Version: versionString(buildInfo), SilenceUsage: true, SilenceErrors: true, TraverseChildren: true, Args: cobra.NoArgs}
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
		explicit := map[string]any{}
		for _, p := range r.Params {
			if selected || !cmd.Flags().Changed(p.Name) {
				continue
			}
			switch p.Type {
			case "confirm":
				explicit[p.Name], _ = cmd.Flags().GetBool(p.Name)
			case "multiselect":
				value, _ := cmd.Flags().GetString(p.Name)
				if value == "" {
					explicit[p.Name] = []string{}
				} else {
					explicit[p.Name] = strings.Split(value, ",")
				}
			default:
				explicit[p.Name], _ = cmd.Flags().GetString(p.Name)
			}
		}
		var ask recipe.AskFunc
		if !nonInteractive && prompt.IsTerminal() {
			ask = prompt.Ask
		}
		argv, err := r.Prepare(cmd.Context(), explicit, ask)
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
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Running:", commandText(argv)); err != nil {
			return err
		}
		return runner.Run(cmd.Context(), argv)
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if nonInteractive || !prompt.IsTerminal() {
			return fmt.Errorf("a command name is required outside interactive mode; use qrr list or qrr --help")
		}
		r, err := prompt.Choose(cmd.Context(), recipes, prompt.CommandActions{
			Rename: store.Rename,
			Edit:   func(name string) error { return editRecipe(cmd.Context(), store, name) },
			Reload: store.Load,
		})
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
	addManagement(root, store, recipes, problems, func() bool { return nonInteractive })
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
