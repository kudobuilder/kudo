package files

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// CopyOperatorToFs used with afero usually for tests to copy files into a filesystem.
// copy from local file system into in mem
func CopyOperatorToFs(fs afero.Fs, opath string, base string) {

	dir := filepath.Clean(base)
	failed := false
	err := fs.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Println("FAILED: ", err)
		failed = true
	}
	filepath.Walk(opath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// directory copy
		if info.IsDir() {
			if dir != info.Name() {
				dir = filepath.Join(dir, info.Name())
				err := fs.MkdirAll(dir, 0755)
				if err != nil {
					fmt.Println("FAILED: ", err)
					failed = true
				}
			}
			return nil
		}

		if failed {
			return errors.New("unable to write file, as mkdir failed")
		}

		fn := filepath.Join(dir, info.Name())
		fmt.Println(fn)
		w, _ := fs.Create(fn)
		r, _ := os.Open(path)
		io.Copy(w, r)

		return nil
	})
}

// Sha256Sum calculates and returns the sha256 checksum
func Sha256Sum(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
