package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
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
	cmd := repoContextCmd{name: "foo", home: home, fs: fs}
	err = cmd.run()
	assert.EqualError(t, err, `no repo named "foo" found`)
}
