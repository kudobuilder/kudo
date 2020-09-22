package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validVersion(t *testing.T) {

	tests := []struct {
		name     string
		actual   *Version
		expected *Version
		val      int
	}{
		{"expect early version", MustParse("1.5"), MustParse("1.4"), -1},
		{"expect same version", MustParse("1.5"), MustParse("1.5"), 0},
		{"expect newer version", MustParse("1.5"), MustParse("1.6"), 1},
		{"full semver is not a factor", MustParse("1.5.8"), MustParse("1.5.0"), 0},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			val := tt.expected.CompareMajorMinor(tt.actual)
			assert.Equal(t, val, tt.val)
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"clean ver", "1.0.0", "1.0.0"},
		{"clean ver", "v1.0.0", "1.0.0"},
		{"short ver", "v1.0", "1.0"},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			result := Clean(tt.actual)
			assert.Equal(t, tt.expected, result)
		})
	}
}
