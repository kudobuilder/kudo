package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/get"
	"github.com/spf13/cobra"
)

// NewGetCmd creates a new command that lists instances
func NewGetCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "get",
		Short: "-> Show all available instances.",
		Long:  `The get command has subcommands to show all available instances.`,
	}

	newCmd.AddCommand(get.NewGetInstancesCmd())

	return newCmd
}
