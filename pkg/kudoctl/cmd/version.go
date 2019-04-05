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

func NewCmdVersion() *cobra.Command {
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

func VersionCmd(cmd *cobra.Command, args []string) error {
	kudoVersion := version.Get()
	fmt.Printf("KUDO Version: %s\n", fmt.Sprintf("%#v", kudoVersion))
	return nil
}
