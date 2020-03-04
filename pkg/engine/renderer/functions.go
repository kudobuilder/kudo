package renderer

import "gopkg.in/yaml.v2"

// ToYaml takes any value, and returns its YAML representation as a string.
func ToYaml(v interface{}) (string, error) {
	out, err := yaml.Marshal(v)
	return string(out), err
}
