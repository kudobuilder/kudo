package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

const (
	repoRemoveExample = `  kubectl kudo repo remove local
  kubectl kudo repo rm local`
)

type repoRemoveCmd struct {
	out  io.Writer
	name string
	home kudohome.Home
	fs   afero.Fs
}

func newRepoRemoveCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	remove := &repoRemoveCmd{out: out}

	cmd := &cobra.Command{
		Use:     "remove [flags] [NAME]",
		Aliases: []string{"rm"},
		Short:   "Remove an operator repository",
		Example: repoRemoveExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("need at least one argument, name of operator repository")
			}

			remove.home = Settings.Home
			remove.fs = fs
			for i := 0; i < len(args); i++ {
				remove.name = args[i]
				if err := remove.run(); err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}

func (r *repoRemoveCmd) run() error {
	return removeRepoLine(r.fs, r.out, r.name, r.home)
}

func removeRepoLine(fs afero.Fs, out io.Writer, name string, home kudohome.Home) error {
	repos, err := repo.LoadRepositories(fs, home.RepositoryFile())
	if err != nil {
		return err
	}

	if !repos.Remove(name) {
		return fmt.Errorf("no repo named %q found", name)
	}
	if err := repos.WriteFile(fs, home.RepositoryFile(), 0644); err != nil {
		return err
	}

	fmt.Fprintf(out, "%q has been removed from your repositories\n", name)

	return nil
}
