package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdInstallReturnsCmd(t *testing.T) {

	newCmdInstall := newInstallCmd(afero.NewOsFs())

	if newCmdInstall.Parent() != nil {
		t.Fatal("We expect the newCmdInstall command to be returned")
	}

	sub := newCmdInstall
	for sub.HasSubCommands() {
		sub = sub.Commands()[0]
	}

	// In case of failure of this test check this PR: spf13/cobra#110
	if reflect.ValueOf(sub.Flags().GetNormalizeFunc()).Pointer() != reflect.ValueOf(newCmdInstall.Flags().GetNormalizeFunc()).Pointer() {
		t.Fatal("child and root commands should have the same normalization functions")
	}
}

var cmdParameterTests = []struct {
	flags        map[string]string
	parameters   []string
	errorMessage string
}{
	{map[string]string{}, []string{"foo"}, "a parameter without value worked"},                                                           // 1
	{map[string]string{}, []string{"bar="}, "a parameter with empty value worked"},                                                       // 2
	{map[string]string{}, []string{"foo=bar", "fiz="}, "one of many arguments with empty value worked"},                                  // 3
	{map[string]string{}, []string{"foo", "bar"}, "multiple empty arguments worked"},                                                     // 4
	{map[string]string{}, []string{}, "get flag: flag accessed but not defined: kubeconfig"},                                             // 5
	{map[string]string{"kubeconfig": "/tmp"}, []string{}, "could not check kubeconfig path: getting config failed: /tmp is a directory"}, // 6
}

func TestTableNewInstallCmd_WithParameters(t *testing.T) {
	for _, test := range cmdParameterTests {
		newCmdInstall := newInstallCmd(afero.NewOsFs())
		for _, flag := range test.parameters {
			if err := newCmdInstall.Flags().Set("parameter", flag); err != nil {
				t.Fatal(err)
			}
		}
		err := newCmdInstall.RunE(newCmdInstall, []string{})
		assert.NotNil(t, err, test.errorMessage)
	}
}
