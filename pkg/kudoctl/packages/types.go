package packages

import (
	"fmt"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

const (
	APIVersion = "kudo.dev/v1beta1"
)

// Resolver will try to resolve a given package name to either local tarball, folder, remote url or
// an operator in the remote repository.
type Resolver interface {
	Resolve(name string, appVersion string, operatorVersion string) (*PackageScope, error)
}

// PackageScope type provides operator resources together with a "scope-aware" dependency resolver.
// For example, a dependency like `./child-operator` in an local directory needs the host system to be resolved, but when
// found in a tarball, it needs the contents of the tarball because the dependency is not allowed to "escape" into the
// host system.
type PackageScope struct {
	Resources            *Resources
	DependenciesResolver Resolver
}

// Resources is collection of CRDs that are used when installing operator
// during installation, package format is converted to this structure
type Resources struct {
	Operator        *kudoapi.Operator
	OperatorVersion *kudoapi.OperatorVersion
	Instance        *kudoapi.Instance
}

func (p *Resources) OperatorName() string {
	if p == nil || p.Operator == nil {
		return ""
	}
	return p.Operator.Name
}

func (p *Resources) OperatorVersionString() string {
	if p == nil || p.OperatorVersion == nil {
		return ""
	}
	return p.OperatorVersion.Spec.Version
}

func (p *Resources) AppVersionString() string {
	if p == nil || p.OperatorVersion == nil {
		return ""
	}
	return p.OperatorVersion.Spec.AppVersion
}

// Modified kudoapi.Parameter that allows for defaults provided as YAML.
type Parameter struct {
	DisplayName string                `json:"displayName,omitempty"`
	Name        string                `json:"name,omitempty"`
	Description string                `json:"description,omitempty"`
	Required    *bool                 `json:"required,omitempty"`
	Default     interface{}           `json:"default,omitempty"`
	Trigger     string                `json:"trigger,omitempty"`
	Type        kudoapi.ParameterType `json:"type,omitempty"`
	Immutable   *bool                 `json:"immutable,omitempty"`
	Enum        *[]interface{}        `json:"enum,omitempty"`

	// The following fields are descriptive only and are not used in the OperatorVersion. They are only used on the
	// package level and are not converted to the CRDs, as they are only used during installation of an operator and
	// are not necessary server-side.
	Group    string `json:"group,omitempty"`
	Advanced *bool  `json:"advanced,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

func (p Parameter) IsImmutable() bool {
	return p.Immutable != nil && *p.Immutable
}

func (p Parameter) IsRequired() bool {
	return p.Required != nil && *p.Required
}

func (p Parameter) IsAdvanced() bool {
	return p.Advanced != nil && *p.Advanced
}

func (p Parameter) IsEnum() bool {
	return p.Enum != nil
}

func (p *Parameter) HasDefault() bool {
	return p.Default != nil
}

func (p *Parameter) ValidateDefault() error {
	if err := kudoapi.ValidateParameterValueForType(p.Type, p.Default); err != nil {
		return fmt.Errorf("parameter \"%s\" has an invalid default value: %v", p.Name, err)
	}
	if p.IsEnum() {
		for _, eValue := range p.EnumValues() {
			if p.Default == eValue {
				return nil
			}
		}
		return fmt.Errorf("parameter \"%s\" has an invalid default value: value is %q, but only allowed values are %v", p.Name, p.Default, p.EnumValues())
	}
	return nil
}

func (p *Parameter) EnumValues() []interface{} {
	if p.IsEnum() {
		return *p.Enum
	}
	return []interface{}{}
}

type Parameters []Parameter

// Len returns the number of params.
// This is needed to allow sorting of params.
func (p Parameters) Len() int { return len(p) }

// Swap swaps the position of two items in the params slice.
// This is needed to allow sorting of params.
func (p Parameters) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Less returns true if the name of a param a is less than the name of param b.
// This is needed to allow sorting of params.
func (p Parameters) Less(x, y int) bool {
	return p[x].Name < p[y].Name
}

type Groups []Group

type Group struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"prio,omitempty"`
}

// Templates is a map of file names and stringified files in the template folder of an operator
type Templates map[string]string

// Files represents the raw operator package format the way it is found in the tgz packages
type Files struct {
	Templates Templates
	Operator  *OperatorFile
	Params    *ParamsFile
}

// ParamsFile is a representation of the package params.yaml
type ParamsFile struct {
	APIVersion string     `json:"apiVersion,omitempty"`
	Groups     Groups     `json:"groups,omitempty"`
	Parameters Parameters `json:"parameters"`
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
	Maintainers       []*kudoapi.Maintainer   `json:"maintainers,omitempty"`
	URL               string                  `json:"url,omitempty"`
	Tasks             []kudoapi.Task          `json:"tasks"`
	Plans             map[string]kudoapi.Plan `json:"plans"`
	NamespaceManifest string                  `json:"namespaceManifest,omitempty"`
}
