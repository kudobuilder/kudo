package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/version"
)

var (
	versionExample = `  # Print the current installed KUDO package version
  kubectl kudo version`
)

type VersionForOutput struct {
	Client version.Info `json:"clientVersion"`
}

type versionCmdOpts struct {
	out    io.Writer
	output output.Type
}

// newVersionCmd returns a new initialized instance of the version sub command
func newVersionCmd(out io.Writer) *cobra.Command {
	versionCmdOpts := &versionCmdOpts{out: out}

	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the current KUDO package version.",
		Long:    `Print the current installed KUDO package version.`,
		Example: versionExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersionCmd(versionCmdOpts)
		},
	}

	versionCmd.Flags().StringVarP((*string)(&versionCmdOpts.output), "output", "o", "", "Output format for command results.")

	return versionCmd
}

// runVersionCmd performs the version sub command
func runVersionCmd(opts *versionCmdOpts) error {
	if err := opts.output.Validate(); err != nil {
		return err
	}
	v := VersionForOutput{
		Client: version.Get(),
	}

	if opts.output.IsFormattedOutput() {
		return output.WriteObject(v, opts.output, opts.out)
	}

	_, err := fmt.Fprintf(opts.out, "KUDO Version: %s\n", fmt.Sprintf("%#v", v.Client))
	return err
}
