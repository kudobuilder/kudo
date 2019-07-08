package finder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetBundle(t *testing.T) {
	m := &Manager{
		local: NewLocal(),
		uri:   nil,
	}
	b, err := m.GetBundle("../testdata/zk", "")
	if err != nil {
		t.Errorf("Manager.GetBundle() error = %v", err)
		return
	}

	crd, err := b.GetCRDs()
	if err != nil {
		t.Errorf("bundle.GetCRDs error = %v", err)
	}
	assert.EqualValues(t, "zookeeper", crd.Operator.Name)
}

func TestLocalFinder_GetBundle(t *testing.T) {
	f := NewLocal()
	b, err := f.GetBundle("../testdata/zk", "")
	if err != nil {
		t.Errorf("Manager.GetBundle() error = %v", err)
		return
	}

	crd, err := b.GetCRDs()
	if err != nil {
		t.Errorf("bundle.GetCRDs error = %v", err)
	}
	assert.EqualValues(t, "zookeeper", crd.Operator.Name)
}

func TestLocalFinder_Failure(t *testing.T) {
	f := NewLocal()
	_, err := f.GetBundle("../testdata/zk-bad", "")
	assert.Errorf(t, err, "should have errored on bad folder name")
}
