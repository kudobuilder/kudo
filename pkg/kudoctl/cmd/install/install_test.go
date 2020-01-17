package install

import (
	"testing"

	"gotest.tools/assert"
)

func TestValidate(t *testing.T) {

	tests := []struct {
		arg []string
		err string
	}{
		{nil, "expecting exactly one argument - name of the package or path to install"},                     // 1
		{[]string{"arg", "arg2"}, "expecting exactly one argument - name of the package or path to install"}, // 2
		{[]string{}, "expecting exactly one argument - name of the package or path to install"},              // 3
	}

	for _, tt := range tests {
		err := validate(tt.arg)
		if tt.err != "" {
			assert.ErrorContains(t, err, tt.err)
		}
	}
}
