package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertYAMLParameters(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		expected map[string]interface{}
	}{
		{
			name:     "empty",
			params:   nil,
			expected: map[string]interface{}{},
		},
		{
			name: "string value",
			params: map[string]string{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			name: "YAML array",
			params: map[string]string{
				"foo": "- bar\n- baz",
			},
			expected: map[string]interface{}{
				"foo": []interface{}{"bar", "baz"},
			},
		},
		{
			name: "complex YAML",
			params: map[string]string{
				"foo": "hello: World\nports:\n  - name: HTTP\n    port: 80\n  - name: FTP\n    port: 21\n",
			},
			expected: map[string]interface{}{
				"foo": map[interface{}]interface{}{
					"hello": "World",
					"ports": []interface{}{
						map[interface{}]interface{}{"name": "HTTP", "port": 80},
						map[interface{}]interface{}{"name": "FTP", "port": 21},
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual, err := convertYAMLParameters(test.params)
		assert.NoError(t, err, test.name)
		assert.Equal(t, test.expected, actual, test.name)
	}
}
