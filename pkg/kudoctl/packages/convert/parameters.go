package convert

import (
	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	utilconvert "github.com/kudobuilder/kudo/pkg/util/convert"
)

// ParametersToCRDType converts parameters to an array of 'Parameter' defined in the KUDO API.
func ParametersToCRDType(parameters packages.Parameters) ([]kudov1beta1.Parameter, error) {
	result := make([]kudov1beta1.Parameter, 0, len(parameters))

	for _, parameter := range parameters {
		d, err := utilconvert.WrapParamValue(parameter.Default, parameter.Type)
		if err != nil {
			return nil, err
		}

		result = append(result, kudov1beta1.Parameter{
			DisplayName: parameter.DisplayName,
			Name:        parameter.Name,
			Description: parameter.Description,
			Required:    parameter.Required,
			Default:     d,
			Trigger:     parameter.Trigger,
			Type:        parameter.Type,
		})
	}

	return result, nil
}
