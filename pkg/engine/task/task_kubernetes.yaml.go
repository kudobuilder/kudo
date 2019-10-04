package task

import (
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	OpApply string = "apply"
)

type KubernetesTask struct {
	Op        string
	Resources []runtime.Object
}

func (k *KubernetesTask) Run(ctx Context) error {
	return nil
	// setup Kubernetes client
}
