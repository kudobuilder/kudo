package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/gofrs/flock"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type repoAddCmd struct {
	name string
	url  string
	home kudohome.Home

	out io.Writer
	fs  afero.Fs
}

func (a repoAddCmd) run() error {
	if err := addRepository(a.fs, a.name, a.url, a.home); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "%q has been added to your repositories\n", a.name)
	return nil

}

func addRepository(fs afero.Fs, name string, url string, home kudohome.Home) error {
	f, err := repo.LoadRepositories(fs, home.RepositoryFile())
	if err != nil {
		return err
	}
	if f.GetRepo(name) != nil {
		return fmt.Errorf("repository name (%s) already exists, please specify a different name", name)
	}
	rc := &repo.RepositoryConfiguration{
		URL:  url,
		Name: name,
	}
	or, err := repo.NewOperatorRepository(rc)
	if err != nil {
		return err
	}
	_, err = or.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("looks like %q is not a valid operator repository or cannot be reached: %s", url, err.Error())
	}
	// lock file
	fLock := flock.New(home.RepositoryFile())
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fLock.Unlock()
	}
	if err != nil {
		return err
	}
	// re-read repositories after lock
	f, err = repo.LoadRepositories(fs, home.RepositoryFile())
	if err != nil {
		return err
	}
	f.Add(rc)
	for _, repo := range f.Repositories {
		fmt.Printf("repos %v\n", repo)
	}

	return f.WriteFile(fs, home.RepositoryFile(), 0644)
}

func newRepoAddCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	add := &repoAddCmd{out: out}

	cmd := &cobra.Command{
		Use:   "add [flags] [NAME] [URL]",
		Short: "Add an operator repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkArgsLength(len(args), "name for the operator repository", "the url of the operator repository"); err != nil {
				return err
			}

			add.name = args[0]
			add.url = args[1]
			add.home = Settings.Home
			add.fs = fs

			return add.run()
		},
	}
	return cmd
}

func checkArgsLength(argsReceived int, requiredArgs ...string) error {
	expectedNum := len(requiredArgs)
	if argsReceived != expectedNum {
		arg := "arguments"
		if expectedNum == 1 {
			arg = "argument"
		}
		return fmt.Errorf("this command needs %v %s: %s", expectedNum, arg, strings.Join(requiredArgs, ", "))
	}
	return nil
}
