package bundle

import "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

//TODO (kensipe): Switch Maintainers to Repo Maintainer: https://github.com/kudobuilder/kudo/blob/master/pkg/kudoctl/util/repo/repo.go#L43

// Operator is a representation of the KEP-9 Operator YAML
type Operator struct {
	Name              string                        `json:"name"`
	Description       string                        `json:"description,omitempty"`
	Version           string                        `json:"version"`
	AppVersion        string                        `json:"appVersion,omitempty"`
	KUDOVersion       string                        `json:"kudoVersion,omitempty"`
	KubernetesVersion string                        `json:"kubernetesVersion,omitempty"`
	Maintainers       []v1alpha1.Maintainer         `json:"maintainers,omitempty"`
	URL               string                        `json:"url,omitempty"`
	Tasks             map[string]v1alpha1.TaskSpec  `json:"tasks"`
	Plans             map[string]v1alpha1.Plan      `json:"plans"`
	Dependencies      []v1alpha1.OperatorDependency `json:"dependencies,omitempty"`
}
