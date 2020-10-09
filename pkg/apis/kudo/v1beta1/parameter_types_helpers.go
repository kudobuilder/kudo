package v1beta1

import (
	"fmt"
	"reflect"
	"strconv"

	"sigs.k8s.io/yaml"
)

func (p *Parameter) IsImmutable() bool {
	return p.Immutable != nil && *p.Immutable
}

func (p *Parameter) IsRequired() bool {
	return p.Required != nil && *p.Required
}

func (p *Parameter) IsEnum() bool {
	return p.Enum != nil
}

func (p *Parameter) HasDefault() bool {
	return p.Default != nil
}

func (p *Parameter) EnumValues() []string {
	if p.IsEnum() {
		return *p.Enum
	}
	return []string{}
}

func (p *Parameter) ValidateDefault() error {
	if err := ValidateParameterValueForType(p.Type, p.Default); err != nil {
		return fmt.Errorf("parameter %q has an invalid default value: %v", p.Name, err)
	}
	if p.IsEnum() {
		if err := ValidateParameterValueForEnum(p.EnumValues(), p.Default); err != nil {
			return fmt.Errorf("parameter %q has an invalid default value: %v", p.Name, err)
		}
	}
	return nil
}

// ValidateValue ensures that a the given value is valid for this parameter
func (p *Parameter) ValidateValue(pValue string) error {
	if p.IsRequired() && !p.HasDefault() && pValue == "" {
		return fmt.Errorf("parameter %q is required but has no value set", p.Name)
	}

	if pValue == "" {
		return nil
	}

	if err := ValidateParameterValueForType(p.Type, pValue); err != nil {
		return fmt.Errorf("parameter %q has an invalid value %q: %v", p.Name, pValue, err)
	}
	if p.IsEnum() {
		if err := ValidateParameterValueForEnum(p.EnumValues(), pValue); err != nil {
			return fmt.Errorf("parameter %q has an invalid value %q: %v", p.Name, pValue, err)
		}
	}
	return nil
}

func ValidateParameterValueForType(pType ParameterType, pValue interface{}) error {
	switch pType {
	case StringValueType:
		_, ok := pValue.(string)
		if !ok {
			return fmt.Errorf("type is %q but format is invalid: %s", pType, pValue)
		}
	case IntegerValueType:
		return validateIntegerType(pValue)
	case NumberValueType:
		return validateNumberType(pValue)
	case BooleanValueType:
		return validateBooleanType(pValue)
	case ArrayValueType:
		return validateArrayType(pValue)
	case MapValueType:
		return validateMapType(pValue)
	}
	return nil
}

func validateIntegerType(pValue interface{}) error {
	switch v := pValue.(type) {
	case int, int8, int16, int32, int64:
		return nil
	case uint, uint8, uint16, uint32, uint64:
		return nil
	case string:
		_, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("type is %q but format of %q is invalid: %v", IntegerValueType, pValue, err)
		}
	default:
		return fmt.Errorf("type is %q but format of %s is invalid", IntegerValueType, pValue)
	}
	return nil
}

func validateNumberType(pValue interface{}) error {
	switch v := pValue.(type) {
	case int, int8, int16, int32, int64:
		return nil
	case uint, uint8, uint16, uint32, uint64:
		return nil
	case float32, float64:
		return nil
	case string:
		_, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("type is %q but format of %q is invalid: %v", NumberValueType, pValue, err)
		}
	default:
		return fmt.Errorf("type is %q but format of %s is invalid", NumberValueType, v)
	}
	return nil
}

func validateBooleanType(pValue interface{}) error {
	switch v := pValue.(type) {
	case bool:
		return nil
	case string:
		_, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("type is %q but format of %q is invalid: %v", BooleanValueType, pValue, err)
		}
	default:
		return fmt.Errorf("type is %q but format of %s is invalid", BooleanValueType, pValue)
	}
	return nil
}

func validateArrayType(pValue interface{}) error {
	t := reflect.TypeOf(pValue)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return nil
	case reflect.String:
		str, _ := pValue.(string) // We know here this is a string

		var result []interface{}
		if err := yaml.Unmarshal([]byte(str), &result); err != nil {
			return fmt.Errorf("type is %q, but format of %s is invalid", ArrayValueType, pValue)
		}

		return nil
	default:
		return fmt.Errorf("type is %q but value %s is not an array", ArrayValueType, pValue)
	}
}

func validateMapType(pValue interface{}) error {
	t := reflect.TypeOf(pValue)
	switch t.Kind() {
	case reflect.Map, reflect.Struct:
		return nil
	case reflect.String:
		str, _ := pValue.(string) // We know here this is a string

		var result map[string]interface{}
		if err := yaml.Unmarshal([]byte(str), &result); err != nil {
			return fmt.Errorf("type is %q, but format of %s is invalid", MapValueType, pValue)
		}

		return nil
	default:
		return fmt.Errorf("type is %q but value %s is not a map", MapValueType, pValue)
	}
}

func ValidateParameterValueForEnum(enumValues []string, pValue interface{}) error {
	for _, eValue := range enumValues {
		if pValue == eValue {
			return nil
		}
	}
	return fmt.Errorf("value is %q, but only allowed values are %v", pValue, enumValues)
}

// GetChangedParamDefs returns a list of parameters from ov2 that changed based on the given compare function between ov1 and ov2
func GetChangedParamDefs(p1, p2 []Parameter, isEqual func(p1, p2 Parameter) bool) []Parameter {
	changedParams := []Parameter{}

	for _, p1 := range p1 {
		for _, p2 := range p2 {
			if p1.Name == p2.Name {
				if !isEqual(p1, p2) {
					changedParams = append(changedParams, p2)
				}
			}
		}
	}

	return changedParams
}

// GetAddedParameters returns a list of parameters that are in oldOv but not in newOv
func GetRemovedParamDefs(old, new []Parameter) []Parameter {
	return GetAddedParameters(new, old)
}

// GetAddedParameters returns a list of parameters that are in newOv but not in oldOv
func GetAddedParameters(old, new []Parameter) []Parameter {
	addedParams := []Parameter{}

NewParams:
	for _, newParam := range new {
		for _, oldParam := range old {
			if newParam.Name == oldParam.Name {
				continue NewParams
			}
		}
		addedParams = append(addedParams, newParam)
	}
	return addedParams
}

// ParameterDiff returns map containing all parameters that were removed or changed between old and new
func ParameterDiff(old, new map[string]string) map[string]string {
	changed, removed := RichParameterDiff(old, new)

	// Join both maps
	for key, val := range removed {
		changed[key] = val
	}
	return changed
}

// RichParameterDiff compares new and old map and returns two maps: first containing all changed/added
// and second all removed parameters.
func RichParameterDiff(old, new map[string]string) (changed, removed map[string]string) {
	changed, removed = make(map[string]string), make(map[string]string)

	for key, val := range old {
		// If a parameter was removed in the new spec
		if _, ok := new[key]; !ok {
			removed[key] = val
		}
	}

	for key, val := range new {
		// If new spec parameter was added or changed
		if v, ok := old[key]; !ok || v != val {
			changed[key] = val
		}
	}
	return
}
