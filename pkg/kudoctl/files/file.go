package files

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// A collection of utility functions for working with files and Afero

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

// FullPathToTarget takes destination path and file name and provides a clean full path while optionally ensuring the file does not already exist
func FullPathToTarget(fs afero.Fs, destination string, name string, overwrite bool) (string, error) {
	if strings.Contains(destination, "~") {
		userHome, _ := os.UserHomeDir()
		destination = strings.Replace(destination, "~", userHome, 1)
	}
	destination, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	fi, err := fs.Stat(destination)
	if err != nil || !fi.Mode().IsDir() {
		return "", fmt.Errorf("destination \"%v\" is not a proper directory", destination)
	}
	target := filepath.Join(destination, name)
	if exists, _ := afero.Exists(fs, target); exists {
		if !overwrite {
			return "", fmt.Errorf("target file \"%v\" already exists", target)
		}
	}
	return target, nil
}

// Sha256Sum calculates and returns the sha256 checksum
func Sha256Sum(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Exists returns true if the path exists
func Exists(fs afero.Fs, path string) bool {
	exists, _ := afero.Exists(fs, path)
	return exists
}
