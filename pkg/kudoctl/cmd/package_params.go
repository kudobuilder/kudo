package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const paramsDesc = `
List operator parameters
`

const paramsExamples = `  kubectl kudo package params list [operator folder]
`

// newPackageParamsCmd for repo commands such as building a repo index
func newPackageParamsCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "params [FLAGS] list [ARGS]",
		Short:   "list kudo operator parameters",
		Long:    paramsDesc,
		Example: paramsExamples,
	}
	cmd.AddCommand(newParamsListCmd(fs, out))

	return cmd
}
