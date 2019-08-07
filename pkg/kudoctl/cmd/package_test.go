package cmd

import (
	"os"
	"reflect"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdBundleReturnsCmd(t *testing.T) {

	packageCmd := newPackageCmd(fs, os.Stdout)

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
	{[]string{"/opt/zk"}, ""},                                                             // 4
}

func TestTableNewBundleCmd(t *testing.T) {
	fs := afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../bundle/testdata/zk", "/opt")
	for _, test := range bundleCmdArgs {
		newCmdBundle := newPackageCmd(fs, os.Stdout)
		err := newCmdBundle.RunE(newCmdBundle, test.arg)
		if err != nil {
			assert.Equal(t, test.errorMessage, err.Error())
		}
	}
}
