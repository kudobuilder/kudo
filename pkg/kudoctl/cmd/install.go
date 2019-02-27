package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	//"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/spf13/cobra"
)

func NewCmdInstall() *cobra.Command {
	installCmd := &cobra.Command{
		Use:          "install",
		Short:        "-> Install a KUDO Framework and Frameworkversion.",
		Long:         `The list command has subcommands to show all available instances.`,
		RunE:         install.InstallCmd,
		SilenceUsage: false,
	}

	installCmd.Flags().BoolVar(&vars.AutoApprove, "auto-approve", false, "Skip interactive approval when existing version found.")
	installCmd.Flags().StringVar(&vars.KubeConfigPath, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	installCmd.Flags().StringVar(&vars.GithubCredentialPath, "githubcredential", "", "The file path to github credential file; defaults to $HOME/.git-credentials")

	installCmd.Flags().StringVar(&vars.Namespace, "namespace", "default", "The namespace where the operator watches for changes.")

	return installCmd
}
