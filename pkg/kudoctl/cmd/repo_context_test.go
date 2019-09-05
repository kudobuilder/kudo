package cmd

import (
	"bytes"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"os"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestRepoContextNoArg(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := newRepoContextCmd(fs)
	err := cmd.RunE(cmd, []string{})
	assert.EqualError(t, err, "need at least one argument, name of operator repository")
}

func TestRepoContextInvalidArg(t *testing.T) {
	// setup (init client)
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	home := kudohome.Home("kudo_home")
	err := fs.Mkdir(home.String(), 0755)
	if err != nil {
		t.Fatal(err)
	}

	i := &initCmd{fs: fs, out: out, home: home}
	if err := i.initialize(); err != nil {
		t.Error(err)
	}
	cmd := repoContextCmd{name: "foo", home: home, fs: fs}
	err = cmd.run()
	assert.EqualError(t, err, `no repo named "foo" found`)
}

func TestRepoContextSwitch(t *testing.T) {
	// There is a bug in afero I fought for some time.. results in a PathError golang.org/x/sys/unix.ENOENT (2)
	// Issue is in the last write of the file in the cmd.run()
	// everything in code looked correct and local manual testing worked.  Switching to temp folder
	file := os.TempDir()
	defer os.RemoveAll(file)

	fs := afero.NewOsFs()
	out := &bytes.Buffer{}

	home := kudohome.Home(file)

	// start with initialized client
	i := &initCmd{fs: fs, out: out, home: home}
	if err := i.initialize(); err != nil {
		t.Error(err)
	}

	// add a repo config
	repos := repo.NewRepositories()
	fooConfig := &repo.Configuration{URL: "test", Name: "foo"}
	repos.Add(fooConfig)

	err := repos.WriteFile(fs, home.RepositoryFile(), 0644)
	if err != nil {
		t.Error(err)
	}

	// test setting to it
	cmd := repoContextCmd{name: "foo", home: home, fs: fs}
	err = cmd.run()
	assert.Nil(t, err)

	// re-read repositories.yaml
	repos, err = repo.LoadRepositories(fs, home.RepositoryFile())
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, repos.Context, "foo")
}
