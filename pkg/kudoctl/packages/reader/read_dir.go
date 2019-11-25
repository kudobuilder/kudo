package reader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/spf13/afero"
)

// ReadPackage creates the implementation of the packages based on the path. The expectation is the packages
// is always local. The path can be relative or absolute location of the packages.
func ReadDir(fs afero.Fs, path string) (*packages.Package, error) {
	//	make sure file exists
	fi, err := fs.Stat(path)
	if err != nil {
		clog.V(4).Printf("error reading package directory %v", path)
		return nil, err
	}
	if !fi.IsDir() {
		clog.V(0).Printf("%s is not a package directory", path)
		return nil, err
	}

	// 1. get files
	files, err := FromDir(fs, path)
	if err != nil {
		return nil, fmt.Errorf("while parsing package files: %w", err)
	}

	// 2. get resources
	resources, err := files.Resources()
	if err != nil {
		return nil, fmt.Errorf("while getting package resources: %w", err)
	}

	return &packages.Package{
		Resources: resources,
		Files:     files,
	}, nil
}

// FromDir walks the path provided and returns package files or an error
func FromDir(fs afero.Fs, packagePath string) (*packages.Files, error) {
	if packagePath == "" {
		return nil, fmt.Errorf("path must be specified")
	}

	if !filepath.IsAbs(packagePath) {
		// Normalize package path to provide more meaningful error messages
		absPackagePath, err := filepath.Abs(packagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize package path %v: %w", packagePath, err)
		}
		packagePath = absPackagePath
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
		return nil, fmt.Errorf("operator package missing operator.yaml in %v", packagePath)
	}
	if result.Params == nil {
		return nil, fmt.Errorf("operator package missing params.yaml in %v", packagePath)
	}
	return &result, nil
}
