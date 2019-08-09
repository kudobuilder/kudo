package repo

import (
	"fmt"
	"io"
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
	Generated  *time.Time                `json:"generated"`
}

// BundleVersions is a list of versioned bundle references.
// Implements a sorter on Version.
type BundleVersions []*BundleVersion

// BundleVersion represents a operator entry in the IndexFile
type BundleVersion struct {
	*Metadata
	URLs       []string   `json:"urls"`
	APIVersion string     `json:"apiVersion"`
	AppVersion string     `json:"appVersion"`
	Created    *time.Time `json:"created,omitempty"`
	Removed    bool       `json:"removed,omitempty"`
	Digest     string     `json:"digest,omitempty"`
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
		return nil, errors.Wrap(err, "unmarshalling index file")
	}
	if i.APIVersion == "" {
		return nil, errors.New("no API version specified")
	}
	i.sortPackages()
	return i, nil
}

func writeIndexFile(i *IndexFile, w io.Writer) error {
	b, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// GetByNameAndVersion returns the operator of given name and version.
// If no specific version is required, pass an empty string as version and the
// the latest version will be returned.
func (i IndexFile) GetByNameAndVersion(name, version string) (*BundleVersion, error) {
	vs, ok := i.Entries[name]
	if !ok || len(vs) == 0 {
		return nil, fmt.Errorf("no operator found for: %s", name)
	}

	for _, ver := range vs {
		if ver.Version == version || version == "" {
			return ver, nil
		}
	}

	if version == "" {
		return nil, fmt.Errorf("no operator version found for %s", name)
	}

	return nil, fmt.Errorf("no operator version found for %s-%v", name, version)
}

func (i IndexFile) addBundleVersion(b *BundleVersion) error {
	name := b.Name
	version := b.Version
	if version == "" {
		return errors.Errorf("operator '%v' is missing version", name)
	}
	if i.Entries == nil {
		i.Entries = make(map[string]BundleVersions)
	}
	vs, ok := i.Entries[name]
	// no entry for operator
	if !ok || len(vs) == 0 {
		i.Entries[name] = BundleVersions{b}
		return nil
	}

	// loop thru all... don't allow dups
	for _, ver := range vs {
		if ver.Version == version {
			return errors.Errorf("operator '%v' version: %v already exists", name, version)
		}
	}

	vs = append(vs, b)
	i.Entries[name] = vs
	return nil
}

// MapPackageToBundleVersion
//func MapPackageToBundleVersion(fs afero.Fs, name string) (*BundleVersion, error) {
//	f, err := fs.Open(name)
//	if err != nil {
//		return nil, err
//	}
//	defer f.Close()
//
//	return mapPackageToBundleVersion(fs, f)
//}
//
//func mapPackageToBundleVersion(fs afero.Fs, r io.Reader) (*BundleVersion, error) {
//	//	todo: get operator.yaml from
//	return nil, nil
//}
