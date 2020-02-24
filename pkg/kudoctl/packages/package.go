package packages

import (
	"errors"
	"fmt"
	"log"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func (p *Files) Resources() (*Resources, error) {
	if p.Operator == nil {
		return nil, errors.New("operator.yaml file is missing")
	}
	if p.Params == nil {
		return nil, errors.New("params.yaml file is missing")
	}
	var errs []string
	for _, tt := range p.Operator.Tasks {
		errs = append(errs, validateTask(tt, p.Templates)...)
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}

	operator := &v1beta1.Operator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operator",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Operator.Name,
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
			Name: fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.OperatorVersion),
		},
		Spec: v1beta1.OperatorVersionSpec{
			Operator: v1.ObjectReference{
				Name: p.Operator.Name,
				Kind: "Operator",
			},
			AppVersion:     p.Operator.AppVersion,
			Version:        p.Operator.OperatorVersion,
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
			Labels: map[string]string{kudo.OperatorLabel: p.Operator.Name},
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.OperatorVersion),
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
	var errs []string
	var resources []string
	switch t.Kind {
	case task.ApplyTaskKind, task.DeleteTaskKind:
		resources = t.Spec.ResourceTaskSpec.Resources
	case task.ToggleTaskKind:
		resources = t.Spec.ResourceTaskSpec.Resources
		if len(t.Spec.Parameter) == 0 {
			errs = append(errs, fmt.Sprintf("toggle task %s does not have parameter specified", t.Name))
		}
	case task.PipeTaskKind:
		resources = append(resources, t.Spec.PipeTaskSpec.Pod)

		if len(t.Spec.PipeTaskSpec.Pipe) == 0 {
			errs = append(errs, fmt.Sprintf("task %s does not have pipe files specified", t.Name))
		}
	case task.DummyTaskKind:
	default:
		log.Printf("no validation for task kind %s implemented", t.Kind)
	}

	for _, res := range resources {
		if _, ok := templates[res]; !ok {
			errs = append(errs, fmt.Sprintf("task %s missing template: %s", t.Name, res))
		}
	}

	return errs
}
