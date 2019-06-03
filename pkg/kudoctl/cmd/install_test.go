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

func TestNewCmdInstall_WithParameters(t *testing.T) {
	newCmdInstall := NewInstallCmd()

	newCmdInstall.Flags().Set("parameter", "foo")
	err := newCmdInstall.RunE(newCmdInstall, []string{})
	assert.NotNil(t, err, "a parameter without value worked")

	newCmdInstall = NewInstallCmd()
	newCmdInstall.Flags().Set("parameter", "bar=")
	err = newCmdInstall.RunE(newCmdInstall, []string{})
	assert.NotNil(t, err, "a parameter with empty value worked")

	newCmdInstall = NewInstallCmd()
	newCmdInstall.Flags().Set("parameter", "foo=bar")
	newCmdInstall.Flags().Set("parameter", "fiz=")
	err = newCmdInstall.RunE(newCmdInstall, []string{})
	assert.NotNil(t, err, "one of many parameters with empty value worked")
}
