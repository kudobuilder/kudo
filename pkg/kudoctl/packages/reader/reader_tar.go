package reader

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
)

// ResourcesFromTar extracts files from the tar provides by he `inFs` and the `path` and converts
// them to resources. All the extracted files are saved in the `outFs` for later use (e.g. searching
// for local dependencies)
func ResourcesFromTar(in afero.Fs, out afero.Fs, path string) (*packages.Resources, error) {
	// 1. read the tarball
	b, err := afero.ReadFile(in, path)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)

	// 2. extract and parse tar files
	files, err := PackageFilesFromTar(out, buf)
	if err != nil {
		return nil, fmt.Errorf("while parsing package files from %s: %v", path, err)
	}

	// 3. convert to resources
	resources, err := convert.FilesToResources(files)
	if err != nil {
		return nil, fmt.Errorf("while getting package resources from %s: %v", path, err)
	}

	return resources, nil
}

// PackageFilesFromTar extracts a tgz archive held by passed reader and returns the package files.
// Additionally, all the files are saved in the `out` Fs (in  the root `/` folder).
func PackageFilesFromTar(out afero.Fs, r io.Reader) (*packages.Files, error) {
	err := ExtractTar(out, r)
	if err != nil {
		return nil, err
	}

	pf, err := PackageFilesFromDir(out, "/")
	return pf, err
}

// ExtractTar extract a tgz archive into the given filesystem. This is a generic extract method
// so no package parsing is performed.
// *Note:* all file paths are prepended by `/` and are extracted into the root of the passed Fs.
// By default tar strips out the leading slash, but leaves `./` when packing a folder and doesn't
// add it when packing a file so that depending on how it was packed the same file might have a path
// like `templates/foo.yaml` or `./templates/foo.yaml`. Since we're extracting into the empty MemFs,
// we can avoid the inconsistency and just extract into the root.
func ExtractTar(out afero.Fs, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		err := gzr.Close()
		if err != nil {
			clog.Printf("Error when closing gzip reader: %s", err)
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

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// check the file type
		switch header.Typeflag {

		// there are no folders in the tar, only files with nested file names e.g. `templates/foo.yaml` ¯\_(ツ)_/¯
		case tar.TypeDir:
			clog.Printf("Tar file contained directory. Did not expect this: %s", header.Name)
			continue

		case tar.TypeReg:
			// read the file
			buf, err := ioutil.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("while reading file %s from package tarball: %v", header.Name, err)
			}

			// copy over contents. the files are extracted into the root of the passed Fs
			// nolint:gosec
			err = afero.WriteFile(out, path.Join("/", header.Name), buf, 0644)
			if err != nil {
				return fmt.Errorf("while writing file %s: %v", header.Name, err)
			}
		}
	}
}
