package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/plan"
	"github.com/spf13/cobra"
)

// newPlanCmd creates a new command that shows the plans available for an instance
func newPlanCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "plan",
		Short: "-> View all available plans.",
		Long:  `The plan command has subcommands to view all available plans.`,
	}

	newCmd.AddCommand(plan.NewPlanHistoryCmd())
	newCmd.AddCommand(plan.NewPlanStatusCmd())

	return newCmd
}
