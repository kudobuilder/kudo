package repo

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const defaultURL = "http://localhost/"

// IndexFile represents the index file in an operator repository
type IndexFile struct {
	APIVersion string                     `json:"apiVersion"`
	Entries    map[string]PackageVersions `json:"entries"`
	Generated  *time.Time                 `json:"generated"`
}

// PackageVersions is a list of versioned package references.
// Implements a sorter on Version.
type PackageVersions []*PackageVersion

// PackageVersion represents an operator entry in the IndexFile
type PackageVersion struct {
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
func (b PackageVersions) Len() int { return len(b) }

// Swap swaps the position of two items in the versions slice.
// this is needed to allow sorting of packages
func (b PackageVersions) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// this is needed to allow sorting of packages
func (b PackageVersions) Less(x, y int) bool {
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
// Entries.PackageVersions array. That way, tooling can predict the newest
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
	fmt.Printf("index: %v", string(b))
	_, err = w.Write(b)
	if err != nil {
		fmt.Printf("err: %v", err)
	}
	return err
}

// GetByNameAndVersion returns the operator of given name and version.
// If no specific version is required, pass an empty string as version and the
// the latest version will be returned.
func (i IndexFile) GetByNameAndVersion(name, version string) (*PackageVersion, error) {
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

func (i *IndexFile) addBundleVersion(b *PackageVersion) error {
	name := b.Name
	version := b.Version
	if version == "" {
		return errors.Errorf("operator '%v' is missing version", name)
	}
	if i.Entries == nil {
		i.Entries = make(map[string]PackageVersions)
	}
	vs, ok := i.Entries[name]
	// no entry for operator
	if !ok || len(vs) == 0 {
		pvs := PackageVersions{b}
		i.Entries[name] = pvs
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

// Map transforms a slice of paths into a slice of PackageVersion
//func MapPaths(paths []string, f func(string) PackageVersion) PackageVersions {
//	pvs := make(PackageVersions, len(paths))
//
//}
func Map(pkgs []*bundle.PackageFiles, url string, creation *time.Time) PackageVersions {
	return mapPackages(pkgs, url, creation, ToPackageVersion)
}

func mapPackages(packages []*bundle.PackageFiles, url string, creation *time.Time, f func(*bundle.PackageFiles, string, *time.Time) *PackageVersion) PackageVersions {
	pvs := make(PackageVersions, len(packages))
	for i, pkg := range packages {
		pvs[i] = f(pkg, url, creation)
	}
	return pvs
}

// ToPackageVersion provided the packageFiles will create a PackageVersion (used for index)
func ToPackageVersion(pf *bundle.PackageFiles, url string, creation *time.Time) *PackageVersion {
	o := pf.Operator
	if url == "" {
		url = defaultURL
	}
	if url[len(url)-1:] != "/" {
		url = url + "/"
	}
	url = fmt.Sprintf("%s%s-%v.tgz", url, o.Name, o.Version)
	pv := PackageVersion{
		Metadata: &Metadata{
			Name:        o.Name,
			Version:     o.Version,
			Description: o.Description,
			Maintainers: o.Maintainers,
		},
		URLs:       []string{url},
		APIVersion: "",
		AppVersion: o.AppVersion,
		Created:    creation,
		//Digest:     "",   // todo: add digest
	}
	return &pv
}

func newIndexFile(t *time.Time) *IndexFile {
	i := IndexFile{
		APIVersion: "v1",
		Generated:  t,
	}
	return &i
}

//func MapPackageToBundleVersion(fs afero.Fs, name string) (*PackageVersion, error) {
//	f, err := fs.Open(name)
//	if err != nil {
//		return nil, err
//	}
//	defer f.Close()
//
//	return mapPackageToBundleVersion(fs, f)
//}
//
//func mapPackageToBundleVersion(fs afero.Fs, r io.Reader) (*PackageVersion, error) {
//	//	todo: get operator.yaml from
//	return nil, nil
//}
