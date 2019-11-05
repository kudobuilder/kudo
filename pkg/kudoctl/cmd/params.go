package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const paramsDesc = `
This command consists of multiple sub-commands to interact with KUDO parameters when working on creating or modifying an operator.

It can be used to list and lint operator properties
`

const paramsExamples = `  kubectl kudo params list [operator folder]
  kubectl kudo params lint [operator folder]
`

// newParamsCmd for repo commands such as building a repo index
func newParamsCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "params [FLAGS] list|lint [ARGS]",
		Short:   "list and lint kudo operator",
		Long:    paramsDesc,
		Example: paramsExamples,
	}
	cmd.AddCommand(newParamsListCmd(fs, out))
	cmd.AddCommand(newParamsLintCmd(fs, out))

	return cmd
}
