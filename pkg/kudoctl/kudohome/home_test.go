package kudohome

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_ = flag.Bool("update", false, "update .golden files")
)

func TestKudoHome(t *testing.T) {
	h := Home("/a")

	assert.Equal(t, "/a", h.String())
	assert.Equal(t, "/a/repository/repositories.yaml", h.RepositoryFile())
}
