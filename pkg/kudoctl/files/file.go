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
	baseDir := filepath.Clean(base)
	root, _ := filepath.Split(opath)
	failed := false
	err := fs.MkdirAll(baseDir, 0755)
	if err != nil {
		fmt.Println("FAILED: ", err)
		failed = true
	}
	err = filepath.Walk(opath, func(path string, info os.FileInfo, e error) (err error) {
		if e != nil {
			return e
		}
		// remove original path base
		fileBase := filepath.Clean(strings.Replace(path, root, "", 1))
		// add to new base dir
		file := filepath.Join(baseDir, fileBase)
		// directory copy
		if info.IsDir() {
			err := fs.MkdirAll(file, 0755)
			if err != nil {
				fmt.Println("FAILED: ", err)
				failed = true
			}
			return nil
		}

		if failed {
			return errors.New("unable to write file, as mkdir failed")
		}

		fmt.Println(file)

		w, err := fs.Create(file)
		if err != nil {
			fmt.Println("FAILED: ", err)
			return fmt.Errorf("unable to create file %s", file)
		}
		defer func() {
			if ferr := w.Close(); ferr != nil {
				fmt.Println("FAILED: ", err)
				err = fmt.Errorf("unable to close file %s", file)
			}
		}()

		r, err := os.Open(path)
		if err != nil {
			fmt.Println("FAILED: ", err)
			return fmt.Errorf("unable to open file %s", path)
		}
		defer func() {
			if ferr := r.Close(); ferr != nil {
				fmt.Println("FAILED: ", err)
				err = fmt.Errorf("unable to close file %s", path)
			}
		}()

		_, err = io.Copy(w, r)
		if err != nil {
			fmt.Println("FAILED: ", err)
			return fmt.Errorf("unable to copy from %s into %s", file, path)
		}

		return nil
	})

	if err != nil {
		fmt.Println("failure while copying operator to filesystem: ", err)
	}
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
