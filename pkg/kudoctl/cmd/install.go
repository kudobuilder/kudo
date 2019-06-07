package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/spf13/cobra"
)

var (
	installExample = `
		# Install the most recent Flink package to your cluster.
		kubectl kudo install flink

		# Install the Kafka package with all of its dependencies to your cluster.
		kubectl kudo install kafka --all-dependencies

		# Specify a package version of Kafka to install to your cluster.
		kubectl kudo install kafka --package-version=0`
)

// newInstallCmd creates the install command for the CLI
func newInstallCmd() *cobra.Command {
	installCmd := &cobra.Command{
		Use:          "install <name>",
		Short:        "-> Install an official KUDO package.",
		Long:         `Install a KUDO package from the official GitHub repo.`,
		Example:      installExample,
		RunE:         install.RunInstall,
		SilenceUsage: true,
	}

	installCmd.Flags().BoolVar(&vars.AllDependencies, "all-dependencies", false, "Installs all dependency packages. (default \"false\")")
	installCmd.Flags().BoolVar(&vars.AutoApprove, "auto-approve", false, "Skip interactive approval when existing version found. (default \"false\")")
	installCmd.Flags().StringVar(&vars.KubeConfigPath, "kubeconfig", "", "The file path to Kubernetes configuration file. (default \"$HOME/.kube/config\")")
	installCmd.Flags().StringVar(&vars.Instance, "instance", "", "The instance name. (default to Framework name)")
	installCmd.Flags().StringVar(&vars.Namespace, "namespace", "default", "The namespace used for the framework installation. (default \"default\"")
	installCmd.Flags().StringArrayVarP(&vars.Parameter, "parameter", "p", nil, "The parameter name and value separated by '='")
	installCmd.Flags().StringVar(&vars.PackageVersion, "package-version", "", "A specific package version on the official GitHub repo. (default to the most recent)")

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	installCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(installCmd.OutOrStderr(), usageFmt, installCmd.UseLine(), installCmd.Flags().FlagUsages())
		return nil
	})
	return installCmd
}
