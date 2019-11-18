package reader

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func ReadTar(fs afero.Fs, path string) (*packages.Package, error) {
	// 1. read the tarball
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)

	// 2. ParseTgz tar files
	files, err := ParseTgz(buf)
	if err != nil {
		return nil, errors.Wrap(err, "while parsing package files")
	}

	// 3. convert to resources
	resources, err := files.Resources()
	if err != nil {
		return nil, errors.Wrap(err, "while getting package resources")
	}

	return &packages.Package{
		Resources: resources,
		Files:     files,
	}, nil
}

func ParseTgz(r io.Reader) (*packages.Files, error) {
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

		case tar.TypeReg:
			buf, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, errors.Wrapf(err, "while reading file from package tarball %s", header.Name)
			}

			err = parsePackageFile(header.Name, buf, &result)
			if err != nil {
				return nil, err
			}
		}
	}
}
