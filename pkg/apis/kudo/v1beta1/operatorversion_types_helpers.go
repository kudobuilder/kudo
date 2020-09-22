package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func OperatorInstanceName(operatorName string) string {
	return fmt.Sprintf("%s-instance", operatorName)
}

func OperatorVersionName(operatorName, version string) string {
	return fmt.Sprintf("%s-%s", operatorName, version)
}

func (ov *OperatorVersion) FullyQualifiedName() string {
	return fmt.Sprintf("%s-%s", ov.Name, ov.Spec.AppVersion)
}

func (ov *OperatorVersion) EqualOperatorVersion(other *OperatorVersion) bool {
	return ov.FullyQualifiedName() == other.FullyQualifiedName()
}

func GetOperatorVersionByName(name, ns string, c client.Reader) (ov *OperatorVersion, err error) {
	ov = &OperatorVersion{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, ov)
	if err != nil {
		return nil, err
	}
	return ov, nil
}
