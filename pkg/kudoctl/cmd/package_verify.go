package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/verify"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// package verify provides verification or linting checks against the package passed to the command.

type packageVerifyCmd struct {
	fs     afero.Fs
	out    io.Writer
	output output.Type
}

func newPackageVerifyCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	verifyCmd := &packageVerifyCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "verify [package]",
		Short:   "verify package parameters",
		Example: "  kubectl kudo package verify ../zk/operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}

			return verifyCmd.run(args[0])
		},
	}

	cmd.Flags().StringVarP((*string)(&verifyCmd.output), "output", "o", "", "Output format")

	return cmd
}

func (c *packageVerifyCmd) run(path string) error {
	if err := c.output.Validate(); err != nil {
		return err
	}
	return verifyPackage(c.fs, path, c.out, c.output)
}

func verifyPackage(fs afero.Fs, path string, out io.Writer, outType output.Type) error {
	pf, err := reader.FromDir(fs, path)
	if err != nil {
		return err
	}
	res := verify.PackageFiles(pf)

	if outType != "" {
		if err = output.WriteObject(res.ForOutput(), outType, out); err != nil {
			return err
		}
	} else {
		verify.PrintResult(res, out)
	}

	if res.IsValid() {
		return nil
	}
	return fmt.Errorf("found %d package verification errors", len(res.Errors))
}
