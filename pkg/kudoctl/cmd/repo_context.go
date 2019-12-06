package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

const (
	repoContextDesc = `When using an KUDO operation that requires access to a repository, KUDO needs to know which repository.
This is defined by the "context". 'kubectl kudo repo list' will provide the list of repositories.  The "*" next to one of the
names is the current context. 'kubectl kudo repo context local' will change the context to repository named local if it exists.
`
)

type repoContextCmd struct {
	name string
	home kudohome.Home

	fs afero.Fs
}

func newRepoContextCmd(fs afero.Fs) *cobra.Command {
	ctxCmd := &repoContextCmd{}

	cmd := &cobra.Command{
		Use:     "context [flags] [NAME]",
		Short:   "Set default for operator repository context",
		Long:    repoContextDesc,
		Example: "  kubectl kudo repo context local",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("need at least one argument, name of operator repository")
			}

			ctxCmd.name = args[0]
			ctxCmd.home = Settings.Home
			ctxCmd.fs = fs
			return ctxCmd.run()
		},
	}

	return cmd
}

func (c *repoContextCmd) run() error {
	repos, err := repo.LoadRepositories(c.fs, c.home.RepositoryFile())
	if err != nil {
		return err
	}
	if len(repos.Repositories) == 0 {
		return errors.New("no repositories to set")
	}
	if repos.GetConfiguration(c.name) == nil {
		return fmt.Errorf("no repo named %q found", c.name)
	}
	repos.Context = c.name
	return repos.WriteFile(fs, c.home.RepositoryFile(), 0644)
}
