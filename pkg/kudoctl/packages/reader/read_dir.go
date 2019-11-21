package reader

import (
	"os"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// ReadPackage creates the implementation of the packages based on the path. The expectation is the packages
// is always local. The path can be relative or absolute location of the packages.
func ReadDir(fs afero.Fs, path string) (*packages.Package, error) {
	//	make sure file exists
	fi, err := fs.Stat(path)
	if err != nil {
		clog.V(4).Printf("error reading package directory %s", path)
		return nil, err
	}
	if !fi.IsDir() {
		clog.V(0).Printf("%s is not a package directory", path)
		return nil, err
	}

	// 1. get files
	files, err := FromDir(fs, path)
	if err != nil {
		return nil, errors.Wrap(err, "while parsing package files")
	}

	// 2. get resources
	resources, err := files.Resources()
	if err != nil {
		return nil, errors.Wrap(err, "while getting package resources")
	}

	return &packages.Package{
		Resources: resources,
		Files:     files,
	}, nil
}

// FromDir walks the path provided and returns package files or an error
func FromDir(fs afero.Fs, packagePath string) (*packages.Files, error) {
	if packagePath == "" {
		return nil, errors.New("path must be specified")
	}
	result := newPackageFiles()

	err := afero.Walk(fs, packagePath, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			// skip directories
			clog.V(6).Printf("folder walking skipping directory %v", file)
			return nil
		}
		if path == packagePath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		buf, err := afero.ReadFile(fs, path)
		if err != nil {
			return err
		}

		return parsePackageFile(path, buf, &result)
	})
	if err != nil {
		return nil, err
	}
	// final check
	if result.Operator == nil {
		return nil, errors.New("operator package missing operator.yaml")
	}
	if result.Params == nil {
		return nil, errors.New("operator package missing params.yaml")
	}
	return &result, nil
}
