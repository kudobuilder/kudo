package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

func TestRepoIndexCmd(t *testing.T) {
	fs := afero.NewMemMapFs()
	var tests = []struct {
		name         string
		flags        map[string]string
		arguments    []string
		errorMessage string
	}{
		{name: "no arguments", errorMessage: "expecting exactly one argument - directory containing the operators to package"},
		{name: "too many arguments", arguments: []string{"foo", "bar"}, errorMessage: "expecting exactly one argument - directory containing the operators to package"},
		{name: "merge and merge-repo together invalid", arguments: []string{""}, flags: map[string]string{"merge": "foo", "merge-repo": "bar"}, errorMessage: "specify either 'merge' or 'merge-repo', not both"},
		{name: "merge and merge-repo together invalid", arguments: []string{""}, flags: map[string]string{"url": "foo", "url-repo": "bar"}, errorMessage: "specify either 'url' or 'url-repo', not both"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			time := time.Now()
			riCmd := newRepoIndexCmd(fs, out, &time)
			for key, value := range tt.flags {
				if err := riCmd.Flags().Set(key, value); err != nil {
					t.Fatalf("%s: %v", tt.name, err)
				}
			}
			err := riCmd.RunE(riCmd, tt.arguments)
			assert.EqualError(t, err, tt.errorMessage)
		})
	}
}

func TestRepoIndexCmd_IndexCreation(t *testing.T) {
	file := "index.yaml"
	fs := afero.NewMemMapFs()
	testdir, _ := filepath.Abs("")
	if err := fs.Mkdir(testdir, 0777); err != nil {
		t.Fatal(err)
	}
	files.CopyOperatorToFs(fs, "../packages/testdata/zk.tgz", "/opt")

	time, _ := time.Parse(time.RFC3339, "2019-10-25T00:00:00Z")
	out := &bytes.Buffer{}
	riCmd := newRepoIndexCmd(fs, out, &time)
	if err := riCmd.RunE(riCmd, []string{"/opt"}); err != nil {
		t.Fatal(err)
	}

	indexOut, err := afero.ReadFile(fs, "/opt/index.yaml")
	if err != nil {
		t.Fatalf("failed reading index file: %s", err)
	}
	gp := filepath.Join("testdata", file+".golden")

	if *updateGolden {
		t.Log("update golden file")

		//nolint:gosec
		if err := ioutil.WriteFile(gp, indexOut, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, string(indexOut), string(g), "yaml does not match .golden file %s", gp)
}

func TestRepoIndexCmd_MergeIndex(t *testing.T) {
	file := "merge-index.yaml"

	indexBytes, _ := ioutil.ReadFile("testdata/index.yaml")
	indexFile, _ := repo.ParseIndexFile(indexBytes)

	mergeBytes, _ := ioutil.ReadFile("testdata/merge.yaml")
	mergeFile, _ := repo.ParseIndexFile(mergeBytes)

	resultBuf := &bytes.Buffer{}
	merge(indexFile, mergeFile)
	if err := indexFile.Write(resultBuf); err != nil {
		t.Fatal(err)
	}

	gp := filepath.Join("testdata", file+".golden")

	if *updateGolden {
		t.Log("update golden file")

		//nolint:gosec
		if err := ioutil.WriteFile(gp, resultBuf.Bytes(), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	if !bytes.Equal(resultBuf.Bytes(), g) {
		t.Errorf("yaml does not match .golden file")
	}

	// local operator takes precedence
	o, _ := indexFile.FindFirstMatch("mysql", "5.7", "0.1.0")
	assert.Equal(t, o.Maintainers[0].Name, "Ken Sipe")
}
