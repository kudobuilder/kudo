package bundle

import (
	"github.com/Masterminds/semver"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

// Framework is a representation of the KEP-9 Framework YAML
type Framework struct {
	Name              string                         `json:"name"`
	Description       string                         `json:"description,omitempty"`
	Version           string                         `json:"version"`
	KUDOVersion       semver.Version                         `json:"kudoVersion,omitempty"`
	KubernetesVersion semver.Version                         `json:"kubernetesVersion,omitempty"`
	Maintainers       []v1alpha1.Maintainer          `json:"maintainers,omitempty"`
	URL               string                         `json:"url,omitempty"`
	Tasks             map[string]v1alpha1.TaskSpec   `json:"tasks"`
	Plans             map[string]v1alpha1.Plan       `json:"plans"`
	Parameters        []v1alpha1.Parameter           `json:"parameters,omitempty"`
	Dependencies      []v1alpha1.FrameworkDependency `json:"dependencies,omitempty"`
}
