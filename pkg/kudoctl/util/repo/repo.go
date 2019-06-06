package repo

// RepositoryConfiguration represents a collection of parameters for framework repository
type RepositoryConfiguration struct {
	//LocalPath string `json:"localPath"`
	URL string `json:"url"`
}

// Default returns a given RepositoryConfiguration struct
func Default() *RepositoryConfiguration {
	return &RepositoryConfiguration{
		URL: "https://kudo-repository.storage.googleapis.com",
	}
}

// Metadata for a Framework. This models the structure of a bundle.yaml file.
type Metadata struct {
	// The name of the framework
	Name string `json:"name,omitempty"`
	// A SemVer 2 conformant version string of the framework
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this framework
	Sources []string `json:"sources,omitempty"`
	// A one-sentence description of the framework
	Description string `json:"description,omitempty"`
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
