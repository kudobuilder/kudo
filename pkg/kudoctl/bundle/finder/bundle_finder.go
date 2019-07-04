package finder

import (
	"fmt"
	"io"
	"os"

	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
)

// Finder is a bundle finder and is any implementation which can find/discover a bundle
type Finder interface {
	GetBundle(name string) (bundle.Bundle, error)
}

// LocalFinder will find local operator bundle: folders or tgz
type LocalFinder struct {
}

// URLFinder will find an operator bundle from a url
type URLFinder struct {
	client http.Client
}

// GetBundle provides a bundle for the url provided
func (f *URLFinder) GetBundle(name string) (bundle.Bundle, error) {
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
func (f *LocalFinder) GetBundle(name string) (bundle.Bundle, error) {
	//	make sure file exists
	_, err := os.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", name)
	}
	// order of discovery
	// 1. tarball
	// 2. file based
	return bundle.NewBundle(name)
}

// NewLocal creates a finder for local operator bundles
func NewLocal() (finder *LocalFinder, err error) {
	return &LocalFinder{}, nil
}

// NewURL creates an instance of a URLFinder
func NewURL() (finder *URLFinder, err error) {
	client, err := http.NewClient()
	if err != nil {
		return nil, fmt.Errorf("could not construct http client: %v", err)
	}

	return &URLFinder{
		client: *client,
	}, nil
}
