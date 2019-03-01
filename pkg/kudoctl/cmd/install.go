package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	//"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/spf13/cobra"
)

var (
	installExample = `
		# Install the most recent Flink Framework to your cluster.
		kubectl kudo install flink

		# Install Kafka Framework with all of its dependencies to your cluster.
		kubectl kudo install kafka --all-dependencies

		# Specify a repo version of Kafka to install to your cluster.
		kubectl kudo install kafka --repo-version=0`
)

func NewCmdInstall() *cobra.Command {
	installCmd := &cobra.Command{
		Use:          "install <name>",
		Short:        "-> Install a KUDO Framework and FrameworkVersion.",
		Long:         `Install a KUDO Framework and FrameworkVersion from the official GitHub repo.`,
		Example:      installExample,
		RunE:         install.InstallCmd,
		SilenceUsage: false,
	}

	installCmd.Flags().BoolVar(&vars.AllDependencies, "all-dependencies", false, "Installs all dependency Frameworks with FrameworkVersions. (default \"false\")")
	installCmd.Flags().BoolVar(&vars.AutoApprove, "auto-approve", false, "Skip interactive approval when existing version found. (default \"false\")")
	installCmd.Flags().StringVar(&vars.KubeConfigPath, "kubeconfig", "", "The file path to Kubernetes configuration file. (default \"$HOME/.kube/config\")")
	installCmd.Flags().StringVar(&vars.GithubCredentialPath, "githubcredential", "", "The file path to GitHub credential file. (default \"$HOME/.git-credentials\")")
	installCmd.Flags().StringVar(&vars.Instance, "instance", "", "The instance name. (default to Framework name)")
	installCmd.Flags().StringVar(&vars.Namespace, "namespace", "default", "The namespace where the operator watches for changes.")
	installCmd.Flags().StringArrayVarP(&vars.Parameter, "parameter", "p", nil, "The instance name. (default to Framework name)")
	installCmd.Flags().StringVar(&vars.RepoVersion, "repo-version", "", "A specific repo version on the official GitHub repo. (default to the most recent)")

	return installCmd
}
