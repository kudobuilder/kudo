package cmd

import (
	"os"

	"github.com/kudobuilder/kudo/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewKudoctlCmd creates a new root command for kudoctl
func NewKudoctlCmd() *cobra.Command {
	cmd := &cobra.Command{
		// Workaround or Compromise as "kubectl kudo" would result in Usage print out "kubectl install <name> [flags]"
		Use:   "kubectl-kudo",
		Short: "CLI to manipulate, inspect and troubleshoot KUDO-specific CRDs.",
		Long: `
KUDO CLI and future sub-commands can be used to manipulate, inspect and troubleshoot KUDO-specific CRDs
and serves as an API aggregation layer.
`,
		Example: `
	# Install a KUDO package from the official GitHub repo.
	kubectl kudo install <name> [flags]

	# View plan history of a specific package
	kubectl kudo plan history <name> [flags]

	# View all plan history of a specific package
	kubectl kudo plan history [flags]

	# Run integration tests against a Kubernetes cluster or mocked control plane.
	kubectl kudo test

	# Get instances
	kubectl kudo get instances [flags]

	# View plan status
	kubectl kudo plan status [flags]

	# View KUDO version
	kubectl kudo version

`,
		Version: version.Get().GitVersion,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUpgradeCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newPlanCmd())
	cmd.AddCommand(newTestCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.PersistentFlags().String("kubeconfig", os.Getenv("HOME")+"/.kube/config", "Path to your Kubernetes configuration file")
	viper.BindEnv("kubeconfig")
	viper.BindPFlag("kubeconfig", cmd.PersistentFlags().Lookup("kubeconfig"))

	return cmd
}
