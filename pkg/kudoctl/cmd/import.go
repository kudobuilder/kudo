package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	importcmd "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/import"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

// newImportCmd creates the import command for the CLI
func newImportCmd() *cobra.Command {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import folder as Framework and FrameworkVersion",
		Long: `Imports a folder with the KUDO or Helm folder structure to be applied	
	

	kubectl kudo import -f /path/to/definition | kubectl apply -f -
	`,
		RunE: importcmd.Run,
	}

	importCmd.Flags().StringVarP(&vars.FrameworkImportPath, "folder", "f", "", "Folder directory to import")
	importCmd.Flags().StringVarP(&vars.Format, "out", "o", "json", "Output format")
	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	importCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(importCmd.OutOrStderr(), usageFmt, importCmd.UseLine(), importCmd.Flags().FlagUsages())
		return nil
	})

	return importCmd
}
