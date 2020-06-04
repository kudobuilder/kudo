package v1beta1

import (
	"fmt"
)

func OperatorInstanceName(operatorName string) string {
	return fmt.Sprintf("%s-instance", operatorName)
}

func OperatorVersionName(operatorName, version string) string {
	return fmt.Sprintf("%s-%s", operatorName, version)
}
