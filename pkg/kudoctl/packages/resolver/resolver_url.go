package resolver

import (
	"bytes"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// URLResolver will resolve a packages from a url
type URLResolver struct {
	client http.Client
}

// Resolve returns a package for the provided url
func (f *URLResolver) Resolve(name string, version string) (*packages.Package, error) {
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

	clog.V(0).Printf("%v is a remote tgz package", name)

	return &packages.Package{
		Resources: resources,
		Files:     files,
	}, nil
}

func (f *URLResolver) getPackageByURL(url string) (*bytes.Buffer, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("resolver: unable to get get reader from url %v", url)
	}

	return resp, nil
}

// NewURL creates an instance of a URLResolver
func NewURL() *URLResolver {
	client := http.NewClient()

	return &URLResolver{
		client: *client,
	}
}
