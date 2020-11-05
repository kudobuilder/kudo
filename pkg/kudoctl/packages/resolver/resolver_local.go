package resolver

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// LocalResolver will find local operator packages: folders or tgz
type LocalResolver struct {
	fs afero.Fs
}

// Resolve a local package. The path can be relative or absolute location of the packages.
// Order of the discovery is:
// 1. tarball
// 2. dir based
func (f *LocalResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Resources, error) {
	//	make sure file exists
	_, err := f.fs.Stat(name)
	if err != nil {
		return nil, err
	}

	//	make sure file exists
	fi, err := f.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	clog.V(1).Printf("determining package type of %v", name)

	if fi.Mode().IsRegular() && strings.HasSuffix(name, ".tgz") {
		clog.V(1).Printf("%v is a local tgz package", name)
		return reader.ReadTar(f.fs, name)
	}
	if fi.IsDir() {
		clog.V(1).Printf("%v is a local file package", name)
		return reader.ResourcesFromDir(f.fs, name)
	}
	return nil, fmt.Errorf("unsupported file system format %v. Expect either a *.tgz file or a folder", name)
}

// NewLocal creates a resolver for local operator package
func NewLocal() *LocalResolver {
	return &LocalResolver{fs: afero.NewOsFs()}
}
