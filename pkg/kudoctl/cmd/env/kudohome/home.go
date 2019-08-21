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
func (h Home) Path(elem ...string) string {
	p := []string{h.String()}
	p = append(p, elem...)
	return filepath.Join(p...)
}

// RepositoryFile returns the path to the repositories.yaml file.
func (h Home) RepositoryFile() string {
	return h.Path("repository", "repositories.yaml")
}
