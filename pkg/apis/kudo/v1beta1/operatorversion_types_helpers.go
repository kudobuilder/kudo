package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func OperatorInstanceName(operatorName string) string {
	return fmt.Sprintf("%s-instance", operatorName)
}

func OperatorVersionName(operatorName, appVersion, opVersion string) string {
	if appVersion == "" {
		return fmt.Sprintf("%s-%s", operatorName, opVersion)
	}
	return fmt.Sprintf("%s-%s-%s", operatorName, appVersion, opVersion)
}

func (ov *OperatorVersion) FullyQualifiedName() string {
	return OperatorVersionName(ov.Spec.Operator.Name, ov.Spec.AppVersion, ov.Spec.Version)
}

func (ov *OperatorVersion) EqualOperatorVersion(other *OperatorVersion) bool {
	return ov.FullyQualifiedName() == other.FullyQualifiedName()
}

func ListOperatorVersions(ns string, c client.Reader) (l *OperatorVersionList, err error) {
	l = &OperatorVersionList{}
	if err := c.List(context.TODO(), l, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	return l, nil
}

func GetOperatorVersionByName(name, ns string, c client.Reader) (ov *OperatorVersion, err error) {
	ov = &OperatorVersion{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, ov)
	if err != nil {
		return nil, err
	}
	return ov, nil
}

// Sortable Operator implements functionality to correctly sort OVs by name, appVersion and operatorVersion
var _ kudo.SortableOperator = &OperatorVersion{}

func (ov *OperatorVersion) OperatorName() string {
	return ov.Spec.Operator.Name
}

func (ov *OperatorVersion) OperatorVersion() string {
	return ov.Spec.Version
}

func (ov *OperatorVersion) AppVersion() string {
	return ov.Spec.AppVersion
}

func ToSortableOperatorList(ovList []OperatorVersion) kudo.SortableOperatorList {
	l := kudo.SortableOperatorList{}
	for _, ov := range ovList {
		ov := ov
		l = append(l, &ov)
	}
	return l
}
