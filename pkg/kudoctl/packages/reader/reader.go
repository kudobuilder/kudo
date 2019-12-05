package reader

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// ReadPackage creates the implementation of the packages based on the path. The expectation is the packages
// is always local. The path can be relative or absolute location of the packages. Order of the discovery is:
// 1. tarball
// 2. dir based
func Read(fs afero.Fs, path string) (*packages.Package, error) {
	//	make sure file exists
	fi, err := fs.Stat(path)
	if err != nil {
		return nil, err
	}
	clog.V(0).Printf("determining package type of %v", path)

	if fi.Mode().IsRegular() && strings.HasSuffix(path, ".tgz") {
		clog.V(0).Printf("%v is a tgz package", path)
		return ReadTar(fs, path)
	} else if fi.IsDir() {
		clog.V(0).Printf("%v is a file package", path)
		return ReadDir(fs, path)
	} else {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a *.tgz file or a folder", path)
	}
}
