package cmd

import (
	"github.com/maestrosdk/maestro/version"
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
	# View plans
	maestroctl plan --instance=<instanceName> --namespace=<default> --kubeconfig=<$HOME/.kube/config>

	# View plan status
	maestroctl plan list <planName> --instance=<instanceName> --namespace=<default> --kubeconfig=<$HOME/.kube/config>
`,
		Version: version.Version,
	}

	cmd.AddCommand(NewMaestroCTLCmd())

	return cmd
}
