package kudohome

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestKudoHome(t *testing.T) {
	h := Home("/a")

	assert.Equal(t, h.String(), "/a")
	assert.Equal(t, h.RepositoryFile(), "/a/repository/repositories.yaml")
}
