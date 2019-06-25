package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// This is an abstraction which abstracts the underlying bundle, which is likely file system or compressed file.
// There should be a complete separation between retrieving a bundle if not local and working with a bundle.

// todo: Does it belong in this package?   I think it should be separate from the repo... more thoughts   I think bundle and package are tightly coupled and repo is not.

// Bundle is an abstraction of the collection of files that makes up a package.  It is anything we can retrieve the PackageCRDs from.
type Bundle interface {
	GetCRDs() (*PackageCRDs, error)
}

// NewBundle creates the implementation of the bundle based on the path.   The expectation is the bundle
// is always local .  The path can be relative or absolute location of the bundle.
func NewBundle(path string) (Bundle, error) {
	//	make sure file exists
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", path)
	}
	// order of discovery
	// 1. tarball
	// 2. file based
	if fi.Mode().IsRegular() && strings.HasSuffix(path, ".tar.gz") {
		r, err := getFileReader(path)
		if err != nil {
			return nil, err
		}
		return tarBundle{r}, nil
	} else if fi.IsDir() {
		return fileBundle{path}, nil
	} else {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", path)
	}
}

func getFileReader(path string) (io.Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// NewBundleFromReader is a bundle from a reader.  This should only be used when a file cache isn't used.
func NewBundleFromReader(r io.Reader) Bundle {
	return tarBundle{r}
}

type tarBundle struct {
	reader io.Reader
}

func (b tarBundle) GetCRDs() (*PackageCRDs, error) {

	p, err := parseTarPackage(b.reader)
	if err != nil {
		return nil, errors.Wrap(err, "while extracting package files")
	}
	return p.getCRDs()
}

type fileBundle struct {
	path string
}

func (b fileBundle) GetCRDs() (*PackageCRDs, error) {
	p, err := fromFolder(b.path)
	if err != nil {
		return nil, errors.Wrap(err, "while reading package from the file system")
	}
	return p.getCRDs()
}

func fromFolder(packagePath string) (*PackageFiles, error) {
	result := newPackageFiles()
	err := filepath.Walk(packagePath, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			// skip directories
			return nil
		}
		if path == packagePath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		return parsePackageFile(path, bytes, &result)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func parseTarPackage(r io.Reader) (*PackageFiles, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := gzr.Close()
		if err != nil {
			fmt.Printf("Error when closing gzip reader: %s", err)
		}
	}()

	tr := tar.NewReader(gzr)

	result := newPackageFiles()
	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return &result, nil

		// return any other error
		case err != nil:
			return nil, err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// check the file type
		switch header.Typeflag {

		case tar.TypeDir:
			// we don't need to handle folders, files have folder name in their names and that should be enough

		// if it's a file create it
		case tar.TypeReg:
			bytes, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, errors.Wrapf(err, "while reading file from bundle tarball %s", header.Name)
			}

			err = parsePackageFile(header.Name, bytes, &result)
			if err != nil {
				return nil, err
			}
		}
	}
}
