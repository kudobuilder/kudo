package engine

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// CreateTask is a task that attempts to create a set of Kubernetes Resources using a given client
// +k8s:deepcopy-gen=true
type CreateTask struct {
	Resources []runtime.Object
}

// CreateTask Run
func (c *CreateTask) Run() error {
	// run Renderable task (resolves and templates a bunch resources)
	// run Kubernetes task (actually performs the Kubernetes op)
	return nil
}
