package client

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"gotest.tools/assert"
)

func TestKudoClientValidate(t *testing.T) {
	tests := []struct {
		err string
	}{
		{"CRDs invalid: failed to retrieve CRD"}, // verify that NewClient tries to validate CRDs
	}

	settings := env.Settings{KubeConfig: "testdata/test-config", Validate: true}

	for _, tt := range tests {
		_, err := GetValidatedClient(&settings)
		assert.ErrorContains(t, err, tt.err)
	}
}
