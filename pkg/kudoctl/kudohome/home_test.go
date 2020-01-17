package kudohome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKudoHome(t *testing.T) {
	h := Home("/a")

	assert.Equal(t, "/a", h.String())
	assert.Equal(t, "/a/repository/repositories.yaml", h.RepositoryFile())
}
