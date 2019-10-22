package test

import (
	"context"
	"fmt"
	"testing"

	kindConfig "sigs.k8s.io/kind/pkg/apis/config/v1alpha3"

	dockertypes "github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/assert"
)

func TestGetTimeout(t *testing.T) {
	h := Harness{}
	assert.Equal(t, 30, h.GetTimeout())

	h.TestSuite.Timeout = 45
	assert.Equal(t, 45, h.GetTimeout())
}

type dockerMock struct{}

func (d *dockerMock) VolumeCreate(ctx context.Context, body volumetypes.VolumeCreateBody) (dockertypes.Volume, error) {
	return dockertypes.Volume{
		Mountpoint: fmt.Sprintf("/var/lib/docker/data/%s", body.Name),
	}, nil
}

func (d *dockerMock) NegotiateAPIVersion(ctx context.Context) {}

func TestAddNodeCaches(t *testing.T) {
	h := Harness{
		T:      t,
		docker: &dockerMock{},
	}

	kindCfg := &kindConfig.Cluster{}
	h.addNodeCaches(kindCfg)
	assert.Nil(t, kindCfg.Nodes)

	h.TestSuite.KINDNodeCache = true
	h.addNodeCaches(kindCfg)
	assert.NotNil(t, kindCfg.Nodes)
	assert.Equal(t, 1, len(kindCfg.Nodes))
	assert.NotNil(t, kindCfg.Nodes[0].ExtraMounts)
	assert.Equal(t, 1, len(kindCfg.Nodes[0].ExtraMounts))
	assert.Equal(t, "/var/lib/containerd", kindCfg.Nodes[0].ExtraMounts[0].ContainerPath)
	assert.Equal(t, "/var/lib/docker/data/kind-0", kindCfg.Nodes[0].ExtraMounts[0].HostPath)

	kindCfg = &kindConfig.Cluster{
		Nodes: []kindConfig.Node{
			{},
			{},
		},
	}
	h.addNodeCaches(kindCfg)
	assert.NotNil(t, kindCfg.Nodes)
	assert.Equal(t, 2, len(kindCfg.Nodes))
	assert.NotNil(t, kindCfg.Nodes[0].ExtraMounts)
	assert.Equal(t, 1, len(kindCfg.Nodes[0].ExtraMounts))
	assert.Equal(t, "/var/lib/containerd", kindCfg.Nodes[0].ExtraMounts[0].ContainerPath)
	assert.Equal(t, "/var/lib/docker/data/kind-0", kindCfg.Nodes[0].ExtraMounts[0].HostPath)
	assert.Equal(t, "/var/lib/docker/data/kind-1", kindCfg.Nodes[1].ExtraMounts[0].HostPath)
}
