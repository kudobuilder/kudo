package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetPackage(t *testing.T) {
	m := &PackageResolver{
		local: NewLocal(),
		uri:   nil,
	}
	pr, err := m.Resolve("../testdata/zk", "", "")
	if err != nil {
		t.Errorf("PackageResolver.Resolve() error = %v", err)
		return
	}

	assert.EqualValues(t, "zookeeper", pr.Operator.Name)
}
