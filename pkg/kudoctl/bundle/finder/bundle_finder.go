package finder

import (
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/spf13/afero"
)

// Finder is a bundle finder and is any implementation which can find/discover a bundle.
// Even Repos are finders.  Local and URL Finders current do nothing with the version information.
type Finder interface {
	GetBundle(name string, version string) (bundle.Bundle, error)
}

// LocalFinder will find local operator bundle: folders or tgz
type LocalFinder struct {
	fs afero.Fs
}

// URLFinder will find an operator bundle from a url
type URLFinder struct {
	client http.Client
}

// Manager is the source of finder of operator bundles.
type Manager struct {
	local *LocalFinder
	uri   *URLFinder
}

// New creates an operator bundle finder for non-repository bundles
func New() *Manager {
	lf := NewLocal()
	uf := NewURL()
	return &Manager{
		local: lf,
		uri:   uf,
	}
}

// GetBundle provides a one stop to acquire any non-repo bundle.  We should refactor repo to be in the finder package and have manager manage it.
func (f *Manager) GetBundle(name string, version string) (bundle.Bundle, error) {

	// if local folder return the bundle
	if _, err := f.local.fs.Stat(name); err == nil {
		b, err := f.local.GetBundle(name, version)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// if url return that bundle
	if http.IsValidURL(name) {
		b, err := f.uri.GetBundle(name, version)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, fmt.Errorf("finder: unable to find bundle for %v", name)
}

// GetBundle provides a bundle for the url provided
func (f *URLFinder) GetBundle(name string, version string) (bundle.Bundle, error) {
	// check to see if name is url
	if !http.IsValidURL(name) {
		return nil, fmt.Errorf("finder: url %v invalid", name)
	}
	reader, err := f.getBundleByURL(name)
	if err != nil {
		return nil, err
	}
	return bundle.NewBundleFromReader(reader), nil
}

func (f *URLFinder) getBundleByURL(url string) (io.Reader, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("finder: unable to get get reader from url %v", url)
	}

	return resp, nil
}

// GetBundle provides a bundle for the local folder or tarball provided
func (f *LocalFinder) GetBundle(name string, version string) (bundle.Bundle, error) {
	//	make sure file exists
	_, err := f.fs.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", name)
	}
	// order of discovery
	// 1. tarball
	// 2. file based
	return bundle.NewBundle(f.fs, name)
}

// NewLocal creates a finder for local operator bundles
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
