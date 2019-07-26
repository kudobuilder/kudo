package cmd

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdBundleReturnsCmd(t *testing.T) {

	newCmdBundle := newBundleCmd(os.Stdout)

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
	{[]string{""}, "Error: expecting exactly one argument - directory of the operator to bundle"}, // 1
	{[]string{"foo"}, "Error: invalid operator"},                                                  // 2
	{[]string{"config/samples/first-operator"}, ""},                                               // 2
}

func TestTableNewBundlemCmd(t *testing.T) {
	for _, test := range bundleCmdArgs {
		newCmdBundle := newBundleCmd(os.Stdout)
		err := newCmdBundle.RunE(newCmdBundle, test.arg)
		assert.NotNil(t, err, test.errorMessage)
	}
}
