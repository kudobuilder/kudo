package packages

import (
	"fmt"
	"log"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/util/kudo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Files) Resources() (*Resources, error) {
	if p.Operator == nil {
		return nil, fmt.Errorf("operator.yaml file is missing")
	}
	if p.Params == nil {
		return nil, fmt.Errorf("params.yaml file is missing")
	}
	var errs []string
	for _, tt := range p.Operator.Tasks {
		errs = append(errs, validateTask(tt, p.Templates)...)
	}

	if len(errs) != 0 {
		return nil, fmt.Errorf(strings.Join(errs, "\n"))
	}

	operator := &v1beta1.Operator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operator",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   p.Operator.Name,
			Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1beta1.OperatorSpec{
			Description:       p.Operator.Description,
			KudoVersion:       p.Operator.KUDOVersion,
			KubernetesVersion: p.Operator.KubernetesVersion,
			Maintainers:       p.Operator.Maintainers,
			URL:               p.Operator.URL,
		},
		Status: v1beta1.OperatorStatus{},
	}

	fv := &v1beta1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OperatorVersion",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.Version),
			Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1beta1.OperatorVersionSpec{
			Operator: v1.ObjectReference{
				Name: p.Operator.Name,
				Kind: "Operator",
			},
			AppVersion:     p.Operator.AppVersion,
			Version:        p.Operator.Version,
			Templates:      p.Templates,
			Tasks:          p.Operator.Tasks,
			Parameters:     p.Params.Parameters,
			Plans:          p.Operator.Plans,
			UpgradableFrom: nil,
		},
		Status: v1beta1.OperatorVersionStatus{},
	}

	instance := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-instance", p.Operator.Name),
			Labels: map[string]string{"controller-tools.k8s.io": "1.0", kudo.OperatorLabel: p.Operator.Name},
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.Version),
			},
		},
		Status: v1beta1.InstanceStatus{},
	}

	return &Resources{
		Operator:        operator,
		OperatorVersion: fv,
		Instance:        instance,
	}, nil
}

func validateTask(t v1beta1.Task, templates map[string]string) []string {
	var resources []string
	switch t.Kind {
	case task.ApplyTaskKind:
		resources = t.Spec.ResourceTaskSpec.Resources
	case task.DeleteTaskKind:
		resources = t.Spec.ResourceTaskSpec.Resources
	case task.DummyTaskKind:
	default:
		log.Printf("no validation for task kind %s implemented", t.Kind)
	}

	var errs []string
	for _, res := range resources {
		if _, ok := templates[res]; !ok {
			errs = append(errs, fmt.Sprintf("task %s missing template: %s", t.Name, res))
		}
	}

	return errs
}
