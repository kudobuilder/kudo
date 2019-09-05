package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type repoListCmd struct {
	out  io.Writer
	home kudohome.Home
}

func newRepoListCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &repoListCmd{out: out}

	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List operator repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			list.home = Settings.Home
			return list.run(fs)
		},
	}

	return cmd
}

func (a *repoListCmd) run(fs afero.Fs) error {
	f, err := repo.LoadRepositories(fs, a.home.RepositoryFile())
	if err != nil {
		return err
	}
	if len(f.Repositories) == 0 {
		return errors.New("no repositories to show")
	}
	table := uitable.New()
	table.AddRow("NAME", "URL")
	for _, re := range f.Repositories {
		table.AddRow(re.Name, re.URL)
	}
	fmt.Fprintln(a.out, table)
	return nil
}
