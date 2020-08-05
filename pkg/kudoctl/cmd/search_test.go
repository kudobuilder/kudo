package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

func TestSearchArgValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	var tests = []struct {
		name         string
		args         []string
		errorMessage string
	}{
		{name: "no valid arguments", args: []string{}, errorMessage: "this command must have only 1 search criterion"},
		{name: "2 arguments", args: []string{"arg1", "arg2"}, errorMessage: "this command must have only 1 search criterion"},
	}

	for _, test := range tests {
		cmd := newSearchCmd(fs, out)
		err := cmd.RunE(cmd, test.args)
		assert.EqualError(t, err, test.errorMessage)
	}
}

func TestSearchIndex(t *testing.T) {
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	abs, _ := filepath.Abs("testdata")
	c := repo.Configuration{
		URL:  fmt.Sprintf("file://%s", abs),
		Name: "testdata",
	}

	r, err := repo.NewClient(&c)
	assert.NoError(t, err)
	search := searchCmd{
		out:         out,
		fs:          fs,
		repoName:    "",
		home:        "",
		allVersions: false,
		repoClient:  r,
	}

	// search for mysql output
	err = search.run("mysql")
	assert.NoError(t, err)

	gp := filepath.Join("testdata", "search1"+".golden")

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

	// assert mysql search
	assert.Equal(t, out.String(), string(g), "cmd output does not match .golden file %s", gp)
	out.Reset()

	// search for "" output, which is all operators
	err = search.run("")
	assert.NoError(t, err)

	gp = filepath.Join("testdata", "search2"+".golden")

	if *updateGolden {
		t.Log("update golden file")

		//nolint:gosec
		if err := ioutil.WriteFile(gp, out.Bytes(), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err = ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	// assert "" search
	assert.Equal(t, out.String(), string(g), "cmd output does not match .golden file %s", gp)
	out.Reset()

	search.allVersions = true
	// search for mysql output again with all versions
	err = search.run("mysql")
	assert.NoError(t, err)

	gp = filepath.Join("testdata", "search3"+".golden")

	if *updateGolden {
		t.Log("update golden file")

		//nolint:gosec
		if err := ioutil.WriteFile(gp, out.Bytes(), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err = ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	// assert mysql search
	assert.Equal(t, out.String(), string(g), "cmd output does not match .golden file %s", gp)
}
