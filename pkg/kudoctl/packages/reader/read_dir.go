package reader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
)

// ReadPackage creates the implementation of the packages based on the path. The expectation is the packages
// is always local. The path can be relative or absolute location of the packages.
func ResourcesFromDir(fs afero.Fs, path string) (*packages.Resources, error) {
	// 1. get files
	files, err := PackageFilesFromDir(fs, path)
	if err != nil {
		return nil, fmt.Errorf("while parsing package files: %v", err)
	}

	// 2. get resources
	resources, err := convert.FilesToResources(files)
	if err != nil {
		return nil, fmt.Errorf("while getting package resources: %v", err)
	}

	return resources, nil
}

// PackageFilesFromDir walks the path provided and returns package files or an error
func PackageFilesFromDir(fs afero.Fs, packagePath string) (*packages.Files, error) {
	if packagePath == "" {
		return nil, fmt.Errorf("path must be specified")
	}

	if !filepath.IsAbs(packagePath) {
		// Normalize package path to provide more meaningful error messages
		absPackagePath, err := filepath.Abs(packagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize package path %s: %v", packagePath, err)
		}
		packagePath = absPackagePath
	}

	//	make sure file exists
	fi, err := fs.Stat(packagePath)
	if err != nil {
		clog.V(4).Printf("error reading package directory %s", packagePath)
		return nil, err
	}
	if !fi.IsDir() {
		clog.V(0).Printf("%s is not a package directory", packagePath)
		return nil, err
	}

	result := newPackageFiles()

	err = afero.Walk(fs, packagePath, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			// skip directories
			clog.V(6).Printf("folder walking through directory %v", file.Name())
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

		// Trim package path to use only package relative paths in parser
		path, err = filepath.Rel(packagePath, path)
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
		return nil, fmt.Errorf("operator package missing operator.yaml in %s", packagePath)
	}
	if result.Params == nil {
		return nil, fmt.Errorf("operator package missing params.yaml in %s", packagePath)
	}
	return &result, nil
}
