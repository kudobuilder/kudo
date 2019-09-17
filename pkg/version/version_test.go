package version

import (
	"testing"

	"github.com/Masterminds/semver"
	"github.com/magiconair/properties/assert"
)

func Test_validVersion(t *testing.T) {

	tests := []struct {
		name     string
		actual   *semver.Version
		expected *semver.Version
		wantErr  bool
	}{
		{"expect early version", semver.MustParse("1.5"), semver.MustParse("1.4"), false},
		{"expect same version", semver.MustParse("1.5"), semver.MustParse("1.5"), false},
		{"expect newer version", semver.MustParse("1.5"), semver.MustParse("1.6"), true},
		{"full semver is not a factor", semver.MustParse("1.5.8"), semver.MustParse("1.5.0"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Valid("test", tt.actual, tt.expected); (err != nil) != tt.wantErr {
				t.Errorf("validVersionExpectations() error = %v, wantErr %v", err, tt.wantErr)
			}
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
		t.Run(tt.name, func(t *testing.T) {
			result := Clean(tt.actual)
			assert.Equal(t, result, tt.expected)
		})
	}
}
