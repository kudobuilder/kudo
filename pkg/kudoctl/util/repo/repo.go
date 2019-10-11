package repo

import (
	"fmt"
	"os"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// A repository is a http backed service which holds operators and an index file for those operators.
// To interact with a repository the client is repo.Client.   To construct the Client
// it is necessary to have a Configuration.   Several Configurations can be stored locally on a
// client in a repositories.yaml file which represented by the Repositories struct.

const (
	// Version is the repo / packaging version
	Version         = "v1"
	defaultRepoName = "community"
)

// Configuration represents a collection of parameters for operator repository.
type Configuration struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// Configurations is a collection of Configuration for Stringer
type Configurations []*Configuration

// Repositories represents the repositories.yaml file usually in the $KUDO_HOME
type Repositories struct {
	RepoVersion  string         `json:"repoVersion"`
	Context      string         `json:"context"`
	Repositories Configurations `json:"repositories"`
}

// String is a stringer function for Configuration
func (c *Configuration) String() string {
	return fmt.Sprintf("{ name:%v, url:%v }", c.Name, c.URL)
}

// String is a stringer function for Configurations
func (c Configurations) String() string {
	s := "repo configs: "
	for _, config := range c {
		s = fmt.Sprintf("%v %v,", s, config)
	}

	return s
}

// Default initialized repository.
var Default = &Configuration{
	Name: defaultRepoName,
	URL:  "https://kudo-repository.storage.googleapis.com",
}

// NewRepositories creates a new repo with only defaults populated
func NewRepositories() *Repositories {
	return &Repositories{
		RepoVersion:  Version,
		Context:      defaultRepoName,
		Repositories: []*Configuration{Default},
	}
}

// GetConfiguration returns a RepoName Config for a name or nil
func (r *Repositories) GetConfiguration(name string) *Configuration {
	for _, repo := range r.Repositories {
		if repo.Name == name {
			return repo
		}
	}
	return nil
}

// CurrentConfiguration provides the repo config for the current context
func (r *Repositories) CurrentConfiguration() *Configuration {
	return r.GetConfiguration(r.Context)
}

// ConfigurationFromSettings gets the repo configuration from settings
func ConfigurationFromSettings(fs afero.Fs, home kudohome.Home, repoName string) (*Configuration, error) {
	r, err := LoadRepositories(fs, home.RepositoryFile())
	if err != nil {
		// this allows for no client init... perhaps we should return the error requesting kudo init
		r = NewRepositories()
	}
	var config *Configuration
	if repoName == "" {
		config = r.CurrentConfiguration()
	} else {
		config = r.GetConfiguration(repoName)
	}
	if config == nil {
		return nil, fmt.Errorf("unable to find respository for %s", repoName)
	}
	return config, nil
}

// LoadRepositories reads the Repositories file
func LoadRepositories(fs afero.Fs, path string) (*Repositories, error) {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf(
			"could not load repositories file (%s).\n"+
				"You might need to run `kudo init` (or "+
				"`kudo init --client-only` if kudo is "+
				"already installed)", path)
	}

	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	r := &Repositories{}
	err = yaml.Unmarshal(b, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Add appends a slice of repo configs to repositories file
func (r *Repositories) Add(repo ...*Configuration) {
	r.Repositories = append(r.Repositories, repo...)
}

// Remove removes the repo config with the provided name
func (r *Repositories) Remove(name string) bool {
	repos := []*Configuration{}
	found := false
	for _, repo := range r.Repositories {
		if repo.Name == name {
			found = true
			continue
		}
		repos = append(repos, repo)
	}
	r.Repositories = repos
	return found
}

// SetContext switches the context to another repo config in the repositories file.  errors if no repo found.
func (r *Repositories) SetContext(context string) error {
	config := r.GetConfiguration(context)
	if config == nil {
		return fmt.Errorf("no config found with name: %s", context)
	}
	r.Context = context
	return nil
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

	// Description is a one-sentence description of the operator.
	Description string `json:"description,omitempty"`

	// Maintainers is a list of name and URL/email addresses of the maintainer(s).
	Maintainers []*v1alpha1.Maintainer `json:"maintainers,omitempty"`
}
