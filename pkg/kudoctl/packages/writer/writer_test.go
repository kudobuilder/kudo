package writer

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
)

const expectedTarballSHA = "0f5716c50420540e5ff60a18fe2c7aa231c0e554d359960929283375fcb98b4c"

func TestRegularFileTarball(t *testing.T) {
	var fs = afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../testdata/zk", "/opt")

	var err error
	f, err := fs.Create("/opt/zk.tgz")
	assert.NoError(t, err)

	o, err := fs.Open("/opt/zk/operator.yaml")
	assert.NoError(t, err)
	expected, err := files.Sha256Sum(o)
	assert.NoError(t, err)

	// path is that copied into in-mem fs
	err = TgzDir(fs, "/opt/zk", f)
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	//open for reading in an untar
	f, err = fs.Open("/opt/zk.tgz")
	assert.NoError(t, err)
	defer f.Close()

	actualTarballSHA, err := files.Sha256Sum(f)
	assert.NoError(t, err)

	if expectedTarballSHA != actualTarballSHA {
		t.Errorf("Expecting the tarball to have a specific (reproducible) hash but it differs: %v, %v", expectedTarballSHA, actualTarballSHA)
	}
	_, err = f.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	err = untar(fs, "/opt/untar", f)
	assert.NoError(t, err)

	u, err := fs.Open("/opt/untar/operator.yaml")
	assert.NoError(t, err)
	actual, err := files.Sha256Sum(u)
	assert.NoError(t, err)

	if expected != actual {
		t.Errorf("Expecting the tarball and untar of operator.yaml to have same hash but they differ: %v, %v", expected, actual)
	}
}

// untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'path' along the way, and writing any files
func untar(fs afero.Fs, path string, r io.Reader) (err error) {
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
