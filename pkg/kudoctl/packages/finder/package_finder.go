package finder

import (
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/spf13/afero"
)

// Finder is a packages finder and is any implementation which can find/discover a packages.
// Even Repos are finders.  Local and URL Finders current do nothing with the version information.
type Finder interface {
	GetPackage(name string, version string) (packages.Package, error)
}

// LocalFinder will find local operator packages: folders or tgz
type LocalFinder struct {
	fs afero.Fs
}

// URLFinder will find an operator packages from a url
type URLFinder struct {
	client http.Client
}

// Manager is the source of finder of operator packages.
type Manager struct {
	local *LocalFinder
	uri   *URLFinder
}

// New creates an operator package finder for non-repository packages
func New() *Manager {
	lf := NewLocal()
	uf := NewURL()
	return &Manager{
		local: lf,
		uri:   uf,
	}
}

// GetPackage provides a one stop to acquire any non-repo packages.  We should refactor repo to be in the finder package and have manager manage it.
func (f *Manager) GetPackage(name string, version string) (packages.Package, error) {

	// if local folder return the package
	if _, err := f.local.fs.Stat(name); err == nil {
		b, err := f.local.GetPackage(name, version)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// if url return that package
	if http.IsValidURL(name) {
		b, err := f.uri.GetPackage(name, version)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, fmt.Errorf("finder: unable to find packages for %v", name)
}

// GetPackage provides a package for the url provided
func (f *URLFinder) GetPackage(name string, version string) (packages.Package, error) {
	// check to see if name is url
	if !http.IsValidURL(name) {
		return nil, fmt.Errorf("finder: url %v invalid", name)
	}
	reader, err := f.getPackageByURL(name)
	if err != nil {
		return nil, err
	}
	return packages.NewPackageFromReader(reader), nil
}

func (f *URLFinder) getPackageByURL(url string) (io.Reader, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("finder: unable to get get reader from url %v", url)
	}

	return resp, nil
}

// GetPackage provides a package for the local folder or tarball provided
func (f *LocalFinder) GetPackage(name string, version string) (packages.Package, error) {
	//	make sure file exists
	_, err := f.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	// order of discovery
	// 1. tarball
	// 2. file based
	return packages.ReadPackage(f.fs, name)
}

// NewLocal creates a finder for local operator package
func NewLocal() *LocalFinder {
	return &LocalFinder{fs: afero.NewOsFs()}
}

// NewURL creates an instance of a URLFinder
func NewURL() *URLFinder {
	client := http.NewClient()

	return &URLFinder{
		client: *client,
	}
}
