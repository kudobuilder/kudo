package resolver

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLocalResolver_GetPackage(t *testing.T) {
	wd, _ := os.Getwd()
	f := newForFilesystem(afero.NewOsFs(), wd)
	pkg, err := f.ResolveDir("../testdata/zk")
	if err != nil {
		t.Errorf("PackageResolver.Resolve() error = %v", err)
		return
	}

	assert.EqualValues(t, "zookeeper", pkg.Operator.Name)
}

func TestLocalFinder_Failure(t *testing.T) {
	wd, _ := os.Getwd()
	f := newForFilesystem(afero.NewOsFs(), wd)
	_, err := f.ResolveDir("../testdata/zk-bad")
	assert.Errorf(t, err, "should have errored on bad folder name")
}
