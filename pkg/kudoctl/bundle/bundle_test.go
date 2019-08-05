package bundle

import (
	"io"
	"strings"
	"testing"
)

func TestToTarBundle(t *testing.T) {
	var noErrorTarballCreator = func(string, io.Writer) error {
		return nil
	}
	var tests = []struct {
		name string
		sourceFolder string
		targetFolder string
		overwrite bool
		err string
	} {
		{"valid package folder", "testdata/zk", ".", false, ""},
		{"nonexisting input folder", "testdata/XXXXX", ".", false, "invalid operator in path"},
		{"nonexisting output folder", "testdata/zk", "nonexistingfolder", false, "is not a proper directory"},
	}

	for _, tt := range tests {
		_, err := ToTarBundle(tt.sourceFolder, tt.targetFolder, false, noErrorTarballCreator)
		if err != nil && !strings.Contains(err.Error(), tt.err) {
			t.Errorf("%s: expecting error '%s' but got '%v'", tt.name, tt.err, err)
		} else if err == nil && tt.err != "" {
			t.Errorf("%s: expecting error %s but got none", tt.name, tt.err)
		}
	}
}