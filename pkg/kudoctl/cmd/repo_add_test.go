package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
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
	if err := i.ensureClient(); err != nil {
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
	errOut := &bytes.Buffer{}

	home := kudohome.Home("kudo_home")
	err := fs.Mkdir(home.String(), 0755)
	if err != nil {
		t.Fatal(err)
	}
	i := &initCmd{fs: fs, out: out, errOut: errOut, home: home}
	if err := i.ensureClient(); err != nil {
		t.Error(err)
	}

	cmd := &repoAddCmd{fs: fs, out: out, home: home, name: "foo", url: "badurl"}
	err = cmd.run()
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), `looks like "badurl" is not a valid operator repository or cannot be reached:`))
}

func TestAddSkipCheck(t *testing.T) {
	file := "skip-check-add-list"
	//	setup
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	home := kudohome.Home("kudo_home")
	err := fs.Mkdir(home.String(), 0755)
	if err != nil {
		t.Fatal(err)
	}
	i := &initCmd{fs: fs, out: out, errOut: errOut, home: home}
	if err := i.ensureClient(); err != nil {
		t.Error(err)
	}

	// skipCheck add under test
	cmd := &repoAddCmd{fs: fs, out: out, home: home, name: "foo", url: "badurl", skipCheck: true}
	err = cmd.run()
	assert.Nil(t, err)

	// verified through repo list golden file
	out = &bytes.Buffer{}
	rl := &repoListCmd{out: out, home: home}
	if err := rl.run(fs); err != nil {
		t.Fatal(err)
	}
	gp := filepath.Join("testdata", file+".golden")

	if *updateGolden {
		t.Log("update golden file")

		//nolint:gosec
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

func TestAddArgValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	var tests = []struct {
		name         string
		args         []string
		errorMessage string
	}{
		{name: "no valid arguments", args: []string{}, errorMessage: "this command needs 2. name and url of the operator repository"},
		{name: "missing 1 argument", args: []string{"arg1"}, errorMessage: "this command needs 2. name and url of the operator repository"},
	}

	for _, test := range tests {
		cmd := newRepoAddCmd(fs, out)
		err := cmd.RunE(cmd, test.args)
		assert.EqualError(t, err, test.errorMessage)
	}
}
