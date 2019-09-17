package finder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetPackage(t *testing.T) {
	m := &Manager{
		local: NewLocal(),
		uri:   nil,
	}
	b, err := m.GetPackage("../testdata/zk", "")
	if err != nil {
		t.Errorf("Manager.GetPackage() error = %v", err)
		return
	}

	crd, err := b.GetCRDs()
	if err != nil {
		t.Errorf("packages.GetCRDs error = %v", err)
	}
	assert.EqualValues(t, "zookeeper", crd.Operator.Name)
}

func TestLocalFinder_GetPackage(t *testing.T) {
	f := NewLocal()
	b, err := f.GetPackage("../testdata/zk", "")
	if err != nil {
		t.Errorf("Manager.GetPackage() error = %v", err)
		return
	}

	crd, err := b.GetCRDs()
	if err != nil {
		t.Errorf("packages.GetCRDs error = %v", err)
	}
	assert.EqualValues(t, "zookeeper", crd.Operator.Name)
}

func TestLocalFinder_Failure(t *testing.T) {
	f := NewLocal()
	_, err := f.GetPackage("../testdata/zk-bad", "")
	assert.Errorf(t, err, "should have errored on bad folder name")
}
