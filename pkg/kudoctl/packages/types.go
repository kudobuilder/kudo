package packages

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

const (
	APIVersion = "kudo.dev/v1beta1"
)

// This is an abstraction which abstracts the underlying packages, which is likely file system or compressed file.
// There should be a complete separation between retrieving a packages if not local and working with a packages.

// Package is an abstraction of the collection of files that makes up a package.  It is anything we can retrieve the Resources from.
type Package struct {
	// transformed server view
	Resources *Resources
	// working with local package files
	Files *Files
}

// Resources is collection of CRDs that are used when installing operator
// during installation, package format is converted to this structure
type Resources struct {
	Operator        *v1beta1.Operator
	OperatorVersion *v1beta1.OperatorVersion
	Instance        *v1beta1.Instance
}

// Files represents the raw operator package format the way it is found in the tgz packages
type Files struct {
	Templates map[string]string
	Operator  *Operator
	Params    []v1beta1.Parameter
}

type ParametersFile struct {
	APIVersion string              `json:"apiVersion,omitempty"`
	Params     []v1beta1.Parameter `json:"parameters"`
}

// Operator is a representation of the KEP-9 Operator YAML
type Operator struct {
	APIVersion        string                  `json:"apiVersion,omitempty"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description,omitempty"`
	Version           string                  `json:"version"`
	AppVersion        string                  `json:"appVersion,omitempty"`
	KUDOVersion       string                  `json:"kudoVersion,omitempty"`
	KubernetesVersion string                  `json:"kubernetesVersion,omitempty"`
	Maintainers       []*v1beta1.Maintainer   `json:"maintainers,omitempty"`
	URL               string                  `json:"url,omitempty"`
	Tasks             []v1beta1.Task          `json:"tasks"`
	Plans             map[string]v1beta1.Plan `json:"plans"`
}
