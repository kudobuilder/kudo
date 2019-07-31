package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var parameterParsingTests = []struct {
	paramStr string
	key      string
	value    string
	err      string
}{
	{"foo", "", "", "parameter not set: foo"},
	{"foo=", "", "", "parameter value can not be empty: foo="},
	{"=bar", "", "", "parameter name can not be empty: =bar"},
	{"foo=bar", "foo", "bar", ""},
}

func TestTableParameterParsing(t *testing.T) {
	for _, test := range parameterParsingTests {
		key, value, err := parseParameter(test.paramStr)
		assert.Equal(t, key, test.key)
		assert.Equal(t, value, test.value)
		if err != nil {
			assert.Equal(t, *err, test.err)
		}
	}
}
