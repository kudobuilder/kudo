package convert

import (
	"fmt"

	"sigs.k8s.io/yaml"

	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

// StringPtr returns a pointer to the string value passed in.
func StringPtr(input string) *string {
	return &input
}

// StringValue returns the value of the string pointer passed in or
// "" if the pointer is nil.
func StringValue(input *string) string {
	if input != nil {
		return *input
	}
	return ""
}

// UnwrapParamValue unwraps a parameter value to an interface according to its type.
// Depending on the parameter type, the input value can represent a string or an object described in YAML.
func UnwrapParamValue(wrapped *string, parameterType kudov1beta1.ParameterType) (unwrapped interface{}, err error) {
	switch parameterType {
	case kudov1beta1.MapValueType:
		unwrapped, err = ToYAMLMap(StringValue(wrapped))
	case kudov1beta1.ArrayValueType:
		unwrapped, err = ToYAMLArray(StringValue(wrapped))
	case kudov1beta1.StringValueType:
		fallthrough
	default:
		unwrapped = StringValue(wrapped)
	}

	return
}

// WrapParamValue wraps a parameter value to a string according to its type.
// Complex parameter types will be described as YAML, simple parameter types use the string value.
func WrapParamValue(unwrapped interface{}, parameterType kudov1beta1.ParameterType) (*string, error) {
	switch parameterType {
	case kudov1beta1.MapValueType:
		fallthrough
	case kudov1beta1.ArrayValueType:
		bytes, err := yaml.Marshal(unwrapped)
		if err != nil {
			return nil, err
		}

		wrapped := string(bytes)
		return &wrapped, nil
	case kudov1beta1.StringValueType:
		fallthrough
	default:
		if unwrapped == nil {
			return nil, nil
		}

		wrapped := fmt.Sprintf("%v", unwrapped)
		return &wrapped, nil
	}
}

// ToYAMLArray unmarshals stringified YAML into an array.
func ToYAMLArray(input string) ([]interface{}, error) {
	var result []interface{}

	if err := yaml.Unmarshal([]byte(input), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ToYAMLMap unmarshals stringified YAML into an interface.
func ToYAMLMap(input string) (interface{}, error) {
	var result interface{}

	if err := yaml.Unmarshal([]byte(input), &result); err != nil {
		return nil, err
	}

	return result, nil
}
