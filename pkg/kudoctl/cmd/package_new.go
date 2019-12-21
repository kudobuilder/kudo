package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	pkgNewDesc = `Create a new KUDO operator on the local filesystem`

	pkgNewExample = `  # Create a new KUDO operator name foo
  kubectl kudo package new foo
`
)

type packageNewCmd struct {
	name        string
	out         io.Writer
	fs          afero.Fs
	interactive bool
	overwrite   bool
}

// newPackageNewCmd creates an operator package on the file system
func newPackageNewCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageNewCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "new <operator name>",
		Short:   "create new operator",
		Long:    pkgNewDesc,
		Example: pkgNewExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			pkg.name = args[0]
			if err := pkg.run(); err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&pkg.interactive, "interactive", "i", false, "Interactively create operator")
	f.BoolVarP(&pkg.overwrite, "overwrite", "w", false, "overwrite existing directory and operator.yaml file")
	return cmd
}

// run returns the errors associated with cmd env
func (pkg *packageNewCmd) run() error {

	// defaults
	pathDefault := "operator"
	opDefault := packages.OperatorFile{
		Name:        pkg.name,
		APIVersion:  reader.APIVersion,
		Version:     "0.1.0",
		KUDOVersion: version.Get().GitVersion,
	}

	if !pkg.interactive {

		return generate.Operator(pkg.fs, pathDefault, &opDefault, pkg.overwrite)
	}

	// interactive mode
	op, path, err := prompt.ForOperator(pkg.fs, pathDefault, pkg.overwrite, opDefault)
	if err != nil {
		return err
	}

	return generate.Operator(pkg.fs, path, op, pkg.overwrite)
}
