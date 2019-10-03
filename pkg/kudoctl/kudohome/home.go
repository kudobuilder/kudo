package kudohome

import (
	"os"
	"path/filepath"
)

// Home describes the location of a CLI configuration.
type Home string

// String returns Home as a string with expansions $HOME == home
func (h Home) String() string {
	return os.ExpandEnv(string(h))
}

// Path returns Home with elements appended.
func (h Home) path(elem ...string) string {
	p := []string{h.String()}
	p = append(p, elem...)
	return filepath.Join(p...)
}

// Repository returns the path to the local repository.
func (h Home) Repository() string {
	return h.path("repository")
}

// RepositoryFile returns the path to the repositories.yaml file.
func (h Home) RepositoryFile() string {
	return h.path("repository", "repositories.yaml")
}
