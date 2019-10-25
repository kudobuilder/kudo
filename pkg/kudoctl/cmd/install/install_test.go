package install

import (
	"testing"
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
		err := validate(tt.arg, DefaultOptions)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("Expecting error message '%s' but got '%s'", tt.err, err)
			}
		}
	}
}
