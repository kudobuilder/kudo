package cmd

import (
	"github.com/universal-operator/universal-operator/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maestroctl",
		Short: "CLI to manipulate, inspect and troubleshoot Maestro-specific CRDs.",
		Long: `
Maestro CLI and future sub-commands can be used to manipulate, inspect and troubleshoot Maestro-specific CRDs
and serves as an API aggregation layer.
`,
		Example: `
	# List instances
	maestroctl list instances --namespace=<default> --kubeconfig=<$HOME/.kube/config>

	# View plan status
	maestroctl plan status --instance=<instanceName> --kubeconfig=<$HOME/.kube/config>

	# View plan history of a specific FrameworkVersion and Instance
	maestroctl plan history <frameworkVersion> --instance=<instanceName> --namespace=<default> --kubeconfig=<$HOME/.kube/config>

	# View all plan history of a specific Instance
	maestroctl plan history --instance=<instanceName> --namespace=<default> --kubeconfig=<$HOME/.kube/config>
`,
		Version: version.Version,
	}

	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewPlanCmd())

	return cmd
}
