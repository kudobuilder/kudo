package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/version"
)

var (
	// initialization of filesystem for all commands
	fs = afero.NewOsFs()

	// Settings defines global flags and settings
	Settings env.Settings
)

// NewKudoctlCmd creates a new root command for kudoctl
func NewKudoctlCmd() *cobra.Command {
	cmd := &cobra.Command{
		// Workaround or Compromise as "kubectl kudo" would result in Usage print out "kubectl install <name> [flags]"
		Use:   "kubectl-kudo",
		Short: "CLI to manipulate, inspect and troubleshoot KUDO-specific CRDs.",
		Long: `KUDO CLI and future sub-commands can be used to manipulate, inspect and troubleshoot KUDO-specific CRDs
and serves as an API aggregation layer.
`,
		SilenceUsage: true,
		Example: `  # Install a KUDO package from the official repo.
  kubectl kudo install <name> [flags]

  # Install a KUDO package from the local filesystem.
  kubectl kudo install <./path> [flags]

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

	cmd.AddCommand(newInstallCmd(fs))
	cmd.AddCommand(newInitCmd(fs, cmd.OutOrStdout(), cmd.ErrOrStderr(), nil))
	cmd.AddCommand(newUpgradeCmd(fs))
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newPackageCmd(fs, cmd.OutOrStdout()))
	cmd.AddCommand(newGetCmd(cmd.OutOrStdout()))
	cmd.AddCommand(newPlanCmd(cmd.OutOrStdout()))
	cmd.AddCommand(newRepoCmd(fs, cmd.OutOrStdout()))
	cmd.AddCommand(newSearchCmd(fs, cmd.OutOrStdout()))
	cmd.AddCommand(newTestCmd())
	cmd.AddCommand(newVersionCmd(cmd.OutOrStdout()))
	cmd.AddCommand(newDiagnosticsCmd(fs))

	initGlobalFlags(cmd, cmd.OutOrStdout())

	return cmd
}

func initGlobalFlags(cmd *cobra.Command, out io.Writer) {
	flags := cmd.PersistentFlags()
	Settings.AddFlags(flags)
	clog.InitWithFlags(flags, out)
}
