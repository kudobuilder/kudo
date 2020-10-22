package cmd

import (
	"errors"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
)

const (
	pkgAddMaintainerDesc = `Adds a maintainer to existing operator package files.
`
	pkgAddMaintainerExample = `  kubectl kudo package add maintainer
  
# Specify a destination folder other than current working directory
  kubectl kudo package add maintainer <name> <email>`
)

type packageAddMaintainerCmd struct {
	path        string
	interactive bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageAddMaintainerCmd adds a maintainer to an exist operator package
func newPackageAddMaintainerCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageAddMaintainerCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "maintainer",
		Short:   "adds a maintainer to the operator.yaml file",
		Long:    pkgAddMaintainerDesc,
		Example: pkgAddMaintainerExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateAddMaintainerArg(args); err != nil {
				return err
			}
			checkMode(pkg, args)
			path, err := generate.OperatorPath(fs)
			if err != nil {
				return err
			}
			pkg.path = path
			if err := pkg.run(args); err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&pkg.interactive, "interactive", "i", false, "Interactive mode.")
	return cmd
}

func checkMode(pkg *packageAddMaintainerCmd, args []string) {
	pkg.interactive = len(args) == 0
}

// valid options are 0 (interactive mode) or 2
func validateAddMaintainerArg(args []string) error {
	if len(args) == 1 || len(args) > 2 {
		return errors.New("expecting two arguments - name and email address")
	}
	return nil
}

// run returns the errors associated with cmd env
func (pkg *packageAddMaintainerCmd) run(args []string) error {

	if !pkg.interactive {
		m := kudoapi.Maintainer{Name: args[0], Email: args[1]}
		return generate.AddMaintainer(pkg.fs, pkg.path, &m)
	}
	// interactive mode
	m, err := prompt.ForMaintainer()
	if err != nil {
		return err
	}

	return generate.AddMaintainer(pkg.fs, pkg.path, m)
}
