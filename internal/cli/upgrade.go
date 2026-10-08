package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newUpgradeCommand(upgrade func(context.Context, string, io.Writer) error) *cobra.Command {
	return &cobra.Command{
		Use: "upgrade", Short: "Upgrade qrr to the latest stable release", Args: cobra.NoArgs,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if dryRun || jsonOutput {
				return fmt.Errorf("upgrade does not support --dry-run or --json")
			}
			return upgrade(cmd.Context(), version, cmd.OutOrStdout())
		},
	}
}
