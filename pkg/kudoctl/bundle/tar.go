package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// tarballWriter creates a tarball *.tgz file for the file system tree at the provided path.
func tarballWriter(fs afero.Fs, path string, w io.Writer) (err error) {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = afero.Walk(fs, path, func(file string, fi os.FileInfo, err error) error {

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

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'path' along the way, and writing any files
func Untar(fs afero.Fs, path string, r io.Reader) (err error) {
	//todo: refactor to combine with parseTarPackage
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		if ferr := gzr.Close(); ferr != nil {
			err = ferr
		}
	}()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(path, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			exists, err := afero.DirExists(fs, target)
			if err != nil {
				return err
			}
			if !exists {
				if err := fs.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := fs.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
