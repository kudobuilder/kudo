package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Repository is an abstraction for a service that can retrieve packages
type Repository interface {
	GetPackage(name string, version string) (packages.Package, error)
}

// Client represents an operator repository
type Client struct {
	Config *Configuration
	Client http.Client
}

func (c *Client) String() string {
	return c.Config.String()
}

// ClientFromSettings retrieves the operator repo for the configured repo in settings
func ClientFromSettings(fs afero.Fs, home kudohome.Home, repoName string) (*Client, error) {
	rc, err := ConfigurationFromSettings(fs, home, repoName)
	if err != nil {
		return nil, err
	}

	return NewClient(rc)
}

// NewClient constructs repository client
func NewClient(conf *Configuration) (*Client, error) {
	_, err := url.Parse(conf.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %s", conf.URL)
	}

	client := http.NewClient()

	return &Client{
		Config: conf,
		Client: *client,
	}, nil
}

// DownloadIndexFile fetches the index file from a repository.
func (c *Client) DownloadIndexFile() (*IndexFile, error) {
	var indexURL string
	parsedURL, err := url.Parse(c.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(parsedURL.Path, "/"))

	indexURL = parsedURL.String()

	resp, err := c.Client.Get(indexURL)
	if err != nil {
		return nil, errors.Wrap(err, "getting index url")
	}

	indexBytes, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, errors.Wrap(err, "reading index response")
	}

	indexFile, err := ParseIndexFile(indexBytes)
	return indexFile, err
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

// GetPackage provides an Package for a provided package name and optional version
func (c *Client) GetPackage(name string, version string) (packages.Package, error) {
	reader, err := c.GetPackageBytes(name, version)
	if err != nil {
		return nil, err
	}
	return packages.NewFromBytes(reader), nil
}
