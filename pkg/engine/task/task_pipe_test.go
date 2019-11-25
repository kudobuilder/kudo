package task

import (
	"fmt"
	"testing"
)

func Test_isRelative(t *testing.T) {
	tests := []struct {
		base     string
		file     string
		relative bool
	}{
		{
			base:     "/dir",
			file:     "/dir/../link",
			relative: false,
		},
		{
			base:     "/dir",
			file:     "/dir/../../link",
			relative: false,
		},
		{
			base:     "/dir",
			file:     "/link",
			relative: false,
		},
		{
			base:     "/dir",
			file:     "/dir/link",
			relative: true,
		},
		{
			base:     "/dir",
			file:     "/dir/int/../link",
			relative: true,
		},
		{
			base:     "dir",
			file:     "dir/link",
			relative: true,
		},
		{
			base:     "dir",
			file:     "dir/int/../link",
			relative: true,
		},
		{
			base:     "dir",
			file:     "dir/../../link",
			relative: false,
		},
		{
			base:     "/tmp",
			file:     "/tmp/foo.txt",
			relative: true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.relative != isRelative(test.base, test.file) {
				t.Errorf("unexpected result for: base %q, file %q", test.base, test.file)
			}
		})
	}
}
