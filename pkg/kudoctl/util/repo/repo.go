package repo

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

const (
	// Version is the repo / packaging version
	Version         = "v1"
	defaultRepoName = "community"
)

// RepositoryConfiguration represents a collection of parameters for operator repository.
type RepositoryConfiguration struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// Repositories represents the repositories.yaml file usually in the $KUDO_HOME
type Repositories struct {
	RepoVersion  string                     `json:"repoVersion"`
	Context      string                     `json:"context"`
	Repositories []*RepositoryConfiguration `json:"repositories"`
}

// Default initialized repository.
var Default = &RepositoryConfiguration{
	Name: defaultRepoName,
	URL:  "https://kudo-repository.storage.googleapis.com",
}

// NewRepoFile creates a new repo with only defaults populated
func NewRepoFile() *Repositories {
	return &Repositories{
		RepoVersion:  Version,
		Context:      defaultRepoName,
		Repositories: []*RepositoryConfiguration{Default},
	}
}

// GetRepo returns a RepoName Config for a name or nil
func (r Repositories) GetRepo(name string) *RepositoryConfiguration {
	for _, repo := range r.Repositories {
		if repo.Name == name {
			return repo
		}
	}
	return nil
}

// CurrentRepo provides the repo config for the current context
func (r Repositories) CurrentRepo() *RepositoryConfiguration {
	return r.GetRepo(r.Context)
}

// LoadRepositories reads the Repositories file
func LoadRepositories(fs afero.Fs, path string) (*Repositories, error) {
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"could not load repositories file (%s).\n"+
					"You might need to run `kudo init` (or "+
					"`kudo init --client-only` if kudo is "+
					"already installed)", path)
		}
		return nil, err
	}

	r := &Repositories{}
	err = yaml.Unmarshal(b, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// WriteFile writes a repositories file to the given path.
func (r *Repositories) WriteFile(fs afero.Fs, path string, perm os.FileMode) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, path, data, perm)
}

// Metadata for an Operator. This models the structure of an operator.yaml file.
type Metadata struct {
	// Name is the name of the operator.
	Name string `json:"name,omitempty"`

	// Version is a A SemVer 2 conformant version string of the operator.
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`

	// AppVersion is the underlying service version (the format is not in our control)
	AppVersion string `json:"appVersion,omitempty"`

	// The URL to a relevant project page, git repo, or contact person.
	Home string `json:"home,omitempty"`

	// Source is the URL to the source code of this operator.
	Sources []string `json:"sources,omitempty"`

	// Description is a one-sentence description of the operator.
	Description string `json:"description,omitempty"`

	// Maintainers is a list of name and URL/email addresses of the maintainer(s).
	Maintainers []*Maintainer `json:"maintainers,omitempty"`

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
