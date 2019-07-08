package bundle

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/bundle"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	operatorFileName      = "operator.yaml"
	templateFileNameRegex = "templates/.*.yaml"
	paramsFileName        = "params.yaml"
)

const apiVersion = "kudo.k8s.io/v1alpha1"

// PackageCRDs is collection of CRDs that are used when installing operator
// during installation, package format is converted to this structure
type PackageCRDs struct {
	Operator        *v1alpha1.Operator
	OperatorVersion *v1alpha1.OperatorVersion
	Instance        *v1alpha1.Instance
}

// PackageFiles represents the raw operator package format the way it is found in the tgz package bundles
type PackageFiles struct {
	Templates map[string]string
	Operator  *bundle.Operator
	Params    []v1alpha1.Parameter
}

func parsePackageFile(filePath string, fileBytes []byte, currentPackage *PackageFiles) error {
	isOperatorFile := func(name string) bool {
		return strings.HasSuffix(name, operatorFileName)
	}

	isTemplateFile := func(name string) bool {
		matched, _ := regexp.Match(templateFileNameRegex, []byte(name))
		return matched
	}

	isParametersFile := func(name string) bool {
		return strings.HasSuffix(name, paramsFileName)
	}

	switch {
	case isOperatorFile(filePath):
		if err := yaml.Unmarshal(fileBytes, &currentPackage.Operator); err != nil {
			return errors.Wrap(err, "failed to unmarshal operator file")
		}
	case isTemplateFile(filePath):
		pathParts := strings.Split(filePath, "templates/")
		name := pathParts[len(pathParts)-1]
		currentPackage.Templates[name] = string(fileBytes)
	case isParametersFile(filePath):
		var params map[string]map[string]string
		if err := yaml.Unmarshal(fileBytes, &params); err != nil {
			return errors.Wrapf(err, "failed to unmarshal parameters file: %s", filePath)
		}
		paramsStruct := make([]v1alpha1.Parameter, 0)
		for paramName, param := range params {
			r := v1alpha1.Parameter{
				Name:        paramName,
				Description: param["description"],
				Default:     param["default"],
				Trigger:     param["trigger"],
				DisplayName: param["displayName"],
			}
			paramsStruct = append(paramsStruct, r)
		}
		currentPackage.Params = paramsStruct
	default:
		return fmt.Errorf("unexpected file when reading package from filesystem: %s", filePath)
	}
	return nil
}

func newPackageFiles() PackageFiles {
	return PackageFiles{
		Templates: make(map[string]string),
	}
}

func (p *PackageFiles) getCRDs() (*PackageCRDs, error) {
	if p.Operator == nil {
		return nil, errors.New("operator.yaml file is missing")
	}
	if p.Params == nil {
		return nil, errors.New("params.yaml file is missing")
	}
	var errs []string
	for k, v := range p.Operator.Tasks {
		for _, res := range v.Resources {
			if _, ok := p.Templates[res]; !ok {
				errs = append(errs, fmt.Sprintf("task %s missing template: %s", k, res))
			}
		}
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}

	operator := &v1alpha1.Operator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operator",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   p.Operator.Name,
			Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1alpha1.OperatorSpec{
			Description:       p.Operator.Description,
			KudoVersion:       p.Operator.KUDOVersion,
			KubernetesVersion: p.Operator.KubernetesVersion,
			Maintainers:       p.Operator.Maintainers,
			URL:               p.Operator.URL,
		},
		Status: v1alpha1.OperatorStatus{},
	}

	fv := &v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OperatorVersion",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.Version),
			Namespace: "default",
			Labels:    map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Operator: v1.ObjectReference{
				Name: p.Operator.Name,
				Kind: "Operator",
			},
			Version:        p.Operator.Version,
			Templates:      p.Templates,
			Tasks:          p.Operator.Tasks,
			Parameters:     p.Params,
			Plans:          p.Operator.Plans,
			Dependencies:   p.Operator.Dependencies,
			UpgradableFrom: nil,
		},
		Status: v1alpha1.OperatorVersionStatus{},
	}

	instance := &v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", p.Operator.Name, rand.String(6)),
			Labels: map[string]string{"controller-tools.k8s.io": "1.0", "operator": "zookeeper"},
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name:      fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.Version),
				Namespace: "default",
			},
		},
		Status: v1alpha1.InstanceStatus{},
	}

	return &PackageCRDs{
		Operator:        operator,
		OperatorVersion: fv,
		Instance:        instance,
	}, nil
}
