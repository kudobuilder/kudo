package repo

import (
	"fmt"
	"sort"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// IndexFile represents the index file in a operator repository
type IndexFile struct {
	APIVersion string                    `json:"apiVersion"`
	Entries    map[string]BundleVersions `json:"entries"`
}

// BundleVersions is a list of versioned bundle references.
// Implements a sorter on Version.
type BundleVersions []*BundleVersion

// BundleVersion represents a operator entry in the IndexFile
type BundleVersion struct {
	*Metadata
	URLs    []string  `json:"urls"`
	Created time.Time `json:"created,omitempty"`
	Removed bool      `json:"removed,omitempty"`
	Digest  string    `json:"digest,omitempty"`
}

// Len returns the length.
// this is needed to allow sorting of packages
func (b BundleVersions) Len() int { return len(b) }

// Swap swaps the position of two items in the versions slice.
// this is needed to allow sorting of packages
func (b BundleVersions) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// this is needed to allow sorting of packages
func (b BundleVersions) Less(x, y int) bool {
	// Failed parse pushes to the back.
	i, err := semver.NewVersion(b[x].Version)
	if err != nil {
		return true
	}
	j, err := semver.NewVersion(b[y].Version)
	if err != nil {
		return false
	}
	return i.LessThan(j)
}

// sortPackages sorts the entries by version in descending order.
//
// In canonical form, the individual version records should be sorted so that
// the most recent release for every version is in the 0th slot in the
// Entries.BundleVersions array. That way, tooling can predict the newest
// version without needing to parse SemVers.
func (i IndexFile) sortPackages() {
	for _, versions := range i.Entries {
		sort.Sort(sort.Reverse(versions))
	}
}

// parseIndexFile loads an index file and sorts packages inside
// this will fail if APIVersion is not specified
func parseIndexFile(data []byte) (*IndexFile, error) {
	i := &IndexFile{}
	if err := yaml.Unmarshal(data, i); err != nil {
		return i, errors.Wrap(err, "unmarshalling index file")
	}
	i.sortPackages()
	if i.APIVersion == "" {
		return i, errors.New("no API version specified")
	}
	return i, nil
}

// GetByNameAndVersion returns the operator of given name and version.
// When no specific version required, submit version as empty string - ""
func (i IndexFile) GetByNameAndVersion(name, version string) (*BundleVersion, error) {
	vs, ok := i.Entries[name]
	if !ok || len(vs) == 0 {
		return nil, fmt.Errorf("no operator of given name %s", name)
	}

	for _, ver := range vs {
		if ver.Version == version || version == "" {
			return ver, nil
		}
	}
	return nil, fmt.Errorf("no operator version found for %s-%v", name, version)
}
