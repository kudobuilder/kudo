package cmd

import (
	"github.com/maestrosdk/maestro/cmd/maestroctl/cmd/plan"
	"github.com/spf13/cobra"
)

func NewPlanCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "plan",
		Short: "-> Show all available plans.",
		Long:  `The plan command has subcommands to show all available plans.`,
	}

	newCmd.AddCommand(plan.NewPlanHistoryCmd())
	newCmd.AddCommand(plan.NewPlanListCmd())
	newCmd.AddCommand(plan.NewPlanStatusCmd())

	return newCmd
}
