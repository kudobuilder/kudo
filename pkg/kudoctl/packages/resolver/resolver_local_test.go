package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalResolver_GetPackage(t *testing.T) {
	f := NewLocal()
	pkg, err := f.Resolve("../testdata/zk", "")
	if err != nil {
		t.Errorf("PackageResolver.Resolve() error = %v", err)
		return
	}

	assert.EqualValues(t, "zookeeper", pkg.Resources.Operator.Name)
}

func TestLocalFinder_Failure(t *testing.T) {
	f := NewLocal()
	_, err := f.Resolve("../testdata/zk-bad", "")
	assert.Errorf(t, err, "should have errored on bad folder name")
}
