package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			time := time.Now()
			riCmd := newRepoIndexCmd(fs, out, &time)
			for key, value := range tt.flags {
				riCmd.Flags().Set(key, value)
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
	fs.Mkdir(testdir, 0777)
	files.CopyOperatorToFs(fs, "../packages/testdata/zk.tgz", "/opt")

	time, _ := time.Parse(time.RFC3339, "2019-10-25T00:00:00Z")
	out := &bytes.Buffer{}
	riCmd := newRepoIndexCmd(fs, out, &time)
	riCmd.RunE(riCmd, []string{"/opt"})

	indexOut, err := afero.ReadFile(fs, "/opt/index.yaml")
	if err != nil {
		t.Fatalf("failed reading index file: %s", err)
	}
	gp := filepath.Join("testdata", file+".golden")

	if *updateGolden {
		t.Log("update golden file")
		if err := ioutil.WriteFile(gp, indexOut, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	if !bytes.Equal(indexOut, g) {
		t.Errorf("yaml does not match .golden file")
	}
}

func TestRepoIndexCmd_MergeIndex(t *testing.T) {
	file := "merge-index.yaml"

	indexBytes, _ := ioutil.ReadFile("testdata/index.yaml")
	indexFile, _ := repo.ParseIndexFile(indexBytes)

	mergeBytes, _ := ioutil.ReadFile("testdata/merge.yaml")
	mergeFile, _ := repo.ParseIndexFile(mergeBytes)

	resultBuf := &bytes.Buffer{}
	merge(indexFile, mergeFile)
	indexFile.Write(resultBuf)

	gp := filepath.Join("testdata", file+".golden")

	if *updateGolden {
		t.Log("update golden file")
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
	o, _ := indexFile.GetByNameAndVersion("mysql", "0.1.0")
	assert.Equal(t, o.Maintainers[0].Name, "Ken Sipe")
}
