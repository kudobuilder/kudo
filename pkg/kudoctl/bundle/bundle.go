package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// This is an abstraction which abstracts the underlying bundle, which is likely file system or compressed file.
// There should be a complete separation between retrieving a bundle if not local and working with a bundle.

// Bundle is an abstraction of the collection of files that makes up a package.  It is anything we can retrieve the PackageCRDs from.
type Bundle interface {
	GetCRDs() (*PackageCRDs, error)
}

type tarBundle struct {
	reader io.Reader
}

type fileBundle struct {
	path string
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

func (b tarBundle) GetCRDs() (*PackageCRDs, error) {

	p, err := parseTarPackage(b.reader)
	if err != nil {
		return nil, errors.Wrap(err, "while extracting package files")
	}
	return p.getCRDs()
}

func (b fileBundle) GetCRDs() (*PackageCRDs, error) {
	p, err := fromFolder(b.path)
	if err != nil {
		return nil, errors.Wrap(err, "while reading package from the file system")
	}
	return p.getCRDs()
}

// ToTarBundle takes a path to operator files and creates a tgz of those files with the destination and name provided
func ToTarBundle(path string, destination string, overwrite bool) (string, error) {
	pkg, err := fromFolder(path)
	if err != nil {
		return "", fmt.Errorf("invalid operator in path: %v", path)
	}

	name := packageVersionedName(pkg)
	target, e := getFullPathToTarget(destination, name, overwrite)
	if e != nil {
		return "", e
	}

	//validate it is an operator
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("unable to bundle files - %v", err.Error())
	}
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// return on non-regular files.  We don't add directories without files and symlinks
		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			fmt.Printf("Error creating tar header for: %v", fi.Name())
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, path, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; deferring would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
	return target, err
}
// getFullPathToTarget takes destination path and file name and provides a clean full path while ensure the file does not exist.
func getFullPathToTarget(destination string, name string, overwrite bool) (string, error) {
	if destination == "." {
		destination = ""
	}
	if destination != "" {
		fi, err := os.Stat(destination)
		if err != nil || !fi.Mode().IsDir() {
			return "", fmt.Errorf("destination \"%v\" is not a proper directory", destination)
		}
		name = fmt.Sprintf("%v/%v", destination, name)
	}
	target := filepath.Clean(fmt.Sprintf("%v.tgz", name))
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		if !overwrite {
			return "", fmt.Errorf("target file exists. Remove or --overwrite. File:%v", target)
		}
	}
	return target, nil
}

// packageVersionedName provides the version name of a package provided a set of PackageFiles.  Ex. "zookeeper-0.1.0"
func packageVersionedName(pkg *PackageFiles) string {
	return fmt.Sprintf("%v-%v", pkg.Operator.Name, pkg.Operator.Version)
}

// fromFolder walks the path provided and returns CRD package files or an error
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
	// final check
	if result.Operator == nil || result.Params == nil {
		return nil, fmt.Errorf("incomplete operator bundle in path: %v", packagePath)
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
