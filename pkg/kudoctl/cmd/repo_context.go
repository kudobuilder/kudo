package cmd

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type repoContextCmd struct {
	name string
	home kudohome.Home

	fs afero.Fs
}

func newRepoContextCmd(fs afero.Fs) *cobra.Command {
	ctxCmd := &repoContextCmd{}

	cmd := &cobra.Command{
		Use:   "context [flags] [NAME]",
		Short: "Set default for operator repository context",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("need at least one argument, name of operator repository")
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
