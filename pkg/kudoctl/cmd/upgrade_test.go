package cmd

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestUpgradeCommand_Validation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		instanceName string
		err          string
	}{
		{"no argument", []string{}, "instance", "expecting exactly one argument - name of the package or path to upgrade"},
		{"too many arguments", []string{"aaa", "bbb"}, "instance", "expecting exactly one argument - name of the package or path to upgrade"},
		{"no instance name", []string{"arg"}, "", "please use --instance and specify instance name. It cannot be empty"},
	}

	for _, tt := range tests {
		cmd := newUpgradeCmd(afero.NewOsFs())
		cmd.SetArgs(tt.args)
		if tt.instanceName != "" {
			if err := cmd.Flags().Set("instance", tt.instanceName); err != nil {
				t.Fatal(err)
			}
		}
		_, err := cmd.ExecuteC()
		assert.EqualError(t, err, tt.err)
	}
}
