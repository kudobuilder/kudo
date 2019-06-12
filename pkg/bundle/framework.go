package bundle

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

// Bundle represents the package format supported by KUDO
type Bundle struct {
	*Framework
	// 	templates map[string]string temporary removed until we introduce the template normalization
}

// Framework is a representation of the KEP-9 Framework YAML
type Framework struct {
	Name              string                         `json:"name"`
	Description       string                         `json:"description,omitempty"`
	Version           string                         `json:"version"`
	KUDOVersion       string                         `json:"kudoVersion,omitempty"`
	KubernetesVersion string                         `json:"kubernetesVersion,omitempty"`
	Maintainers       []v1alpha1.Maintainer          `json:"maintainers,omitempty"`
	URL               string                         `json:"url,omitempty"`
	Tasks             map[string]v1alpha1.TaskSpec   `json:"tasks"`
	Plans             map[string]v1alpha1.Plan       `json:"plans"`
	Parameters        map[string]Parameter           `json:"parameters,omitempty"`
	Dependencies      []v1alpha1.FrameworkDependency `json:"dependencies,omitempty"`
}

// Parameter is a struct defining a parameter in the current KUDO package format
type Parameter struct {
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	Trigger     string `json:"trigger,omitempty"`
}
