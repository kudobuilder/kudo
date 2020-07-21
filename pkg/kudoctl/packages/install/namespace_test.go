package install

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamespaceManifestRendering(t *testing.T) {
	namespaceFile := "testdata/namespace.yaml"

	ns, err := ioutil.ReadFile(namespaceFile)
	assert.NoError(t, err)

	params := make(map[string]string)
	params["foo"] = "bar-param"

	rendered, err := renderNamespaceManifest(string(ns), testResources(), params)
	assert.NoError(t, err)

	file := "rendered-namespace.yaml"
	gf := filepath.Join("testdata", file+".golden")

	if *update {
		t.Log("update golden file")

		//nolint:gosec
		if err := ioutil.WriteFile(gf, []byte(rendered), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}

	golden, err := ioutil.ReadFile(gf)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, string(golden), rendered, "for golden file: %s", gf)
}
