package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/verify"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// package verify provides verification or linting checks against the package passed to the command.

type packageVerifyCmd struct {
	fs  afero.Fs
	out io.Writer
}

func newPackageVerifyCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &packageVerifyCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "verify [package]",
		Short:   "verify package parameters",
		Example: "  kubectl kudo package verify ../zk/operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			return list.run(args[0])
		},
	}

	return cmd
}

func (c *packageVerifyCmd) run(path string) error {

	return verifyPackage(c.fs, path, c.out)
}

func verifyPackage(fs afero.Fs, path string, out io.Writer) error {
	pf, err := reader.FromDir(fs, path)
	if err != nil {
		return err
	}
	res := verify.PackageFiles(pf)
	res.PrintWarnings(out)
	res.PrintErrors(out)

	if res.IsValid() {
		fmt.Fprintf(out, "package is valid\n")
		return nil
	}

	return fmt.Errorf("found %d package verification errors", len(res.Errors))
}
