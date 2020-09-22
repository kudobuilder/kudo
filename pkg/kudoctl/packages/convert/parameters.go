package convert

import (
	"fmt"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	utilconvert "github.com/kudobuilder/kudo/pkg/util/convert"
)

// ParametersToCRDType converts parameters to an array of 'Parameter' defined in the KUDO API.
func ParametersToCRDType(parameters packages.Parameters) ([]kudoapi.Parameter, error) {
	result := make([]kudoapi.Parameter, 0, len(parameters))

	for _, parameter := range parameters {
		d, err := utilconvert.WrapParamValue(parameter.Default, parameter.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s default for parameter '%s': %w", parameter.Type, parameter.Name, err)
		}

		result = append(result, kudoapi.Parameter{
			DisplayName: parameter.DisplayName,
			Name:        parameter.Name,
			Description: parameter.Description,
			Required:    parameter.Required,
			Default:     d,
			Trigger:     parameter.Trigger,
			Type:        parameter.Type,
			Immutable:   parameter.Immutable,
		})
	}

	return result, nil
}
