package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const repoDesc = `
This command consists of multiple sub-commands to interact with KUDO repositories.

It can be used to add, remove, list, and index kudo repositories.
`

const examples = `  kubectl kudo repo add [NAME] [REPO_URL]
  kubectl kudo repo remmove
  kubectl kudo repo list
  kubectl kudo repo context [NAME]
`

// newRepoCmd for repo commands such as building a repo index
func newRepoCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo [FLAGS] add|remove|list|index [ARGS]",
		Short:   "Add, list, remove, and index kudo repositories.",
		Long:    repoDesc,
		Example: examples,
	}

	cmd.AddCommand(newRepoIndexCmd(fs, out))
	cmd.AddCommand(newRepoListCmd(fs, out))
	cmd.AddCommand(newRepoAddCmd(fs, out))
	cmd.AddCommand(newRepoRemoveCmd(fs, out))
	cmd.AddCommand(newRepoContextCmd(fs))

	return cmd
}
