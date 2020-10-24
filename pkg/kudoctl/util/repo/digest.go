package repo

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// PackageFilesDigest is a tuple of data used to return the package files AND the digest of a tarball
type PackageFilesDigest struct {
	PackageFiles *packages.Files
	Digest       string
}

// filesDigest maps []string of paths to the [] Operators
func filesDigest(fs afero.Fs, paths []string) []*PackageFilesDigest {
	return mapPaths(fs, paths, pathToOperator)
}

// work of map path, swallows errors to return only packages that are valid
func mapPaths(fs afero.Fs, paths []string, f func(afero.Fs, string) (*PackageFilesDigest, error)) []*PackageFilesDigest {
	ops := make([]*PackageFilesDigest, 0)
	for _, path := range paths {
		op, err := f(fs, path)
		if err != nil {
			clog.Printf("WARNING: operator: %v is invalid", path)
			continue
		}
		ops = append(ops, op)
	}

	return ops
}

// pathToOperator takes a single path and returns an operator or error
func pathToOperator(fs afero.Fs, path string) (pfd *PackageFilesDigest, err error) {
	fsreader, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ferr := fsreader.Close(); ferr != nil {
			err = ferr
		}
	}()

	digest, err := files.Sha256Sum(fsreader)
	if err != nil {
		return nil, err
	}
	// restart reading of file after getting digest
	_, err = fsreader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(fsreader)
	if err != nil {
		return nil, err
	}

	files, err := reader.PackageFilesFromTar(afero.NewMemMapFs(), bytes.NewBuffer(b))
	pfd = &PackageFilesDigest{
		files,
		digest,
	}
	return pfd, err
}
