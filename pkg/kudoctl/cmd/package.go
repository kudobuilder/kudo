package cmd

import (
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	example = `
		The package argument must be a directory which contains the operator definition files.  The package command will create a tgz file containing the operator.

		# package zookeeper (where zookeeper is a folder in the current directory)
		kubectl kudo package zookeeper

		# Specify a destination folder other than current working directory
		kubectl kudo package ../operators/repository/zookeeper/operator/ --destination=out-folder`
)

type packageCmd struct {
	path        string
	destination string
	overwrite   bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageCmd creates an operator tarball. fs is the file system, out is stdout for CLI
func newPackageCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "package <operator_dir>",
		Short:   "Package a local KUDO operator into a tarball.",
		Long:    `Package a KUDO operator from local filesystem into a package tarball.`,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			pkg.path = args[0]
			if err := pkg.run(); err != nil {
				return err
			}
			return nil
		},
		SilenceUsage: true,
	}

	f := cmd.Flags()
	f.StringVarP(&pkg.destination, "destination", "d", ".", "Location to write the package.")
	f.BoolVarP(&pkg.overwrite, "overwrite", "o", false, "Overwrite existing package.")
	return cmd
}

func validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - directory of the operator to package")
	}
	return nil
}

// run returns the errors associated with cmd env
func (pkg *packageCmd) run() error {
	tarfile, err := bundle.ToTarBundle(pkg.fs, pkg.path, pkg.destination, pkg.overwrite)
	if err == nil {
		fmt.Fprintf(pkg.out, "Package created: %v\n", tarfile)
	}
	return err
}
