package engine

import (
	"fmt"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		expected string
	}{
		{name: "empty", template: "", expected: ""},
		{name: "basic template", template: "name: {{ .Params.Name }}", params: map[string]interface{}{"Name": "Some Name"}, expected: "name: Some Name"},
		{
			name:     "nested template",
			template: "name: {{ .Params.User.Name }}",
			params: map[string]interface{}{
				"User": map[string]interface{}{"Name": "Bob User"},
			},
			expected: "name: Bob User"},
		{name: "function", template: "name: {{ .Params.Name | upper }}", params: map[string]interface{}{"Name": "hello"}, expected: "name: HELLO"},
	}

	engine := New()

	for _, test := range tests {
		params := map[string]interface{}{}

		for k, v := range test.params {
			params[k] = v
		}

		vals := map[string]interface{}{
			"Params": params,
		}

		rendered, err := engine.Render(test.template, vals)
		if err != nil {
			t.Errorf("error rendering template: %s", err)
		}

		if rendered != test.expected {
			t.Errorf("template mismatch, expected: %+v, got: %+v", test.expected, rendered)
		}
	}
}

func TestUnsafeFuncs(t *testing.T) {
	engine := New()

	unsafeFuncs := []string{"env", "expandenv", "base", "dir", "clean", "ext", "isAbs"}

	for _, fun := range unsafeFuncs {
		_, err := engine.Render(fmt.Sprintf("{{ \"foo\" | %s }}", fun), nil)

		if err == nil {
			t.Errorf("expected error for %s, got none", fun)
		}
	}

}
