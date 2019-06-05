package engine

import (
	"testing"
)

func TestRenderBasic(t *testing.T) {
	tests := []struct{
		name string
		template string
		params map[string]interface{}
		expected string
	}{
		{name: "empty", template: "", expected: ""},
		{name: "basic template", template: "name: {{ .Params.Name }}", params: map[string]interface{}{"Name": "Some Name"}, expected: "name: Some Name"},
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
			t.Errorf("Error rendering template: %s", err)
		}

		if rendered != test.expected {
			t.Errorf("Template mismatch, expected: %+v, got: %+v", test.expected, rendered)
		}
	}
}