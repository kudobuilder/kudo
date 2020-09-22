package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {

	tests := []struct {
		args []string
		opts *Options
		err  string
	}{
		{args: nil, opts: &Options{}, err: "expecting exactly one argument - name of the package or path to install"},
		{args: []string{"arg", "arg2"}, opts: &Options{}, err: "expecting exactly one argument - name of the package or path to install"},
		{args: []string{}, opts: &Options{}, err: "expecting exactly one argument - name of the package or path to install"},
		{args: []string{"arg"}, opts: &Options{
			SkipInstance: true,
			InCluster:    true,
		}, err: "you can't use skip-instance or repo option when installing from in-cluster operators"},
	}

	for _, tt := range tests {
		err := validate(tt.args, tt.opts)
		if tt.err != "" {
			assert.EqualError(t, err, tt.err)
		}
	}
}
