package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/version"

	"github.com/spf13/cobra"
)

var (
	versionExample = `
		# Print the current installed KUDO package version
		kubectl kudo version`
)

// NewVersionCmd returns a new initialized instance of the version sub command
func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:          "version",
		Short:        "-> Print the current KUDO package version.",
		Long:         `Print the current installed KUDO package version.`,
		Example:      versionExample,
		RunE:         VersionCmd,
		SilenceUsage: true,
	}

	const usageFmt = "Usage:\n  %s"
	versionCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(versionCmd.OutOrStderr(), usageFmt, versionCmd.UseLine())
		return nil
	})
	return versionCmd
}

// VersionCmd performs the version sub command
func VersionCmd(cmd *cobra.Command, args []string) error {
	kudoVersion := version.Get()
	fmt.Printf("KUDO Version: %s\n", fmt.Sprintf("%#v", kudoVersion))
	return nil
}
