package writer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// WriteTgz takes a path to operator files and creates a tgz of those files with the destination and name provided
func WriteTgz(fs afero.Fs, path string, destination string, overwrite bool) (target string, err error) {
	pkg, err := reader.FromDir(fs, path)
	if err != nil {
		return "", fmt.Errorf("invalid operator in path: %v error: %w", path, err)
	}

	name := packageVersionedName(pkg)
	target, err = files.FullPathToTarget(fs, destination, fmt.Sprintf("%v.tgz", name), overwrite)
	if err != nil {
		return "", err
	}

	if _, err := fs.Stat(path); err != nil {
		return "", fmt.Errorf("unable to package files - %v", err.Error())
	}
	file, err := fs.Create(target)
	if err != nil {
		return "", err
	}
	defer func() {
		if ferr := file.Close(); ferr != nil {
			err = ferr
		}
	}()

	err = TgzDir(fs, path, file)
	return target, err
}

// TgzDir writes a tarball *.tgz file to a 'io.Writer' for the file system tree at the provided path.
func TgzDir(fs afero.Fs, path string, w io.Writer) (err error) {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	return Tar(fs, path, gw)
}

// Tar writes a tarball to a 'io.Writer' for the file system tree at the provided path.
func Tar(fs afero.Fs, root string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	err := afero.Walk(fs, root, func(file string, fi os.FileInfo, err error) error {
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
		header.Name, err = filepath.Rel(root, file)
		if err != nil {
			fmt.Printf("Error setting tar file name for: %v", file)
		}

		// change certain header metadata to make the build reproducible
		header.ModTime = time.Time{}
		header.Uid = 0
		header.Gid = 0
		header.Uname = "root"
		header.Gname = "root"

		// tar_zcvf the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// open files for taring
		f, err := fs.Open(file)
		if err != nil {
			return err
		}

		defer func() {
			if ferr := f.Close(); ferr != nil {
				err = ferr
			}
		}()

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
	return err
}

// packageVersionedName provides the version name of a package provided a set of Files.  Ex. "zookeeper-0.1.0"
func packageVersionedName(pkg *packages.Files) string {
	return fmt.Sprintf("%v-%v", pkg.Operator.Name, pkg.Operator.Version)
}
