package bundle

import "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

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
	Dependencies      []v1alpha1.FrameworkDependency `json:"dependencies,omitempty"`
}

type Parameter struct {
	Name        string
	Default     string
	Description string
}
