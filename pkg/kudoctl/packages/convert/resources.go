package convert

import (
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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

	operator := &kudoapi.Operator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operator",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: files.Operator.Name,
		},
		Spec: kudoapi.OperatorSpec{
			Description:       files.Operator.Description,
			KudoVersion:       files.Operator.KUDOVersion,
			KubernetesVersion: files.Operator.KubernetesVersion,
			Maintainers:       files.Operator.Maintainers,
			URL:               files.Operator.URL,
			NamespaceManifest: files.Operator.NamespaceManifest,
		},
		Status: kudoapi.OperatorStatus{},
	}

	parameters, err := ParametersToCRDType(files.Params.Parameters)
	if err != nil {
		return nil, err
	}

	fv := &kudoapi.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OperatorVersion",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kudoapi.OperatorVersionName(files.Operator.Name, files.Operator.AppVersion, files.Operator.OperatorVersion),
		},
		Spec: kudoapi.OperatorVersionSpec{
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
		Status: kudoapi.OperatorVersionStatus{},
	}

	instance := BuildInstanceResource(files.Operator.Name, files.Operator.AppVersion, files.Operator.OperatorVersion)

	return &packages.Resources{
		Operator:        operator,
		OperatorVersion: fv,
		Instance:        instance,
	}, nil
}

func BuildInstanceResource(operatorName, appVersion, operatorVersion string) *kudoapi.Instance {
	return &kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   kudoapi.OperatorInstanceName(operatorName),
			Labels: map[string]string{kudo.OperatorLabel: operatorName},
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: corev1.ObjectReference{
				Name: kudoapi.OperatorVersionName(operatorName, appVersion, operatorVersion),
			},
		},
		Status: kudoapi.InstanceStatus{},
	}
}

func validateTask(t kudoapi.Task, templates map[string]string) []string {
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
		// Nothing to validate for Dummy Task
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
