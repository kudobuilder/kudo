package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

type repoListCmd struct {
	out  io.Writer
	home kudohome.Home
}

func newRepoListCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &repoListCmd{out: out}

	cmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   "List operator repositories",
		Example: "  kubectl kudo repo list",
		RunE: func(cmd *cobra.Command, args []string) error {
			list.home = Settings.Home
			return list.run(fs)
		},
	}

	return cmd
}

func (a *repoListCmd) run(fs afero.Fs) error {
	repos, err := repo.LoadRepositories(fs, a.home.RepositoryFile())
	if err != nil {
		return err
	}
	if len(repos.Repositories) == 0 {
		return errors.New("no repositories to show")
	}
	table := uitable.New()
	table.AddRow("NAME", "URL")
	for _, re := range repos.Repositories {
		if re.Name == repos.Context {
			table.AddRow(fmt.Sprintf("*%s", re.Name), re.URL)
		} else {
			table.AddRow(re.Name, re.URL)
		}
	}
	fmt.Fprintln(a.out, table)
	return nil
}
