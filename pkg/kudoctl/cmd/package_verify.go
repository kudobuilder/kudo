package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/verify"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier/template"
)

// package verify provides verification or linting checks against the package passed to the command.

type packageVerifyCmd struct {
	fs          afero.Fs
	out         io.Writer
	output      output.Type
	paramChecks []string
}

func newPackageVerifyCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	verifyCmd := &packageVerifyCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "verify [package]",
		Short:   "verify KUDO package",
		Example: "  kubectl kudo package verify ../zk/operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}

			return verifyCmd.run(args[0])
		},
	}

	cmd.Flags().StringVarP((*string)(&verifyCmd.output), "output", "o", "", "Output format for command results.")
	cmd.Flags().StringSliceVar(&verifyCmd.paramChecks, "param-checks", []string{}, fmt.Sprintf("Additional parameter checks: %v", template.ParamVerifyArguments))
	return cmd
}

func (c *packageVerifyCmd) run(path string) error {
	if err := c.output.Validate(); err != nil {
		return err
	}
	opts := template.ExtendedParametersVerifier{}
	if err := opts.SetFromArguments(c.paramChecks); err != nil {
		return err
	}
	return verifyPackage(c.fs, path, c.out, c.output, []packages.Verifier{&opts})
}

func verifyPackage(fs afero.Fs, path string, out io.Writer, outType output.Type, additionalVerifiers []packages.Verifier) error {
	pf, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return err
	}
	res := verify.PackageFiles(pf)
	for _, v := range additionalVerifiers {
		res.Merge(v.Verify(pf))
	}

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
