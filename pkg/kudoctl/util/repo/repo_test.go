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
