package convert

import (
	"fmt"

	"sigs.k8s.io/yaml"

	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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

// ParamValue unwraps a parameter value.
// Depending on the parameter type, the value can represent a string or an object described in YAML.
func ParamValue(v *string, t kudov1beta1.ParameterType) (r interface{}, err error) {
	switch t {
	case kudov1beta1.MapValueType:
		r, err = YAMLMap(StringValue(v))
	case kudov1beta1.ArrayValueType:
		r, err = YAMLArray(StringValue(v))
	case kudov1beta1.StringValueType:
		fallthrough
	default:
		r = StringValue(v)
	}

	return
}

// YAMLValue wraps a parameter value.
func YAMLValue(i interface{}, t kudov1beta1.ParameterType) (*string, error) {
	switch t {
	case kudov1beta1.MapValueType:
		fallthrough
	case kudov1beta1.ArrayValueType:
		bytes, err := yaml.Marshal(i)
		if err != nil {
			return nil, err
		}

		result := string(bytes)
		return &result, nil
	case kudov1beta1.StringValueType:
		fallthrough
	default:
		if i == nil {
			return nil, nil
		}

		result := fmt.Sprintf("%v", i)
		return &result, nil
	}
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
func YAMLMap(v string) (interface{}, error) {
	var result interface{}

	if err := yaml.Unmarshal([]byte(v), &result); err != nil {
		return nil, err
	}

	return result, nil
}
