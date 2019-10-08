package task

import (
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Context struct {
	Client     client.Client
	Enhancer   instance.KubernetesObjectEnhancer
	Meta       instance.ExecutionMetadata
	Templates  map[string]string // Raw templates
	Parameters map[string]string // I and OV parameters merged
}
