package cmd

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/verify"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

//TODO (kensipe): add long desc

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
	warnings, errors := verify.Parameters(pf.Params.Parameters)
	if warnings != nil {
		printWarnings(out, warnings)
	}
	if errors == nil {
		fmt.Fprintf(out, "package is valid\n")
		return nil
	}
	printErrors(out, errors)
	return fmt.Errorf("package verification errors: %v", len(errors))
	//TODO (kensipe): add linting
	// 2. warning on params not used
	// 3. error on param in template not defined
}

func printErrors(out io.Writer, errors verify.ParamErrors) {
	table := uitable.New()
	table.AddRow("Errors")
	for _, err := range errors {
		table.AddRow(err)
	}
	fmt.Fprintln(out, table)
}

func printWarnings(out io.Writer, warnings verify.ParamWarnings) {
	table := uitable.New()
	table.AddRow("Warnings")
	for _, warning := range warnings {
		table.AddRow(warning)
	}
	fmt.Fprintln(out, table)
}
