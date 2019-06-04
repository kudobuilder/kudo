package repo

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

// RepositoryConfiguration represents a collection of parameters for framework repository
type RepositoryConfiguration struct {
	LocalPath string `json:"localPath"`
	URL       string `json:"url"`
}

func NewRepositoryConfiguration() *RepositoryConfiguration {
	return &RepositoryConfiguration{
		LocalPath: "$HOME/.kudo/repository", // this won't work on windows
		URL: "https://kudo-test-repo.storage.googleapis.com",
	}
}

// Metadata for a Framework. This models the structure of a bundle.yaml file.
type Metadata struct {
	// The name of the framework
	Name string `json:"name,omitempty"`
	// A SemVer 2 conformant version string of the framework
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`
	// The base version of the application enclosed inside of this framework.
	BaseVersion string `json:"baseVersion,omitempty"`
	// The KUDO API Version of this framework.
	KudoVersion string `json:"kudoVersion,omitempty"`
	// KubernetesVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// A SemVer 2 conformant version string of the framework to upgrade from
	UpgradeVersion string `json:"upgradeVersion,omitempty"`
	// Version dependencies to other frameworks
	Dependencies []v1alpha1.FrameworkDependency `json:"dependencies,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this framework
	Sources []string `json:"sources,omitempty"`
	// A one-sentence description of the framework
	Description string `json:"description,omitempty"`
	// The tag under which the framework can be found, e.g. incubating or stable
	Tag string `json:"tag,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*Maintainer `json:"maintainers,omitempty"`
	// Whether or not this framework is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
}

// Maintainer describes a Framework maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	Name string `json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email,omitempty"`
	// Url is an optional URL to an address for the named maintainer
	URL string `json:"url,omitempty"`
}
