package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/get"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new command that lists instances
func NewListCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "list",
		Short: "-> Show all available instances.",
		Long:  `The list command has subcommands to show all available instances.`,
	}

	newCmd.AddCommand(get.NewGetInstancesCmd())

	return newCmd
}
