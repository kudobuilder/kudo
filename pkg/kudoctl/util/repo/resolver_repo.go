package repo

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

type EntrySummaries []EntrySummary

type EntrySummary struct {
	Name            string
	OperatorVersion string
	AppVersion      string
}

// Len returns the number of entry summaries.
// This is needed to allow sorting of entries.
func (b EntrySummaries) Len() int { return len(b) }

// Swap swaps the position of two items in the entry summaries slice.
// This is needed to allow sorting of entry summaries.
func (b EntrySummaries) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// This is needed to allow sorting of entry summaries.
func (b EntrySummaries) Less(x, y int) bool {
	return b[x].Name < b[y].Name
}

// Resolve returns a Package for a passed package name and optional version. This is an implementation
// of the DependenciesResolver interface located in packages/resolver/resolver.go
func (c *Client) Resolve(out afero.Fs, name string, appVersion string, operatorVersion string) (*packages.Resources, error) {
	buf, err := c.GetPackageBytes(name, appVersion, operatorVersion)
	if err != nil {
		return nil, err
	}
	clog.V(2).Printf("%v is a repository package from %v", name, c.Config)

	files, err := reader.PackageFilesFromTar(out, buf)
	if err != nil {
		return nil, err
	}

	resources, err := convert.FilesToResources(files)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (c *Client) Find(search string, allVersions bool) (EntrySummaries, error) {
	indexFile, err := c.DownloadIndexFile()
	if err != nil {
		return nil, fmt.Errorf("could not download repository index file: %w", err)
	}
	return indexFile.Find(search, allVersions)
}

// GetPackageBytes provides an io.Reader for a provided package name and optional version
func (c *Client) GetPackageBytes(name string, appVersion string, operatorVersion string) (*bytes.Buffer, error) {
	clog.V(4).Printf("getting package reader for %v, %v_%v", name, appVersion, operatorVersion)
	clog.V(5).Printf("repository using: %v", c.Config)
	// Construct the package name and download the index file from the remote repo
	indexFile, err := c.DownloadIndexFile()
	if err != nil {
		return nil, fmt.Errorf("could not download repository index file: %w", err)
	}

	pkgVersion, err := indexFile.FindFirstMatch(name, appVersion, operatorVersion)
	if err != nil {
		return nil, fmt.Errorf("getting %s in index file: %w", name, err)
	}

	return c.getPackageReaderByAPackageURL(pkgVersion)
}

// getPackageReaderByAPackageURL downloads the tgz file from the remote repository and returns a reader
// The PackageVersion is a package configuration from the index file which has a list of urls where
// the package can be pulled from.  This will cycle through the list of urls and will return the reader
// from the first successful url.  If all urls fail, the last error will be returned.
func (c *Client) getPackageReaderByAPackageURL(pkg *PackageVersion) (*bytes.Buffer, error) {
	var pkgErr error
	for _, u := range pkg.URLs {
		r, err := c.getPackageBytesByURL(u)
		if err == nil {
			return r, nil
		}
		pkgErr = fmt.Errorf("unable to read package %w", err)
		clog.V(2).Printf("failure against url: %v  %v", u, pkgErr)
	}
	clog.Printf("Giving up with err %v", pkgErr)
	return nil, pkgErr
}

func (c *Client) getPackageBytesByURL(packageURL string) (*bytes.Buffer, error) {
	clog.V(4).Printf("attempt to retrieve package from url: %v", packageURL)
	resp, err := c.Client.Get(packageURL)
	if err != nil {
		return nil, fmt.Errorf("getting package url: %w", err)
	}

	return resp, nil
}

// FindFirstMatch returns the operator of given name and version.
// If no specific version is required, pass an empty string as version and the
// the latest version will be returned.
// Possible package options include: foo-1.0.0.tgz, foo-2.0.0_1.0.1.tgz and foo-3.0.0_1.0.1.tgz
// The Entries are sorted by AppVersion first, then OpVersion.  Entries with no appVersion are later in the sort order than
// entries with appVersion.  Given a search for an opVersion = 1.0.1 (without appVersion) given the above foo options,
// foo-3.0.0-1.0.1 (the latest app version for this opVersion)
// appVersion could be arbitrary.  if appVersion is "bar" than foo-var_1.0.1.tgz
func (i IndexFile) FindFirstMatch(name string, appVersion string, operatorVersion string) (*PackageVersion, error) {
	vs, ok := i.Entries[name]
	if !ok || len(vs) == 0 {
		return nil, fmt.Errorf("no operator found for: %s", name)
	}
	return findFirstMatchForEntries(vs, name, appVersion, operatorVersion)
}

func findFirstMatchForEntries(versions PackageVersions, name, appVersion, operatorVersion string) (*PackageVersion, error) {
	for _, ver := range versions {
		if (ver.AppVersion == appVersion || appVersion == "") &&
			(ver.OperatorVersion == operatorVersion || operatorVersion == "") {
			return ver, nil
		}
	}

	if operatorVersion == "" {
		return nil, fmt.Errorf("no operator version found for %s", name)
	}

	return nil, fmt.Errorf("no operator version found for %s-%v", name, operatorVersion)
}

func (i IndexFile) Find(search string, allVersions bool) (EntrySummaries, error) {

	summaries := make(EntrySummaries, 0)
	for name, versions := range i.Entries {
		if strings.Contains(name, search) {
			if allVersions {
				for _, pv := range versions {
					summary := EntrySummary{
						Name:            name,
						OperatorVersion: pv.OperatorVersion,
						AppVersion:      pv.AppVersion,
					}
					summaries = append(summaries, summary)
				}
				//	 only the current version
			} else {
				pv, err := i.FindFirstMatch(name, "", "")
				if err != nil {
					return nil, err
				}
				summary := EntrySummary{
					Name:            name,
					OperatorVersion: pv.OperatorVersion,
					AppVersion:      pv.AppVersion,
				}
				summaries = append(summaries, summary)
			}
		}
	}
	sort.Sort(summaries)

	return summaries, nil
}
