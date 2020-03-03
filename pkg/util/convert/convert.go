package convert

import (
	"sigs.k8s.io/yaml"
)

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// StringValue returns the value of the string pointer passed in or
// "" if the pointer is nil.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// YAMLArray converts YAML input describing an array.
func YAMLArray(v string) ([]interface{}, error) {
	var result []interface{}

	if err := yaml.Unmarshal([]byte(v), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// YAMLObject converts YAML input describing a mapping type.
func YAMLObject(v string) (interface{}, error) {
	var result interface{}

	if err := yaml.Unmarshal([]byte(v), &result); err != nil {
		return nil, err
	}

	return result, nil
}
