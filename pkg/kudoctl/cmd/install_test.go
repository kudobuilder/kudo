package cmd

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdInstallReturnsCmd(t *testing.T) {

	newCmdInstall := NewInstallCmd()

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

var parameterTests = []struct {
	flags        []string
	errorMessage string
}{
	{[]string{"foo"}, "a parameter without value worked"},                           // 1
	{[]string{"bar="}, "a parameter with empty value worked"},                       // 2
	{[]string{"foo=bar", "fiz="}, "one of many parameters with empty value worked"}, // 3
	{[]string{"foo", "bar"}, "multiple empty parameters worked"},                    // 4
}

func TestTableNewInstallCmd_WithParameters(t *testing.T) {
	for _, test := range parameterTests {
		newCmdInstall := NewInstallCmd()
		for _, flag := range test.flags {
			newCmdInstall.Flags().Set("parameter", flag)
		}
		err := newCmdInstall.RunE(newCmdInstall, []string{})
		assert.NotNil(t, err, test.errorMessage)
	}
}
