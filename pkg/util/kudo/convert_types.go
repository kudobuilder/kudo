package kudo

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

// YAMLList converts YAML input describing a list.
func YAMLList(v string) ([]interface{}, error) {
	var result []interface{}

	if err := yaml.Unmarshal([]byte(v), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// YAMLDict converts YAML input describing a dictionary.
func YAMLDict(v string) (interface{}, error) {
	var result interface{}

	if err := yaml.Unmarshal([]byte(v), &result); err != nil {
		return nil, err
	}

	return result, nil
}
