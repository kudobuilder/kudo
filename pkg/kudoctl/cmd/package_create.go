package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/writer"
)

const (
	pkgCreateDesc = `Package a KUDO operator from local filesystem into a package tarball.
The package argument must be a directory which contains the operator definition files.  The package command will create a tgz file containing the operator.
`
	pkgCreateExample = `  # package zookeeper (where zookeeper is a folder in the current directory)
  kubectl kudo package create zookeeper

  # Specify a destination folder other than current working directory
  kubectl kudo package create ../operators/repository/zookeeper/operator/ --destination=out-folder`
)

type packageCreateCmd struct {
	path        string
	destination string
	overwrite   bool
	out         io.Writer
	output      output.Type
	fs          afero.Fs
}

// newPackageCreateCmd creates an operator tarball. fs is the file system, out is stdout for CLI
func newPackageCreateCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageCreateCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "create <operator_dir>",
		Short:   "package a local KUDO operator into a tarball.",
		Long:    pkgCreateDesc,
		Example: pkgCreateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			if err := pkg.output.Validate(); err != nil {
				return err
			}
			pkg.path = args[0]
			if err := pkg.run(); err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&pkg.destination, "destination", "d", ".", "Location to write the package.")
	f.BoolVarP(&pkg.overwrite, "overwrite", "w", false, "Overwrite existing package.")
	f.StringVarP((*string)(&pkg.output), "output", "o", "", "Output format for command results, NOT the package format.")

	return cmd
}

func validateOperatorArg(args []string) error {
	if len(args) != 1 {
		return errors.New("expecting exactly one argument - directory of the operator or name of package")
	}
	return nil
}

// run returns the errors associated with cmd env
func (pkg *packageCreateCmd) run() error {
	err := verifyPackage(pkg.fs, pkg.path, pkg.out, pkg.output, []packages.Verifier{})
	if err != nil {
		return err
	}
	tarfile, err := writer.WriteTgz(pkg.fs, pkg.path, pkg.destination, pkg.overwrite)
	if err == nil && pkg.output == "" {
		fmt.Fprintf(pkg.out, "Package created: %v\n", tarfile)
	}
	return err
}
