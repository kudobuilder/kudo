package engine

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

type Context struct {
	context.Context
	KubernetesClient kubernetes.Interface
}
