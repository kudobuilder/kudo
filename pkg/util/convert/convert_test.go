package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYAMLArray(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  []interface{}
	}{
		{
			"empty input",
			"",
			false,
			nil,
		},
		{
			"single value",
			"foo",
			true,
			nil,
		},
		{
			"array",
			"[ a, b ]",
			false,
			[]interface{}{"a", "b"},
		},
		{
			"non-array",
			"a: b",
			true,
			nil,
		},
	}

	for _, test := range tests {
		actual, err := ToYAMLArray(test.input)

		if test.expectErr {
			assert.Error(t, err, test.name)
		} else {
			assert.NoError(t, err, test.name)
			assert.Equal(t, test.expected, actual, test.name)
		}
	}
}

func TestYAMLMap(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  interface{}
	}{
		{
			"empty input",
			"",
			false,
			nil,
		},
		{
			"single value",
			"foo",
			false,
			"foo",
		},
		{
			"array",
			"[ a, b ]",
			false,
			[]interface{}{"a", "b"},
		},
		{
			"map",
			"a: b",
			false,
			map[string]interface{}{"a": "b"},
		},
		{
			"complex map",
			"a: b\nc: [a, b: c]",
			false,
			map[string]interface{}{"a": "b", "c": []interface{}{"a", map[string]interface{}{"b": "c"}}},
		},
	}

	for _, test := range tests {
		actual, err := ToYAMLMap(test.input)

		if test.expectErr {
			assert.Error(t, err, test.name)
		} else {
			assert.NoError(t, err, test.name)
			assert.Equal(t, test.expected, actual, test.name)
		}
	}
}
