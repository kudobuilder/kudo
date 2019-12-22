package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const packageAddDesc = `
This command consists of multiple sub-commands to interact with KUDO packages.  These commands are used in the creation 
or updating of an operator by an operator developer.

It can be used to add parameters, tasks and maintainers.  It is expected to be used inside an operator package or optional 
can be used with the operator as a sub folder (such as a folder named "operator").  The sub folder name doesn't matter as 
long as there is only one sub folder with an operator.yaml file.
`

const packageAddExamples = `  kubectl kudo package add parameter
  kubectl kudo package add maintainer
  kubectl kudo package add task
`

// newPackageAddCmd for operator commands such as adding parameters or maintainers to a package
func newPackageAddCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <operator_dir>",
		Short:   "add content to an operator package files",
		Long:    packageAddDesc,
		Example: packageAddExamples,
	}

	cmd.AddCommand(newPackageAddMaintainerCmd(fs, out))
	cmd.AddCommand(newPackageAddParameterCmd(fs, out))
	cmd.AddCommand(newPackageAddTaskCmd(fs, out))
	cmd.AddCommand(newPackageAddPlanCmd(fs, out))

	return cmd
}
