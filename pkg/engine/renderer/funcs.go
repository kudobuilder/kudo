package renderer

import (
	"strings"

	"sigs.k8s.io/yaml"
)

// toYAML takes an interface, marshals it to yaml, and returns a string.
// It is designed to be called from a template.
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// any error is swallowed resulting in a zero length string
		return ""
	}
	return strings.TrimSpace(string(data))
}
