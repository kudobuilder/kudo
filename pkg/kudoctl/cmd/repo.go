package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const repoDesc = `
	This command consists of multiple sub-commands to interact with kudo repositories.
	
	It can be used to add, remove, list, and index kudo repositories.
	Example usage:
		$ kubectl kudo repo add [NAME] [REPO_URL]`


// newRepoCmd for repo commands such as building a repo index
func newRepoCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "repo [FLAGS] add|remove|list|index| update [ARGS]",
		Short:   "Package an official KUDO operator.",
		Long:    `Add, list, remove, update, and index kudo repositories`,
		Example: repoDesc,
	}

	cmd.AddCommand(newRepoIndexCmd(fs, out))

	return cmd
}