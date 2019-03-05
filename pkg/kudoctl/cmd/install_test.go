package cmd

import (
	"reflect"
	"testing"
)

func TestNewCmdInstallReturnsCmd(t *testing.T) {

	newCmdInstall := NewCmdInstall()

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
