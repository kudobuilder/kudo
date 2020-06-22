package convert

import (
	"errors"
	"fmt"
	"log"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func FilesToResources(files *packages.Files) (*packages.Resources, error) {
	if files.Operator == nil {
		return nil, errors.New("operator.yaml file is missing")
	}
	if files.Params == nil {
		return nil, errors.New("params.yaml file is missing")
	}
	var errs []string
	for _, tt := range files.Operator.Tasks {
		errs = append(errs, validateTask(tt, files.Templates)...)
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}

	operator := &v1beta1.Operator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operator",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: files.Operator.Name,
		},
		Spec: v1beta1.OperatorSpec{
			Description:       files.Operator.Description,
			KudoVersion:       files.Operator.KUDOVersion,
			KubernetesVersion: files.Operator.KubernetesVersion,
			Maintainers:       files.Operator.Maintainers,
			URL:               files.Operator.URL,
			NamespaceManifest: files.Operator.NamespaceManifest,
		},
		Status: v1beta1.OperatorStatus{},
	}

	parameters, err := ParametersToCRDType(files.Params.Parameters)
	if err != nil {
		return nil, err
	}

	fv := &v1beta1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OperatorVersion",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: v1beta1.OperatorVersionName(files.Operator.Name, files.Operator.OperatorVersion),
		},
		Spec: v1beta1.OperatorVersionSpec{
			Operator: corev1.ObjectReference{
				Name: files.Operator.Name,
				Kind: "Operator",
			},
			AppVersion:     files.Operator.AppVersion,
			Version:        files.Operator.OperatorVersion,
			Templates:      files.Templates,
			Tasks:          files.Operator.Tasks,
			Parameters:     parameters,
			Plans:          files.Operator.Plans,
			UpgradableFrom: nil,
		},
		Status: v1beta1.OperatorVersionStatus{},
	}

	instance := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   v1beta1.OperatorInstanceName(files.Operator.Name),
			Labels: map[string]string{kudo.OperatorLabel: files.Operator.Name},
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: corev1.ObjectReference{
				Name: v1beta1.OperatorVersionName(files.Operator.Name, files.Operator.OperatorVersion),
			},
		},
		Status: v1beta1.InstanceStatus{},
	}

	return &packages.Resources{
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
	case task.KudoOperatorTaskKind:
		if len(t.Spec.ParameterFile) != 0 {
			resources = append(resources, t.Spec.ParameterFile)
		}
	case task.DummyTaskKind:
		log.Printf("no validation for task kind %s implemented", t.Kind)
	default:
		errs = append(errs, fmt.Sprintf("unknown task kind %s", t.Kind))
	}

	for _, res := range resources {
		if _, ok := templates[res]; !ok {
			errs = append(errs, fmt.Sprintf("task %s missing template: %s", t.Name, res))
		}
	}

	return errs
}
