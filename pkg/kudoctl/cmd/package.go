package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const packageDesc = `
This command consists of multiple sub-commands to interact with KUDO packages.

It can be used to package or verify an operator, or list parameters.  When working with parameters it can 
provide a list of parameters from a remote operator given a url or repository along with the name and version.
`

const packageExamples = `  kubectl kudo package create [operator folder]
  kubectl kudo package list parameters [operator]
  kubectl kudo package verify [operator]
  kubectl kudo package add [subcommand]
`

// newPackageCmd for operator commands such as packaging an operator or retrieving it's parameters
func newPackageCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "package [SUBCOMMAND] [FLAGS] [ARGS]",
		Short:   "package an operator, or understand it's content",
		Long:    packageDesc,
		Example: packageExamples,
	}

	cmd.AddCommand(newPackageAddCmd(fs, out))
	cmd.AddCommand(newPackageCreateCmd(fs, out))
	cmd.AddCommand(newPackageNewCmd(fs, out))
	cmd.AddCommand(newPackageParamsCmd(fs, out))
	cmd.AddCommand(newPackageVerifyCmd(fs, out))

	return cmd
}
