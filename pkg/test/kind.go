package test

import (
	"context"

	"sigs.k8s.io/kind/pkg/apis/config/v1alpha3"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

// kind provides a thin abstraction layer for a KIND cluster.
type kind struct {
	Provider     *cluster.Provider
	context      string
	explicitPath string
}

func newKind(kindContext string, explicitPath string) kind {
	provider := cluster.NewProvider()

	return kind{
		Provider:     provider,
		context:      kindContext,
		explicitPath: explicitPath,
	}
}

// Run starts a KIND cluster from a given configuration.
func (k *kind) Run(config *v1alpha3.Cluster) error {
	return k.Provider.Create(
		k.context,
		cluster.CreateWithV1Alpha3Config(config),
		cluster.CreateWithKubeconfigPath(k.explicitPath),
	)
}

// IsRunning checks if a KIND cluster is already running for the current context.
func (k *kind) IsRunning() bool {
	contexts, err := k.Provider.List()
	if err != nil {
		panic(err)
	}

	for _, context := range contexts {
		if context == k.context {
			return true
		}
	}

	return false
}

// AddContainers loads the named Docker containers into a KIND cluster.
// The cluster must be running for this to work.
func (k *kind) AddContainers(docker testutils.DockerClient, containers []string) error {
	if !k.IsRunning() {
		panic("KIND cluster isn't running")
	}

	nodes, err := k.Provider.ListNodes(k.context)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		for _, container := range containers {
			if err := loadContainer(docker, node, container); err != nil {
				return err
			}
		}
	}

	return nil
}

// CollectLogs saves the cluster logs to a directory.
func (k *kind) CollectLogs(dir string) error {
	return k.Provider.CollectLogs(k.context, dir)
}

// Stop stops the KIND cluster.
func (k *kind) Stop() error {
	return k.Provider.Delete(k.context, k.explicitPath)
}

func loadContainer(docker testutils.DockerClient, node nodes.Node, container string) error {
	image, err := docker.ImageSave(context.TODO(), []string{container})
	if err != nil {
		return err
	}

	defer image.Close()

	if err := nodeutils.LoadImageArchive(node, image); err != nil {
		return err
	}

	return nil
}
