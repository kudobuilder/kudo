package cmd

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdBundleReturnsCmd(t *testing.T) {

	newCmdBundle := newPackageCmd(os.Stdout)

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
	{[]string{"../../../config/samples/first-operator"}, ""},                              // 4
}

func TestTableNewBundleCmd(t *testing.T) {
	rmOperator()
	defer rmOperator()
	for _, test := range bundleCmdArgs {
		newCmdBundle := newPackageCmd(os.Stdout)
		err := newCmdBundle.RunE(newCmdBundle, test.arg)
		if err != nil {
			assert.Equal(t, test.errorMessage, err.Error())
		}
	}

}

func rmOperator() {
	fmt.Println("Cleaning up")
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(dir)
	o := "first-operator-0.1.0.tgz"
	if _, err := os.Stat(o); !os.IsNotExist(err) {
		err := os.Remove(o)
		if err != nil {
			fmt.Println(fmt.Sprintf("WARNING: unable to delete operator: %v", o))
		}
	}
}
