package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestAddDupRepo(t *testing.T) {
	//	setup
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

	//	test
	cmd := &repoAddCmd{fs: fs, out: out, home: home, name: "community", url: "doesn't matter"}
	err = cmd.run()
	assert.EqualError(t, err, "repository name (community) already exists, please specify a different name")
}

func TestAddBadURLRepo(t *testing.T) {
	//	setup
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

	cmd := &repoAddCmd{fs: fs, out: out, home: home, name: "foo", url: "badurl"}
	err = cmd.run()
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), `looks like "badurl" is not a valid operator repository or cannot be reached:`))
}

func TestForceAdd(t *testing.T) {
	file := "force-add-list"
	//	setup
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

	// force add under test
	cmd := &repoAddCmd{fs: fs, out: out, home: home, name: "foo", url: "badurl", force: true}
	err = cmd.run()
	assert.Nil(t, err)

	// verified through repo list golden file
	out = &bytes.Buffer{}
	rl := &repoListCmd{out: out, home: home}
	rl.run(fs)
	gp := filepath.Join("testdata", file+".golden")

	if *updateGolden {
		t.Log("update golden file")
		if err := ioutil.WriteFile(gp, out.Bytes(), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	if !bytes.Equal(out.Bytes(), g) {
		t.Errorf("json does not match .golden file")
	}
}

func TestAddWithoutInit(t *testing.T) {
	//	setup
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	home := kudohome.Home("kudo_home")

	cmd := &repoAddCmd{fs: fs, out: out, home: home, name: "foo", url: "badurl"}
	err := cmd.run()
	assert.EqualError(t, err, "could not load repositories file (kudo_home/repository/repositories.yaml).\nYou might need to run `kudo init` (or `kudo init --client-only` if kudo is already installed)")
}
