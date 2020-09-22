package cmd

import (
	"errors"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
			if err := pkg.validateOperatorArg(args); err != nil {
				return err
			}
			if len(args) > 0 {
				pkg.name = args[0]
			}
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

func (pkg *packageNewCmd) validateOperatorArg(args []string) error {
	if pkg.interactive {
		// For interactive mode we use a default package name that can be adjusted with the prompt
		if len(args) > 1 {
			return errors.New("expecting at most one argument - name of the operator")
		}
		return nil
	}

	if len(args) != 1 {
		return errors.New("expecting exactly one argument - name of the operator. Or use -i for interactive mode")
	}
	return nil
}

// run returns the errors associated with cmd env
func (pkg *packageNewCmd) run() error {

	// defaults
	pathDefault := "operator"
	opDefault := packages.OperatorFile{
		Name:              "myoperator",
		APIVersion:        reader.APIVersion,
		OperatorVersion:   "0.1.0",
		AppVersion:        "0.1.0",
		KUDOVersion:       version.Get().GitVersion,
		KubernetesVersion: "0.16.0",
	}

	if pkg.name != "" {
		opDefault.Name = pkg.name
	}

	if !pkg.interactive {
		opDefault.Maintainers = []*kudoapi.Maintainer{
			{
				Name:  "My Name",
				Email: "MyEmail@invalid",
			},
		}
		return generate.Operator(pkg.fs, pathDefault, &opDefault, pkg.overwrite)
	}

	// interactive mode
	op, path, err := prompt.ForOperator(pkg.fs, pathDefault, pkg.overwrite, opDefault)
	if err != nil {
		return err
	}

	// Query First maintainer
	maintainer, err := prompt.ForMaintainer()
	if err != nil {
		return err
	}
	op.Maintainers = append(op.Maintainers, maintainer)

	return generate.Operator(pkg.fs, path, op, pkg.overwrite)
}
