package cmd

import (
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/verify"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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
		Short:   "verify operator parameters",
		Example: "  kubectl kudo operator verify",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			return list.run(fs, args[0])
		},
	}

	return cmd
}

func (c *packageVerifyCmd) run(fs afero.Fs, path string) error {

	pf, err := packages.FromFolder(fs, path)
	if err != nil {
		return err
	}
	warnings, errors := verify.Parameters(pf.Params)

	if warnings != nil {
		printWarnings(c.out, warnings)
	}
	if errors != nil {
		printErrors(c.out, errors)
		return fmt.Errorf("operator verification errors: %v", len(errors))
	}
	if warnings == nil && errors == nil {
		fmt.Fprintf(c.out, "operator is valid\n")
	}

	//TODO (kensipe): add linting
	// 2. warning on params not used
	// 3. error on param in template not defined
	return nil
}

func printErrors(out io.Writer, errors verify.ParamErrors) {
	table := uitable.New()
	table.AddRow("Errors")
	for _, warning := range errors {
		table.AddRow(warning)
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
