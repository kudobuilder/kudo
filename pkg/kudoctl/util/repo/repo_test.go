package repo

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLoadRepositoriesErrorHandling(t *testing.T) {

	fs := afero.NewMemMapFs()
	path := "/opt"

	_, err := LoadRepositories(fs, path)
	contains := "You might need to run `kudo init` (or `kudo init --client-only` if kudo is already installed)"
	fmt.Println(err.Error())
	assert.True(t, strings.Contains(err.Error(), contains))
}

func TestLoadRepositories(t *testing.T) {
	file := "repositories.yaml"
	fs := afero.NewOsFs()
	gp := filepath.Join("testdata", file+".golden")

	if *update {
		t.Log("update golden file")
		r := NewRepositories()

		if err := r.WriteFile(fs, gp, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	r, err := LoadRepositories(fs, gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, r.CurrentConfiguration().Name, Default.Name)
	assert.Equal(t, r.CurrentConfiguration().URL, Default.URL)
}

func TestDownloadMultiRepo(t *testing.T) {

	p, err := filepath.Abs("testdata/include-index")
	assert.NoError(t, err)
	config := &Configuration{
		URL: fmt.Sprintf("file://%s", p),
	}
	client, err := NewClient(config)
	assert.NoError(t, err)
	index, err := client.DownloadIndexFile()
	assert.NoError(t, err)
	// mysql package only there, if include worked
	assert.NotNil(t, index.Entries["mysql"])

	// the merge for flink will have 1 dup that doesn't merge
	flink, err := index.FindFirstMatch("flink", "", "0.3.0")
	assert.NoError(t, err)
	assert.Equal(t, "correct flink", flink.Description)

	// and a new version that does merge
	flink, err = index.FindFirstMatch("flink", "", "0.4.0")
	assert.NoError(t, err)
	assert.Equal(t, "0.4.0", flink.OperatorVersion)

	// and a version that is nested in a repository
	flink, err = index.FindFirstMatch("flink", "", "0.4.1")
	assert.NoError(t, err)
	assert.Equal(t, "0.4.1", flink.OperatorVersion)
	assert.Equal(t, "this merges and overwrites the version in nested-included-repo", flink.Description)

}
