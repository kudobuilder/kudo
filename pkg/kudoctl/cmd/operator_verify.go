package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type operatorVerifyCmd struct {
	fs  afero.Fs
	out io.Writer
}

func newOperatorVerifyCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &operatorVerifyCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "verify [operator]",
		Short:   "verify operator parameters",
		Example: "  kubectl kudo operator verify",
		RunE: func(cmd *cobra.Command, args []string) error {
			//list.home = Settings.Home
			return list.run(fs)
		},
	}

	return cmd
}

func (c *operatorVerifyCmd) run(fs afero.Fs) error {

	//TODO (kensipe): add linting
	// 1. error on dups
	// 2. warning on params not used
	// 3. error on param in template not defined
	return nil
}
