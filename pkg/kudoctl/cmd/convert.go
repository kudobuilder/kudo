package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/convert"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

// NewConvertCmd creates a new command that shows the plans available for an instance
func NewConvertCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "convert",
		Short: "Converts helm chart into KUDO repo structure",
		Long:  `Converts the provided helm repo into KUDO structure by modifying `,
		RunE:  convert.Run,
	}

	newCmd.Flags().StringVarP(&vars.FrameworkImportPath, "folder", "f", "", "Folder directory to import")
	newCmd.Flags().StringVarP(&vars.Format, "out", "o", "", "Folder Directory to output REPO.  Should NOT exist")
	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	newCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(newCmd.OutOrStderr(), usageFmt, newCmd.UseLine(), newCmd.Flags().FlagUsages())
		return nil
	})

	return newCmd
}
