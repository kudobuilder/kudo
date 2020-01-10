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

type Parameter []v1beta1.Parameter

// Templates is a map of file names and stringified files in the template folder of an operator
type Templates map[string]string

// Len returns the number of params.
// This is needed to allow sorting of params.
func (p Parameter) Len() int { return len(p) }

// Swap swaps the position of two items in the params slice.
// This is needed to allow sorting of params.
func (p Parameter) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Less returns true if the name of a param a is less than the name of param b.
// This is needed to allow sorting of params.
func (p Parameter) Less(x, y int) bool {
	return p[x].Name < p[y].Name
}

// Files represents the raw operator package format the way it is found in the tgz packages
type Files struct {
	Templates Templates
	Operator  *OperatorFile
	Params    *ParamsFile
}

// ParamsFile is a representation of the package params.yaml
type ParamsFile struct {
	APIVersion string    `json:"apiVersion,omitempty"`
	Parameters Parameter `json:"parameters"`
}

// OperatorFile is a representation of the package operator.yaml
type OperatorFile struct {
	APIVersion        string                  `json:"apiVersion,omitempty"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description,omitempty"`
	OperatorVersion   string                  `json:"operatorVersion"`
	AppVersion        string                  `json:"appVersion,omitempty"`
	KUDOVersion       string                  `json:"kudoVersion,omitempty"`
	KubernetesVersion string                  `json:"kubernetesVersion,omitempty"`
	Maintainers       []*v1beta1.Maintainer   `json:"maintainers,omitempty"`
	URL               string                  `json:"url,omitempty"`
	Tasks             []v1beta1.Task          `json:"tasks"`
	Plans             map[string]v1beta1.Plan `json:"plans"`
}
