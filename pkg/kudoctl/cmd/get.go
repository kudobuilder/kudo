package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/get"
)

const getExample = `  # Get all available instances
  kubectl kudo get instances 
`

// newGetCmd creates a command that lists the instances in the cluster
func newGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:     "get instances",
		Short:   "Gets all available instances.",
		Example: getExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return get.Run(args, &Settings)
		},
	}

	return getCmd
}
