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
			// Prior to command execution we parse and validate passed parameters
			return get.Run(args, options)
		},
	}

	getCmd.Flags().StringVar(&options.KubeConfigPath, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	getCmd.Flags().StringVar(&options.Namespace, "namespace", "default", "The namespace where the operator watches for changes.")

	return getCmd
}
