package repo

import (
	"bytes"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
	"github.com/pkg/errors"
)

// Resolve returns a Package for a passed package name and optional version. This is an implementation
// of the Resolver interface located in packages/resolver/resolver.go
func (c *Client) Resolve(name string, version string) (*packages.Package, error) {
	buf, err := c.GetPackageBytes(name, version)
	if err != nil {
		return nil, err
	}
	files, err := reader.ParseTgz(buf)
	if err != nil {
		return nil, err
	}

	resources, err := files.Resources()
	if err != nil {
		return nil, err
	}

	return &packages.Package{
		Resources: resources,
		Files:     files,
	}, nil
}

// GetPackageBytes provides an io.Reader for a provided package name and optional version
func (c *Client) GetPackageBytes(name string, version string) (*bytes.Buffer, error) {
	clog.V(4).Printf("getting package reader for %v, %v", name, version)
	clog.V(5).Printf("repository using: %v", c.Config)
	// Construct the package name and download the index file from the remote repo
	indexFile, err := c.DownloadIndexFile()
	if err != nil {
		return nil, errors.WithMessage(err, "could not download repository index file")
	}

	pkgVersion, err := indexFile.GetByNameAndVersion(name, version)
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s in index file", name)
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
		return nil, errors.Wrap(err, "getting package url")
	}

	return resp, nil
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
