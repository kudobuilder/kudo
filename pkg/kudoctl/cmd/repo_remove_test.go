package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
)

func TestRemoveWithoutArg(t *testing.T) {

	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	cmd := newRepoRemoveCmd(fs, out)
	err := cmd.RunE(cmd, []string{})
	assert.EqualError(t, err, "need at least one argument, name of operator repository")
}

func TestRemoveWithoutValidName(t *testing.T) {

	// setup (init client)
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

	// reset buffer for repo list
	out = &bytes.Buffer{}

	cmd := repoRemoveCmd{out: out, name: "foo", home: home, fs: fs}
	err = cmd.run()
	assert.EqualError(t, err, `no repo named "foo" found`)
}

func TestRemoveValidName(t *testing.T) {

	// setup (init client)
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

	// reset buffer for repo list
	out = &bytes.Buffer{}

	cmd := repoRemoveCmd{out: out, name: "community", home: home, fs: fs}
	err = cmd.run()
	assert.Nil(t, err)
	assert.Equal(t, out.String(), fmt.Sprintf("%q has been removed from your repositories\n", "community"))
}
