package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdBundleReturnsCmd(t *testing.T) {

	newCmdBundle := newPackageCmd(fs, os.Stdout)

	if newCmdBundle.Parent() != nil {
		t.Fatal("We expect the newBundleInstall command to be returned")
	}

	sub := newCmdBundle
	for sub.HasSubCommands() {
		sub = sub.Commands()[0]
	}

	if reflect.ValueOf(sub.Flags().GetNormalizeFunc()).Pointer() != reflect.ValueOf(newCmdBundle.Flags().GetNormalizeFunc()).Pointer() {
		t.Fatal("child and root commands should have the same normalization functions")
	}
}

var bundleCmdArgs = []struct {
	arg          []string
	errorMessage string
}{
	{[]string{}, "expecting exactly one argument - directory of the operator to package"}, // 1
	{[]string{""}, "invalid operator in path: "},                                          // 2
	{[]string{"foo"}, "invalid operator in path: foo"},                                    // 3
	{[]string{"/opt/first-operator"}, ""},                                                 // 4
}

func TestTableNewBundleCmd(t *testing.T) {
	fs := afero.NewMemMapFs()
	copyOperatorToFs(fs, "../../../config/samples/first-operator")
	for _, test := range bundleCmdArgs {
		newCmdBundle := newPackageCmd(fs, os.Stdout)
		err := newCmdBundle.RunE(newCmdBundle, test.arg)
		if err != nil {
			assert.Equal(t, test.errorMessage, err.Error())
		}
	}
}

// copy from local file system into in mem
func copyOperatorToFs(fs afero.Fs, opath string) {

	dir := "/opt"
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
