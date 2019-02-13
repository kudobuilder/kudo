package cmd

import (
	"github.com/kudobuilder/kudo/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kudoctl",
		Short: "CLI to manipulate, inspect and troubleshoot Kudo-specific CRDs.",
		Long: `
Kudo CLI and future sub-commands can be used to manipulate, inspect and troubleshoot Kudo-specific CRDs
and serves as an API aggregation layer.
`,
		Example: `
	# List instances
	kudoctl list instances --namespace=<default> --kubeconfig=<$HOME/.kube/config>

	# View plan status
	kudoctl plan status --instance=<instanceName> --kubeconfig=<$HOME/.kube/config>

	# View plan history of a specific FrameworkVersion and Instance
	kudoctl plan history <frameworkVersion> --instance=<instanceName> --namespace=<default> --kubeconfig=<$HOME/.kube/config>

	# View all plan history of a specific Instance
	kudoctl plan history --instance=<instanceName> --namespace=<default> --kubeconfig=<$HOME/.kube/config>
`,
		Version: version.Version,
	}

	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewPlanCmd())

	return cmd
}
