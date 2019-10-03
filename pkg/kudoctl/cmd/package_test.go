package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var packageCmdArgs = []struct {
	name         string
	arg          []string
	errorMessage string
}{
	{"expect exactly one argument", []string{}, "expecting exactly one argument - directory of the operator to package"}, // 1
	{"empty string argument", []string{""}, "invalid operator in path:  error: path must be specified"},                  // 2
	{"invalid operator", []string{"foo"}, "invalid operator in path: foo error: open foo: file does not exist"},          // 3
	{"valid operator", []string{"/opt/zk"}, ""},                                                                          // 4
}

func TestTableNewPackageCmd(t *testing.T) {
	fs := afero.NewMemMapFs()
	testdir, _ := filepath.Abs("")
	fs.Mkdir(testdir, 0777)
	files.CopyOperatorToFs(fs, "../packages/testdata/zk", "/opt")
	for _, test := range packageCmdArgs {
		newCmd := newPackageCmd(fs, os.Stdout)
		err := newCmd.RunE(newCmd, test.arg)
		if err != nil {
			assert.Equal(t, test.errorMessage, err.Error(), test.name)
		}
	}
}
