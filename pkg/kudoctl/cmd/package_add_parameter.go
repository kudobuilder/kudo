package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
)

const (
	pkgAddParameterDesc = `Adds a parameter to existing operator package files.
`
	pkgAddParameterExample = `  kubectl kudo package add parameter
`
)

type packageAddParameterCmd struct {
	path        string
	interactive bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageAddParameterCmd adds a parameter to an exist operator params.yaml file
func newPackageAddParameterCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageAddParameterCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "parameter",
		Short:   "adds a parameter to the params.yaml file",
		Long:    pkgAddParameterDesc,
		Example: pkgAddParameterExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := generate.OperatorPath(fs)
			if err != nil {
				return err
			}
			pkg.path = path
			if err := pkg.run(); err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&pkg.interactive, "interactive", "i", false, "Interactive mode.")
	return cmd
}

func (pkg *packageAddParameterCmd) run() error {

	// interactive mode
	planNames, err := generate.PlanNameList(pkg.fs, pkg.path)
	if err != nil {
		return err
	}

	paramNames, err := generate.ParameterNameList(pkg.fs, pkg.path)
	if err != nil {
		return err
	}

	param, err := prompt.ForParameter(planNames, paramNames)
	if err != nil {
		return err
	}

	return generate.AddParameter(pkg.fs, pkg.path, param)
}
