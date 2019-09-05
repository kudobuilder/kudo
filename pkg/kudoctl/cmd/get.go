package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/get"
	"github.com/spf13/cobra"
)

// newGetCmd creates a command that lists the instances in the cluster
func newGetCmd() *cobra.Command {
	options := get.DefaultOptions
	getCmd := &cobra.Command{
		Use:   "get instances",
		Short: "Gets all available instances.",
		Long: `
	# Get all available instances
	kudoctl get instances`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return get.Run(args, options, &Settings)
		},
	}

	getCmd.Flags().StringVar(&options.Namespace, "namespace", "default", "The namespace where the operator watches for changes.")
	return getCmd
}
