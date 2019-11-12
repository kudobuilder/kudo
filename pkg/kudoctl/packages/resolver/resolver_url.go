package resolver

import (
	"bytes"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// URLFinder will find an operator packages from a url
type URLFinder struct {
	client http.Client
}

// Resolve provides a package for the url provided
func (f *URLFinder) Resolve(name string, version string) (*packages.Package, error) {
	// check to see if name is url
	if !http.IsValidURL(name) {
		return nil, fmt.Errorf("resolver: url %v invalid", name)
	}
	buf, err := f.getPackageByURL(name)
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

func (f *URLFinder) getPackageByURL(url string) (*bytes.Buffer, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("resolver: unable to get get reader from url %v", url)
	}

	return resp, nil
}

// NewURL creates an instance of a URLFinder
func NewURL() *URLFinder {
	client := http.NewClient()

	return &URLFinder{
		client: *client,
	}
}
