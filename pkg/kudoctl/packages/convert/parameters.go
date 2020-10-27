package convert

import (
	"fmt"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	utilconvert "github.com/kudobuilder/kudo/pkg/util/convert"
)

func ParameterGroupsToPackageType(groups []kudoapi.ParameterGroup) packages.Groups {
	if len(groups) == 0 {
		return nil
	}

	result := make([]packages.Group, 0, len(groups))

	for _, group := range groups {
		result = append(result, packages.Group{
			Name:        group.Name,
			DisplayName: group.DisplayName,
			Description: group.Description,
			Priority:    group.Priority,
		})
	}

	return result
}

func ParameterGroupsToCRDType(groups packages.Groups) []kudoapi.ParameterGroup {
	if len(groups) == 0 {
		return nil
	}
	result := make([]kudoapi.ParameterGroup, 0, len(groups))

	for _, group := range groups {
		result = append(result, kudoapi.ParameterGroup{
			Name:        group.Name,
			DisplayName: group.DisplayName,
			Description: group.Description,
			Priority:    group.Priority,
		})
	}

	return result
}

func ParametersToPackageType(parameters []kudoapi.Parameter) (packages.Parameters, error) {
	result := make([]packages.Parameter, 0, len(parameters))

	for _, parameter := range parameters {
		var defaultVal interface{} = nil
		if parameter.HasDefault() {
			var err error
			defaultVal, err = utilconvert.UnwrapParamValue(parameter.Default, parameter.Type)
			if err != nil {
				return nil, fmt.Errorf("failed to convert %s default for parameter '%s': %w", parameter.Type, parameter.Name, err)
			}
		}
		var enumValues *[]interface{}
		if parameter.IsEnum() {
			var ev []interface{}
			for _, v := range parameter.EnumValues() {
				v := v
				vUnwrapped, err := utilconvert.UnwrapParamValue(&v, parameter.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to convert %s enum value '%s' for parameter '%s': %w", parameter.Type, v, parameter.Name, err)
				}
				ev = append(ev, vUnwrapped)
			}
			enumValues = &ev
		}

		result = append(result, packages.Parameter{
			DisplayName: parameter.DisplayName,
			Name:        parameter.Name,
			Description: parameter.Description,
			Required:    parameter.Required,
			Advanced:    parameter.Advanced,
			Hint:        parameter.Hint,
			Group:       parameter.Group,
			Default:     defaultVal,
			Trigger:     parameter.Trigger,
			Type:        parameter.Type,
			Immutable:   parameter.Immutable,
			Enum:        enumValues,
		})
	}

	return result, nil
}

// ParametersToCRDType converts parameters to an array of 'Parameter' defined in the KUDO API.
func ParametersToCRDType(parameters packages.Parameters) ([]kudoapi.Parameter, error) {
	result := make([]kudoapi.Parameter, 0, len(parameters))

	for _, parameter := range parameters {
		d, err := utilconvert.WrapParamValue(parameter.Default, parameter.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s default for parameter '%s': %w", parameter.Type, parameter.Name, err)
		}

		var enumValues *[]string
		if parameter.IsEnum() {
			var ev []string
			for _, v := range parameter.EnumValues() {
				vWrapped, err := utilconvert.WrapParamValue(v, parameter.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to convert %s enum value '%s' for parameter '%s': %w", parameter.Type, v, parameter.Name, err)
				}
				ev = append(ev, *vWrapped)
			}
			enumValues = &ev
		}

		result = append(result, kudoapi.Parameter{
			DisplayName: parameter.DisplayName,
			Name:        parameter.Name,
			Description: parameter.Description,
			Required:    parameter.Required,
			Advanced:    parameter.Advanced,
			Hint:        parameter.Hint,
			Group:       parameter.Group,
			Default:     d,
			Trigger:     parameter.Trigger,
			Type:        parameter.Type,
			Immutable:   parameter.Immutable,
			Enum:        enumValues,
		})
	}

	return result, nil
}
