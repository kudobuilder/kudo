package repo

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/Masterminds/semver"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

const defaultURL = "http://localhost/"

// IndexFile represents the index file in an operator repository.
type IndexFile struct {
	APIVersion string                     `json:"apiVersion"`
	Entries    map[string]PackageVersions `json:"entries"`
	Generated  *time.Time                 `json:"generated"`
}

// PackageVersions is a list of versioned package references.
// Implements a sorter on Version.
type PackageVersions []*PackageVersion

// PackageVersion represents an operator entry in the IndexFile.
type PackageVersion struct {
	*Metadata
	URLs    []string `json:"urls"`
	Removed bool     `json:"removed,omitempty"`
	Digest  string   `json:"digest,omitempty"`
}

// Len returns the number of package versions.
// This is needed to allow sorting of packages.
func (b PackageVersions) Len() int { return len(b) }

// Swap swaps the position of two items in the versions slice.
// This is needed to allow sorting of packages.
func (b PackageVersions) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// This is needed to allow sorting of packages.
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

// ParseIndexFile loads an index file and sorts the included packages by version.
// The function will fail if `APIVersion` is not specified.
func ParseIndexFile(data []byte) (*IndexFile, error) {
	i := &IndexFile{}
	if err := yaml.Unmarshal(data, i); err != nil {
		return nil, fmt.Errorf("unmarshalling index file: %w", err)
	}
	if i.APIVersion == "" {
		return nil, errors.New("no API version specified")
	}
	i.sortPackages()
	return i, nil
}

func (i IndexFile) Write(w io.Writer) error {
	b, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if err != nil {
		fmt.Printf("err: %v", err)
	}
	return err
}

// AddPackageVersion adds an entry to the IndexFile (does not allow dups)
func (i *IndexFile) AddPackageVersion(pv *PackageVersion) error {
	name := pv.Name
	version := pv.Version
	if version == "" {
		return fmt.Errorf("operator '%v' is missing version", name)
	}
	if i.Entries == nil {
		i.Entries = make(map[string]PackageVersions)
	}
	vs, ok := i.Entries[name]
	// no entry for operator
	if !ok || len(vs) == 0 {
		pvs := PackageVersions{pv}
		i.Entries[name] = pvs
		return nil
	}

	// loop thru all... don't allow dups
	for _, ver := range vs {
		if ver.Version == version {
			return fmt.Errorf("operator '%v' version: %v already exists", name, version)
		}
	}

	vs = append(vs, pv)
	i.Entries[name] = vs
	return nil
}

// WriteFile is used to write the index file
func (i *IndexFile) WriteFile(fs afero.Fs, file string) (err error) {
	i.sortPackages()
	f, err := fs.Create(file)
	if err != nil {
		return err
	}

	defer func() {
		if ferr := f.Close(); ferr != nil {
			err = ferr
		}
	}()

	return i.Write(f)
}

// Map transforms a slice of packagefiles with file digests into a slice of PackageVersions
func Map(pkgs []*PackageFilesDigest, url string) PackageVersions {
	return mapPackages(pkgs, url, ToPackageVersion)
}

func mapPackages(packages []*PackageFilesDigest, url string, f func(*packages.Files, string, string) *PackageVersion) PackageVersions {
	pvs := make(PackageVersions, len(packages))
	for i, pkg := range packages {
		pvs[i] = f(pkg.PackageFiles, pkg.Digest, url)
	}
	return pvs
}

// ToPackageVersion provided the packageFiles will create a PackageVersion (used for index)
func ToPackageVersion(pf *packages.Files, digest string, url string) *PackageVersion {
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
			AppVersion:  o.AppVersion,
		},
		URLs:   []string{url},
		Digest: digest,
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

// IndexDirectory creates an index file for the operators in the path
func IndexDirectory(fs afero.Fs, path string, url string, now *time.Time) (*IndexFile, error) {
	archives, err := afero.Glob(fs, filepath.Join(path, "*.tgz"))
	if err != nil {
		return nil, err
	}
	if len(archives) == 0 {
		return nil, errors.New("no packages discovered")
	}
	index := newIndexFile(now)
	ops := filesDigest(fs, archives)
	pvs := Map(ops, url)
	for _, pv := range pvs {
		err = index.AddPackageVersion(pv)
		// on error we report and continue
		if err != nil {
			fmt.Print(err.Error())
		}
	}
	index.sortPackages()
	return index, nil
}
