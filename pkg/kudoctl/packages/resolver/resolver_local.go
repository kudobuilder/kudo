package resolver

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
	"github.com/spf13/afero"
)

// LocalFinder will find local operator packages: folders or tgz
type LocalFinder struct {
	fs afero.Fs
}

// Resolve provides a package for the local folder or tarball provided
func (f *LocalFinder) Resolve(name string, version string) (*packages.Package, error) {
	//	make sure file exists
	_, err := f.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	// order of discovery
	// 1. tarball
	// 2. file based
	return reader.Read(f.fs, name)
}

// NewLocal creates a resolver for local operator package
func NewLocal() *LocalFinder {
	return &LocalFinder{fs: afero.NewOsFs()}
}
