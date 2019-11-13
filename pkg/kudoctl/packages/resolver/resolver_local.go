package resolver

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
	"github.com/spf13/afero"
)

// LocalResolver will find local operator packages: folders or tgz
type LocalResolver struct {
	fs afero.Fs
}

// Resolve provides a package for the local folder or tarball provided
func (f *LocalResolver) Resolve(name string, version string) (*packages.Package, error) {
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
func NewLocal() *LocalResolver {
	return &LocalResolver{fs: afero.NewOsFs()}
}
