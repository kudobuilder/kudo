package cmd

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdDeleteReturnsCmd(t *testing.T) {
	newCmdDelete := NewDeleteCmd()

	if newCmdDelete.Parent() != nil {
		t.Fatal("We expect the newCmdDelete command to be returned")
	}

	sub := newCmdDelete
	for sub.HasSubCommands() {
		sub = sub.Commands()[0]
	}

	// In case of failure of this test check this PR: spf13/cobra#110
	if reflect.ValueOf(sub.Flags().GetNormalizeFunc()).Pointer() != reflect.ValueOf(newCmdDelete.Flags().GetNormalizeFunc()).Pointer() {
		t.Fatal("child and root commands should have the same normalization functions")
	}
}

func TestTableNewDeleteCmd_ArgsValidation(t *testing.T) {
	var tests = []struct {
		flags        []string
		errorMessage string
	}{
		{[]string{"foo", "bar"}, "more than one framework instance worked"}, // 1
		{[]string{}, "no framework instance worked"},                        // 2
	}

	for _, test := range tests {
		newCmdDelete := NewDeleteCmd()
		err := newCmdDelete.RunE(newCmdDelete, test.flags)
		assert.NotNil(t, err, test.errorMessage)
	}
}
