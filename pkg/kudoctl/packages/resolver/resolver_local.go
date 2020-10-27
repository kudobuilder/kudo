package resolver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// LocalHelper will find local operator packages: folders or tgz
type LocalHelper struct {
	fs        afero.Fs
	directory string
}

// newForFilesystem creates a resolver with an fs and a working directory
func newForFilesystem(fs afero.Fs, dir string) *LocalHelper {
	return &LocalHelper{
		fs:        fs,
		directory: dir,
	}
}

func (f *LocalHelper) LocalPackagePath(path string) (string, error) {
	fi, err := f.fs.Stat(path)

	// force local operators usage to be either absolute or express a relative path
	// or put another way, a name can NOT be mistaken to be the name of a local folder
	if filepath.Base(path) != path && err == nil {
		var abs string
		abs, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}

		// expecting either a folder or .tgz file
		if fi.IsDir() || (fi.Mode().IsRegular() && strings.HasSuffix(abs, ".tgz")) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("%s is not a valid local package path", path)
}

func (f *LocalHelper) ResolveTar(out afero.Fs, path string) (*packages.Resources, error) {
	abs, err := f.LocalPackagePath(path)
	if strings.HasSuffix(abs, ".tgz") && err == nil {
		return reader.ResourcesFromTar(f.fs, out, path)
	}

	return nil, fmt.Errorf("unsupported file system format %v. Expect a *.tgz file", abs)
}

func (f *LocalHelper) ResolveDir(path string) (*packages.Resources, error) {
	abs, err := f.LocalPackagePath(path)
	if err == nil {
		return reader.ResourcesFromDir(f.fs, path)
	}
	return nil, fmt.Errorf("unsupported file system format %v. Expect a folder", abs)
}
