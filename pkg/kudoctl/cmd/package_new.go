package cmd

import (
	"errors"
	"io"

	"github.com/Masterminds/semver"
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
	pathDefault := "operator"
	ovDefault := "0.1.0"
	kudoDefault := version.Get().GitVersion
	apiVersionDefault := reader.APIVersion

	if !pkg.interactive {
		op := packages.OperatorFile{
			Name:        pkg.name,
			APIVersion:  apiVersionDefault,
			Version:     ovDefault,
			KUDOVersion: kudoDefault,
		}

		return generate.Operator(pkg.fs, pathDefault, op, pkg.overwrite)
	}

	// interactive mode
	nameValid := func(input string) error {
		if len(input) < 3 {
			return errors.New("Operator name must have more than 3 characters")
		}
		return nil
	}

	name, err := prompt.WithValidator("Operator Name", pkg.name, nameValid)
	if err != nil {
		return err
	}

	pathValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Operator directory must have more than 1 character")
		}
		return generate.CanGenerateOperator(pkg.fs, input, pkg.overwrite)
	}

	path, err := prompt.WithValidator("Operator directory", pathDefault, pathValid)
	if err != nil {
		return err
	}

	versionValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Operator version is required in semver format")
		}
		_, err := semver.NewVersion(input)
		return err
	}
	opVersion, err := prompt.WithValidator("Operator Version", ovDefault, versionValid)
	if err != nil {
		return err
	}

	appVersion, err := prompt.WithDefault("Application Version", "")
	if err != nil {
		return err
	}

	kudoVersion, err := prompt.WithDefault("Required KUDO Version", kudoDefault)
	if err != nil {
		return err
	}

	url, err := prompt.WithDefault("Project URL", "")
	if err != nil {
		return err
	}

	op := packages.OperatorFile{
		Name:        name,
		APIVersion:  apiVersionDefault,
		Version:     opVersion,
		AppVersion:  appVersion,
		KUDOVersion: kudoVersion,
		URL:         url,
	}

	return generate.Operator(pkg.fs, path, op, pkg.overwrite)
}
