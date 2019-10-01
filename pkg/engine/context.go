package engine

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Context struct {
	context.Context
	KubernetesClient client.Client
}
