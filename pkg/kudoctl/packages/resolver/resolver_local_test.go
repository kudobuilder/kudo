package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalResolver_GetPackage(t *testing.T) {
	f := NewLocalHelper()
	pkg, err := f.ResolveDir("../testdata/zk")
	if err != nil {
		t.Errorf("PackageResolver.Resolve() error = %v", err)
		return
	}

	assert.EqualValues(t, "zookeeper", pkg.Operator.Name)
}

func TestLocalFinder_Failure(t *testing.T) {
	f := NewLocalHelper()
	_, err := f.ResolveDir("../testdata/zk-bad")
	assert.Errorf(t, err, "should have errored on bad folder name")
}
