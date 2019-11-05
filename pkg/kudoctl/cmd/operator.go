package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const operatorDesc = `
This command consists of multiple sub-commands to interact with KUDO operators.

It can be used to package or verify an operator, or list parameters.  When working with parameters it can 
provide a list of parameters from a remote operator given a url or repository along with the name and version.
`

const operatorExamples = `  kubectl kudo operator package [operator folder]
  kubectl kudo operator params list [operator]
  kubectl kudo operator verify [operator]
`

// newOperatorCmd for operator commands such as packaging an operator or retrieving it's parameters
func newOperatorCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "operator [FLAGS] package|params|verify [ARGS]",
		Short:   "Package an operator, or understand it's content",
		Long:    operatorDesc,
		Example: operatorExamples,
	}

	cmd.AddCommand(newPackageCmd(fs, out))
	cmd.AddCommand(newParamsCmd(fs, out))
	cmd.AddCommand(newOperatorVerifyCmd(fs, out))

	return cmd
}
