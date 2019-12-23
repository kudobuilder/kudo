package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const paramsDesc = `
This command consists of multiple sub-commands to interact with KUDO packages.  These commands are used in the listing 
of an operator details such as parameters, tasks or plans.

List operator parameters
`

const paramsExamples = `  kubectl kudo package list parameters [operator folder]
  kubectl kudo package list task [operator folder]
  kubectl kudo package list plans [operator folder]
`

// newPackageParamsCmd for repo commands such as building a repo index
func newPackageParamsCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [FLAGS]",
		Short:   "list context from an operator package",
		Long:    paramsDesc,
		Example: paramsExamples,
	}
	cmd.AddCommand(newParamsListCmd(fs, out))

	return cmd
}
