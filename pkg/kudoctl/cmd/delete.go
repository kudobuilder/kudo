package cmd

import (
	"fmt"

	deletePkg "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/delete"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"

	"github.com/spf13/cobra"
)

var (
	deleteExample = `
		# Delete an instance of a framework from your cluster.
		kubectl kudo delete flink`
)

// NewDeleteCmd creates the delete command for the CLI
func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:          "delete <instance>",
		Short:        "-> Delete an instance of a framework.",
		Long:         "Delete an instance of a framework.",
		Example:      deleteExample,
		RunE:         deletePkg.CmdErrorProcessor,
		SilenceUsage: true,
	}

	deleteCmd.Flags().StringVar(&vars.KubeConfigPath, "kubeconfig", "", "The file path to Kubernetes configuration file. (default \"$HOME/.kube/config\")")
	deleteCmd.Flags().StringVar(&vars.Namespace, "namespace", "default", "The namespace where the operator watches for changes. (default \"default\")")

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	deleteCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(deleteCmd.OutOrStderr(), usageFmt, deleteCmd.UseLine(), deleteCmd.Flags().FlagUsages())
		return nil
	})
	return deleteCmd
}
