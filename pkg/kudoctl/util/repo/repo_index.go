package repo

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/kudobuilder/kudo/pkg/version"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"sigs.k8s.io/yaml"
	"sort"
	"time"
)

// IndexFile represents the index file in a framework repository
type IndexFile struct {
	APIVersion string                    `json:"apiVersion"`
	Generated  time.Time                 `json:"generated"`
	Entries    map[string]BundleVersions `json:"entries"`
}

// BundleVersions is a list of versioned bundle references.
// Implements a sorter on Version.
type BundleVersions []*BundleVersion

// BundleVersion represents a framework entry in the IndexFile
type BundleVersion struct {
	*Metadata
	URLs    []string  `json:"urls"`
	Created time.Time `json:"created,omitempty"`
	Removed bool      `json:"removed,omitempty"`
	Digest  string    `json:"digest,omitempty"`
}

// Len returns the length.
func (b BundleVersions) Len() int { return len(b) }

// Swap swaps the position of two items in the versions slice.
func (b BundleVersions) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
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

// NewIndexFile initializes an index.
func NewIndexFile() *IndexFile {
	return &IndexFile{
		APIVersion: version.Get().GitVersion,
		Generated:  time.Now(),
		Entries:    map[string]BundleVersions{},
	}
}

// SortEntries sorts the entries by version in descending order.
//
// In canonical form, the individual version records should be sorted so that
// the most recent release for every version is in the 0th slot in the
// Entries.BundleVersions array. That way, tooling can predict the newest
// version without needing to parse SemVers.
func (i IndexFile) SortEntries() {
	for _, versions := range i.Entries {
		sort.Sort(sort.Reverse(versions))
	}
}

// LoadIndexFile takes a file at the given path and returns an IndexFile object
func (i IndexFile) LoadIndexFile(path string) (*IndexFile, error) {
	b, err := ioutil.ReadFile(path + "/index.yaml")
	if err != nil {
		return nil, err
	}

	return loadIndex(b)
}

// loadIndex loads an index file and does minimal validity checking.
//
// This will fail if API Version is not set (ErrNoAPIVersion) or if the unmarshal fails.
func loadIndex(data []byte) (*IndexFile, error) {
	i := &IndexFile{}
	if err := yaml.Unmarshal(data, i); err != nil {
		return i, errors.Wrap(err, "unmarshalling index file")
	}
	i.SortEntries()
	if i.APIVersion == "" {
		return i, errors.New("no API version specified")
	}
	return i, nil
}

// Get returns the ChartVersion for the given name.
//
// If version is empty, this will return the chart with the highest version.
func (i IndexFile) Get(name, version string) (*BundleVersion, error) {
	vs, ok := i.Entries[name]
	if !ok {
		return nil, errors.New("no chart name found")
	}
	if len(vs) == 0 {
		return nil, errors.New("no chart version found")
	}

	var constraint *semver.Constraints
	if len(version) == 0 {
		constraint, _ = semver.NewConstraint("*")
	} else {
		var err error
		constraint, err = semver.NewConstraint(version)
		if err != nil {
			return nil, err
		}
	}

	// when customer input exact version, check whether have exact match one first
	if len(version) != 0 {
		for _, ver := range vs {
			if version == ver.Version {
				return ver, nil
			}
		}
	}

	for _, ver := range vs {
		test, err := semver.NewVersion(ver.Version)
		if err != nil {
			continue
		}

		if constraint.Check(test) {
			return ver, nil
		}
	}
	return nil, fmt.Errorf("No chart version found for %s-%s", name, version)
}

// Exists returns true if the index.yaml file exists on the local file system
func (i IndexFile) Exists() bool {
	_, err := os.Stat(vars.RepoPath + "/index.yaml")
	if err != nil {
		return false
	}
	return true
}
