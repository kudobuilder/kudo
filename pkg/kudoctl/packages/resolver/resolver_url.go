package resolver

import (
	"bytes"
	"fmt"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// URLHelper will resolve a packages from a url
type URLHelper struct {
	client http.Client
}

// ResolveURL returns a package for the provided url
func (f *URLHelper) ResolveURL(out afero.Fs, url string) (*packages.Resources, error) {
	// check to see if url is url
	if !http.IsValidURL(url) {
		return nil, fmt.Errorf("resolver: url %v invalid", url)
	}
	buf, err := f.getPackageByURL(url)
	if err != nil {
		return nil, err
	}
	files, err := reader.PackageFilesFromTar(out, buf)
	if err != nil {
		return nil, err
	}

	resources, err := convert.FilesToResources(files)
	if err != nil {
		return nil, err
	}

	clog.V(0).Printf("%v is a remote .tgz package", url)

	return resources, nil
}

func (f *URLHelper) getPackageByURL(url string) (*bytes.Buffer, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("resolver: unable to get get reader from url %v", url)
	}

	return resp, nil
}

// NewURLHelper creates an instance of a URLHelper
func NewURLHelper() *URLHelper {
	client := http.NewClient()

	return &URLHelper{
		client: *client,
	}
}
