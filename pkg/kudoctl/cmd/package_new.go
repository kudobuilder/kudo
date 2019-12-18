package cmd

import (
	"errors"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	pkgNewDesc = `Create a new KUDO operator on the local filesystem`

	pkgNewExample = `  # Create a new KUDO operator name foo 
  kubectl kudo package foo
`
)

type packageNewCmd struct {
	name string
	out  io.Writer
	fs   afero.Fs
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

	return cmd
}

// run returns the errors associated with cmd env
func (pkg *packageNewCmd) run() error {

	nvalid := func(input string) error {
		if len(input) < 3 {
			return errors.New("Operator name must have more than 3 characters")
		}
		return nil
	}

	name, err := prompt.WithValidator("Operator Name", pkg.name, nvalid)
	if err != nil {
		return err
	}

	dvalid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Operator directory must have more than 1 character")
		}
		return generate.OperatorCheck(pkg.fs, input)
	}

	dir, err := prompt.WithValidator("Operator directory", "operator", dvalid)
	if err != nil {
		return err
	}

	//TODO (kensipe): need list of supported versions some where
	vOptions := []string{"kudo.dev/v1beta1"}
	apiVersion, err := prompt.WithOptions("API Version", vOptions)
	if err != nil {
		return err
	}

	opVersion, err := prompt.WithDefault("Operator Version", "")
	if err != nil {
		return err
	}

	appVersion, err := prompt.WithDefault("Application Version", "")
	if err != nil {
		return err
	}

	kudoVersion, err := prompt.WithDefault("Required KUDO Version", version.Get().GitVersion)
	if err != nil {
		return err
	}

	url, err := prompt.WithDefault("Project URL", "")
	if err != nil {
		return err
	}

	op := packages.OperatorFile{
		Name:        name,
		APIVersion:  apiVersion,
		Version:     opVersion,
		AppVersion:  appVersion,
		KUDOVersion: kudoVersion,
		URL:         url,
	}

	return generate.Operator(pkg.fs, dir, op)
}
