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

// newPackageCmd creates an operator bundle.  fs is the file system, out is stdout for CLI
func newPackageCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	b := &packageCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "package <operator_dir>",
		Short:   "Package an official KUDO operator.",
		Long:    `Package a KUDO operator from local filesystem.`,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			b.path = args[0]
			if err := b.run(); err != nil {
				return err
			}
			return nil
		},
		SilenceUsage: true,
	}

	f := cmd.Flags()
	f.StringVarP(&b.destination, "destination", "d", ".", "Location to write the package.")
	f.BoolVarP(&b.overwrite, "overwrite", "o", false, "Overwrite existing package.")
	return cmd
}

func validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - directory of the operator to package")
	}
	return nil
}

// run returns the errors associated with cmd env
func (b *packageCmd) run() error {
	tarfile, err := bundle.ToTarBundle(b.fs, b.path, b.destination, b.overwrite)
	if err == nil {
		fmt.Fprintf(b.out, "Package created: %v\n", tarfile)
	}
	return err
}
