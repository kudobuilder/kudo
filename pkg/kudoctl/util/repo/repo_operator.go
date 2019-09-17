package repo

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Repository is an abstraction for a service that can retrieve package bundles
type Repository interface {
	GetBundle(name string, version string) (bundle.Bundle, error)
}

// Client represents an operator repository
type Client struct {
	Config *Configuration
	Client http.Client
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
func (r *Client) DownloadIndexFile() (*IndexFile, error) {
	var indexURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(parsedURL.Path, "/"))

	indexURL = parsedURL.String()

	resp, err := r.Client.Get(indexURL)
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

// getPackageReaderByFullPackageName downloads the tgz file from the remote repository and unmarshals it to the package CRDs
// this will cycle through urls for the
func (r *Client) getPackageReaderByFullPackageName(pkg *PackageVersion) (io.Reader, error) {

	pkgURL, err := repositoryBackedPkgURL(r.Config.URL, pkg)
	if err != nil {
		return nil, err
	}
	urls := append(pkg.URLs, pkgURL)

	var pkgErr error
	for _, url := range urls {
		r, err := r.getPackageReaderByURL(url)
		if err == nil {
			return r, nil
		}
		pkgErr = fmt.Errorf("unable to read package %w", err)
		clog.Errorf("failure against url: %v  %v", url, pkgErr)
	}
	clog.Printf("Giving up with err %v", pkgErr)
	return nil, pkgErr
}

func repositoryBackedPkgURL(repoURL string, pkg *PackageVersion) (string, error) {
	packageName := pkg.Name + "-" + pkg.Version
	clog.V(4).Printf("bundle version returned  %v", packageName)

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/%s.tgz", parsedURL.Path, packageName)
	pkgURL := parsedURL.String()
	return pkgURL, nil
}

func (r *Client) getPackageReaderByURL(packageURL string) (io.Reader, error) {
	clog.V(4).Printf("attempt to retrieve package from url: %v", packageURL)
	resp, err := r.Client.Get(packageURL)
	if err != nil {
		return nil, errors.Wrap(err, "getting package url")
	}

	return resp, nil
}

// GetPackageReader provides an io.Reader for a provided package name and optional version
func (r *Client) GetPackageReader(name string, version string) (io.Reader, error) {
	clog.V(4).Printf("getting package reader for %v, %v", name, version)
	clog.V(5).Printf("repository using: %v", r.Config)
	// Construct the package name and download the index file from the remote repo
	indexFile, err := r.DownloadIndexFile()
	if err != nil {
		return nil, errors.WithMessage(err, "could not download repository index file")
	}

	bundleVersion, err := indexFile.GetByNameAndVersion(name, version)
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s in index file", name)
	}

	return r.getPackageReaderByFullPackageName(bundleVersion)
}

// GetBundle provides an Bundle for a provided package name and optional version
func (r *Client) GetBundle(name string, version string) (bundle.Bundle, error) {
	reader, err := r.GetPackageReader(name, version)
	if err != nil {
		return nil, err
	}
	return bundle.NewBundleFromReader(reader), nil
}

// GetOperatorVersionDependencies helper method returns a slice of strings that contains the names of all
// dependency Operators
func GetOperatorVersionDependencies(ov *v1alpha1.OperatorVersion) ([]string, error) {
	var dependencyOperators []string
	if ov.Spec.Dependencies != nil {
		for _, v := range ov.Spec.Dependencies {
			dependencyOperators = append(dependencyOperators, v.Name)
		}
	}
	return dependencyOperators, nil
}
