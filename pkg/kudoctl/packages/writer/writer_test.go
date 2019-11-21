package writer

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/spf13/afero"
)

const expectedTarballSHA = "ad0b1650b6f50979815acedae884851527b4e721696a7cc1d37fef3970888b19"

func TestRegularFileTarball(t *testing.T) {
	var fs = afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../testdata/zk", "/opt")

	f, _ := fs.Create("/opt/zk.tgz")

	o, _ := os.Open("/opt/zk/operator.yaml")
	expected, _ := files.Sha256Sum(o)

	// path is that copied into in-mem fs
	_ = TgzDir(fs, "/opt/zk", f)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	//open for reading in an untar
	f, _ = fs.Open("/opt/zk.tgz")
	defer f.Close()

	actualTarballSHA, err := files.Sha256Sum(f)
	if err != nil {
		t.Fatal(err)
	}

	if expectedTarballSHA != actualTarballSHA {
		t.Errorf("Expecting the tarball to have a specific (reproducible) hash but it differs: %v, %v", expectedTarballSHA, actualTarballSHA)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	if err := untar(fs, "/opt/untar", f); err != nil {
		t.Fatal(err)
	}

	u, _ := os.Open("/opt/untar/operator.yaml")
	actual, _ := files.Sha256Sum(u)

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
