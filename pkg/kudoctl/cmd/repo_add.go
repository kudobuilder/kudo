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
	repoAddExample = `  kubectl kudo repo add local http://localhost
  # to skip url and index.yaml validation
  kubectl kudo repo add local http://localhost --skip-check
`
)

type repoAddCmd struct {
	name      string
	url       string
	home      kudohome.Home
	skipCheck bool

	out io.Writer
	fs  afero.Fs
}

func (addCmd repoAddCmd) run() error {
	if err := addRepository(addCmd.fs, addCmd.name, addCmd.url, addCmd.home, addCmd.skipCheck); err != nil {
		return err
	}
	fmt.Fprintf(addCmd.out, "%q has been added to your repositories\n", addCmd.name)
	return nil

}

func addRepository(fs afero.Fs, name string, url string, home kudohome.Home, force bool) error {
	repos, err := repo.LoadRepositories(fs, home.RepositoryFile())
	if err != nil {
		return err
	}
	if repos.GetConfiguration(name) != nil {
		return fmt.Errorf("repository name (%s) already exists, please specify a different name", name)
	}
	config := &repo.Configuration{
		URL:  url,
		Name: name,
	}
	client, err := repo.NewClient(config)
	if err != nil {
		return err
	}

	if !force {
		// valid the url and that we can pull and index is valid
		_, err = client.DownloadIndexFile()
		if err != nil {
			return fmt.Errorf("looks like %q is not a valid operator repository or cannot be reached: %s", url, err.Error())
		}
	}
	repos.Add(config)

	return repos.WriteFile(fs, home.RepositoryFile(), 0644)
}

func newRepoAddCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	add := &repoAddCmd{out: out}

	cmd := &cobra.Command{
		Use:     "add [flags] [NAME] [URL]",
		Short:   "Add an operator repository",
		Example: repoAddExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("this command needs 2. name and url of the operator repository")
			}

			add.name = args[0]
			add.url = args[1]
			add.home = Settings.Home
			add.fs = fs

			return add.run()
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&add.skipCheck, "skip-check", "f", false, "Skip URL and index file validation.")

	return cmd
}
