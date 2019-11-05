package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type paramsLintCmd struct {
	fs  afero.Fs
	out io.Writer
}

func newParamsLintCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &paramsLintCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "lint [operator]",
		Short:   "Lint operator parameters",
		Example: "  kubectl kudo params lint",
		RunE: func(cmd *cobra.Command, args []string) error {
			//list.home = Settings.Home
			return list.run(fs)
		},
	}

	return cmd
}

func (c *paramsLintCmd) run(fs afero.Fs) error {

	//TODO (kensipe): add linting
	// 1. error on dups
	// 2. warning on params not used
	// 3. error on param in template not defined
	return nil
}
