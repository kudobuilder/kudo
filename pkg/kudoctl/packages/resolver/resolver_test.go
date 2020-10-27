package resolver

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestManager_GetPackage(t *testing.T) {
	wd, _ := os.Getwd()

	m := &PackageResolver{
		local: newForFilesystem(afero.NewOsFs(), wd),
		uri:   nil,
	}
	pr, err := m.Resolve("../testdata/zk", "", "")
	if err != nil {
		t.Errorf("PackageResolver.Resolve() error = %v", err)
		return
	}

	assert.EqualValues(t, "zookeeper", pr.Resources.Operator.Name)
}
