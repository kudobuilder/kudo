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

// Metadata for an Operator. This models the structure of an operator.yaml file.
type Metadata struct {
	// Name is the name of the operator.
	Name string `json:"name,omitempty"`

	// Version is a A SemVer 2 conformant version string of the operator.
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`

	// AppVersion is the underlying service version (the format is not in our control)
	AppVersion string     `json:"appVersion"`

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

// Maintainer describes an Operator maintainer.
type Maintainer struct {
	// Name is a user name or organization name.
	Name string `json:"name,omitempty"`

	// Email is an optional email address to contact the named maintainer.
	Email string `json:"email,omitempty"`

	// URL is an optional URL to an address for the named maintainer.
	URL string `json:"url,omitempty"`
}
