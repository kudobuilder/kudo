package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// This is an abstraction which abstracts the underlying bundle, which is likely file system or compressed file.
// There should be a complete separation between retrieving a bundle if not local and working with a bundle.

// Bundle is an abstraction of the collection of files that makes up a package.  It is anything we can retrieve the PackageCRDs from.
type Bundle interface {
	// transformed server view
	GetCRDs() (*PackageCRDs, error)
	// working with local package files
	GetPkgFiles() (*PackageFiles, error)
}

type tarBundle struct {
	reader io.Reader
}

type fileBundle struct {
	path string
	fs   afero.Fs
}

// NewBundle creates the implementation of the bundle based on the path.   The expectation is the bundle
// is always local .  The path can be relative or absolute location of the bundle.
func NewBundle(fs afero.Fs, path string) (Bundle, error) {
	//	make sure file exists
	fi, err := fs.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", path)
	}
	// order of discovery
	// 1. tarball
	// 2. file based
	if fi.Mode().IsRegular() && strings.HasSuffix(path, ".tar.gz") {
		r, err := getFileReader(fs, path)
		if err != nil {
			return nil, err
		}
		return tarBundle{r}, nil
	} else if fi.IsDir() {
		return fileBundle{path, fs}, nil
	} else {
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", path)
	}
}

func getFileReader(fs afero.Fs, path string) (io.Reader, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// NewBundleFromReader is a bundle from a reader.  This should only be used when a file cache isn't used.
func NewBundleFromReader(r io.Reader) Bundle {
	return tarBundle{r}
}

// GetPkgFiles returns the command side package files
func (b tarBundle) GetPkgFiles() (*PackageFiles, error) {
	return parseTarPackage(b.reader)
}

// GetCRDs returns the server side CRDs
func (b tarBundle) GetCRDs() (*PackageCRDs, error) {
	p, err := b.GetPkgFiles()
	if err != nil {
		return nil, errors.Wrap(err, "while extracting package files")
	}
	return p.getCRDs()
}

func (b fileBundle) GetCRDs() (*PackageCRDs, error) {
	p, err := b.GetPkgFiles()
	if err != nil {
		return nil, errors.Wrap(err, "while reading package from the file system")
	}
	return p.getCRDs()
}

func (b fileBundle) GetPkgFiles() (*PackageFiles, error) {
	return fromFolder(b.fs, b.path)
}

// ToTarBundle takes a path to operator files and creates a tgz of those files with the destination and name provided
func ToTarBundle(fs afero.Fs, path string, destination string, overwrite bool) (string, error) {
	pkg, err := fromFolder(fs, path)
	if err != nil {
		//TODO (kensipe): use wrapped err at high verbosity
		return "", fmt.Errorf("invalid operator in path: %v error: %v", path, err)
	}

	name := packageVersionedName(pkg)
	target, e := files.FullPathToTarget(fs, destination, fmt.Sprintf("%v.tgz", name), overwrite)
	if e != nil {
		return "", e
	}

	if _, err := fs.Stat(path); err != nil {
		return "", fmt.Errorf("unable to package files - %v", err.Error())
	}
	file, err := fs.Create(target)
	if err != nil {
		return "", err
	}
	defer file.Close()

	err = tarballWriter(fs, path, file)
	return target, err
}

// packageVersionedName provides the version name of a package provided a set of PackageFiles.  Ex. "zookeeper-0.1.0"
func packageVersionedName(pkg *PackageFiles) string {
	return fmt.Sprintf("%v-%v", pkg.Operator.Name, pkg.Operator.Version)
}

// fromFolder walks the path provided and returns CRD package files or an error
func fromFolder(fs afero.Fs, packagePath string) (*PackageFiles, error) {
	result := newPackageFiles()

	err := afero.Walk(fs, packagePath, func(path string, file os.FileInfo, err error) error {
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
		bytes, err := afero.ReadFile(fs, path)
		if err != nil {
			return err
		}

		return parsePackageFile(path, bytes, &result)
	})
	if err != nil {
		return nil, err
	}
	// final check
	if result.Operator == nil || result.Params == nil {
		return nil, fmt.Errorf("incomplete operator package in path: %v", packagePath)
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
				return nil, errors.Wrapf(err, "while reading file from package tarball %s", header.Name)
			}

			err = parsePackageFile(header.Name, bytes, &result)
			if err != nil {
				return nil, err
			}
		}
	}
}
