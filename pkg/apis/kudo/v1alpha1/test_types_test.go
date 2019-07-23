package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetManifestsDirs(t *testing.T) {
	ts := TestSuite{}
	assert.Equal(t, []string{}, ts.GetManifestsDirs())

	ts = TestSuite{
		ManifestsDir: "./hello",
	}
	assert.Equal(t, []string{"./hello"}, ts.GetManifestsDirs())

	ts = TestSuite{
		ManifestsDirs: []string{
			"./hello",
		},
	}
	assert.Equal(t, []string{"./hello"}, ts.GetManifestsDirs())

	ts = TestSuite{
		ManifestsDirs: []string{
			"./hello",
		},
		ManifestsDir: "./world",
	}
	assert.Equal(t, []string{"./hello", "./world"}, ts.GetManifestsDirs())
}
