package cmd

import (
	"github.com/kudobuilder/kudo/version"
	"github.com/spf13/cobra"
)

// NewKudoctlCmd creates a new root command for kudoctl
func NewKudoctlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kudoctl",
		Short: "CLI to manipulate, inspect and troubleshoot Kudo-specific CRDs.",
		Long: `
Kudo CLI and future sub-commands can be used to manipulate, inspect and troubleshoot Kudo-specific CRDs
and serves as an API aggregation layer.
`,
		Example: `
	# Install a KUDO Framework and FrameworkVersion from the official GitHub repo.
	kudoctl install <name> [flags]

	# View plan history of a specific FrameworkVersion and Instance
	kudoctl plan history <name> [flags]

	# View all plan history of a specific Instance
	kudoctl plan history [flags]

	# List instances
	kudoctl list instances [flags]

	# View plan status
	kudoctl plan status [flags]

`,
		Version: version.Version,
	}

	cmd.AddCommand(NewCmdInstall())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewPlanCmd())

	return cmd
}
