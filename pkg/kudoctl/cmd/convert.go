package cmd

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/convert"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

// newConvertCmd creates a new command that shows the plans available for an instance
func newConvertCmd() *cobra.Command {
	options := convert.DefaultOptions
	newCmd := &cobra.Command{
		Use:   "convert",
		Short: "Converts helm chart into KUDO repo structure",
		Long:  `Converts the provided helm repo into KUDO structure by modifying `,
		RunE:  func(cmd *cobra.Command, args []string) error {
			return convert.Run(cmd, args, options)
		},
	}

	newCmd.Flags().StringVarP(&options.ChartImportPath, "folder", "f", "", "Folder directory to import")
	newCmd.Flags().StringVarP(&options.OutputPath, "out", "o", "", "Folder Directory to output REPO.  Should NOT exist")
	// newCmd.Flags().StringVarp()

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	newCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(newCmd.OutOrStderr(), usageFmt, newCmd.UseLine(), newCmd.Flags().FlagUsages())
		return nil
	})

	return newCmd
}
