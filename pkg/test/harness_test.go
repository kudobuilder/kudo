package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTimeout(t *testing.T) {
	h := Harness{}
	assert.Equal(t, 30, h.GetTimeout())

	h.TestSuite.Timeout = 45
	assert.Equal(t, 45, h.GetTimeout())
}
