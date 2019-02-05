package cmd

import (
	"github.com/universal-operator/universal-operator/cmd/maestroctl/cmd/list"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "list",
		Short: "-> Show all available instances.",
		Long:  `The list command has subcommands to show all available instances.`,
	}

	newCmd.AddCommand(list.NewListInstancesCmd())

	return newCmd
}
