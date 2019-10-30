package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/spf13/afero"
)

func TestRepoList(t *testing.T) {

	// setup (init client)
	file := "repo-list"
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

	// reset buffer for repo list
	out = &bytes.Buffer{}
	rl := &repoListCmd{out: out, home: home}
	if err := rl.run(fs); err != nil {
		t.Fatal(err)
	}
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
