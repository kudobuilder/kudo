package repo

import "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

// RepositoryConfiguration represents a collection of parameters for operator repository.
type RepositoryConfiguration struct {
	URL string `json:"url"`
}

// Default initialized repository.
var Default = &RepositoryConfiguration{
	URL: "https://kudo-repository.storage.googleapis.com",
}

// Metadata for a Operator. This models the structure of a operator.yaml file.
type Metadata struct {
	// Name is the name of the operator.
	Name string `json:"name,omitempty"`

	// Version is a A SemVer 2 conformant version string of the operator.
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`

	// The URL to a relevant project page, git repo, or contact person.
	Home string `json:"home,omitempty"`

	// Source is the URL to the source code of this operator.
	Sources []string `json:"sources,omitempty"`

	// Description is a one-sentence description of the operator.
	Description string `json:"description,omitempty"`

	// Maintainers is a list of name and URL/email addresses of the maintainer(s).
	Maintainers []v1alpha1.Maintainer `json:"maintainers,omitempty"`

	// Deprecated reflects whether this operator is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`
}
