package init

import (
	"fmt"
	"os"
	"strings"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
)

//Defines the CRDs that the KUDO manager implements and watches.

// Install uses Kubernetes client to install KUDO Crds.
func installCrds(client apiextensionsclient.Interface) error {
	if err := install(client.ApiextensionsV1beta1(), operatorCrd()); err != nil {
		return err
	}
	if err := install(client.ApiextensionsV1beta1(), operatorVersionCrd()); err != nil {
		return err
	}
	if err := install(client.ApiextensionsV1beta1(), instanceCrd()); err != nil {
		return err
	}
	return nil
}

func validateInstallation(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition) error {
	existingCrd, err := client.CustomResourceDefinitions().Get(crd.Name, v1.GetOptions{})
	if err != nil {
		if os.IsTimeout(err) {
			return err
		}
		return fmt.Errorf("failed to retrieve CRD %s: %v", crd.Name, err)
	}
	if existingCrd.Spec.Version != crd.Spec.Version {
		return fmt.Errorf("installed CRD %s has invalid version %s, expected %s", crd.Name, existingCrd.Spec.Version, crd.Spec.Version)
	}
	return nil
}

func install(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition) error {
	_, err := client.CustomResourceDefinitions().Create(crd)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("crd %v already exists", crd.Name)
		return nil
	}
	return err

}

// operatorCrd provides definition of the operator CRD
func operatorCrd() *apiextv1beta1.CustomResourceDefinition {

	maintainers := map[string]apiextv1beta1.JSONSchemaProps{
		"name":  {Type: "string"},
		"email": {Type: "string"},
	}

	crd := generateCrd("Operator", "operators")
	specProps := map[string]apiextv1beta1.JSONSchemaProps{
		"description":       {Type: "string"},
		"kubernetesVersion": {Type: "string"},
		"kudoVersion":       {Type: "string"},
		"maintainers": {Type: "array",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Properties: maintainers,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"url": {Type: "string"},
	}

	validationProps := map[string]apiextv1beta1.JSONSchemaProps{
		"apiVersion": {Type: "string"},
		"kind":       {Type: "string"},
		"metadata":   {Type: "object"},
		"spec":       {Properties: specProps, Type: "object"},
		"status":     {Type: "object"},
	}
	crd.Spec.Validation = &apiextv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{Type: "object",
			Properties: validationProps,
		},
	}
	return crd
}

// operatorVersionCrd provides definition of the operatorversion crd
func operatorVersionCrd() *apiextv1beta1.CustomResourceDefinition {
	crd := generateCrd("OperatorVersion", "operatorversions")
	paramProps := map[string]apiextv1beta1.JSONSchemaProps{
		"default":     {Type: "string", Description: "Default is a default value if no parameter is provided by the instance"},
		"description": {Type: "string", Description: "Description captures a longer description of how the variable will be used"},
		"displayName": {Type: "string", Description: "Human friendly crdVersion of the parameter name"},
		"name":        {Type: "string", Description: "Name is the string that should be used in the template file for example, if `name: COUNT` then using the variable `.Params.COUNT`"},
		"required":    {Type: "boolean", Description: "Required specifies if the parameter is required to be provided by all instances, or whether a default can suffice"},
		"trigger":     {Type: "string", Description: "Trigger identifies the plan that gets executed when this parameter changes in the Instance object. Default is `update` if present, or `deploy` if not present"},
	}
	taskProps := map[string]apiextv1beta1.JSONSchemaProps{
		"name": {Type: "string"},
		"kind": {Type: "string"},
		"spec": {Type: "object"},
	}
	specProps := map[string]apiextv1beta1.JSONSchemaProps{
		"connectionString": {Type: "string", Description: "ConnectionString defines a mustached string that can be used to connect to an instance of the Operator"},
		"operator":         {Type: "object"},
		"parameters": {
			Type: "array",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Properties: paramProps,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"plans": {Type: "object", Description: "Plans specify a map a plans that specify how to"},
		"tasks": {
			Type:        "array",
			Description: "List of all tasks available in this OperatorVersions",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Properties: taskProps,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"templates": {Type: "object", Description: "List of go templates YAML files that define the application operator instance"},
		"upgradableFrom": {
			Type:        "array",
			Description: "UpgradableFrom lists all OperatorVersions that can upgrade to this OperatorVersion",
			Items:       &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{Type: "object"}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"crdVersion": {Type: "string"},
	}

	validationProps := map[string]apiextv1beta1.JSONSchemaProps{
		"apiVersion": {Type: "string"},
		"kind":       {Type: "string"},
		"metadata":   {Type: "object"},
		"spec":       {Properties: specProps, Type: "object"},
		"status":     {Type: "object"},
	}

	crd.Spec.Validation = &apiextv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{Type: "object",
			Properties: validationProps,
		},
	}
	return crd
}

// instanceCrd provides the Instance CRD manifest for printing
func instanceCrd() *apiextv1beta1.CustomResourceDefinition {
	crd := generateCrd("Instance", "instances")
	specProps := map[string]apiextv1beta1.JSONSchemaProps{
		"OperatorVersion": {Type: "object", Description: "Operator specifies a reference to a specific Operator object"},
		"parameters":      {Type: "object"},
	}
	statusProps := map[string]apiextv1beta1.JSONSchemaProps{
		"planStatus":       {Type: "object"},
		"aggregatedStatus": {Type: "object"},
	}

	validationProps := map[string]apiextv1beta1.JSONSchemaProps{
		"apiVersion": {Type: "string"},
		"kind":       {Type: "string"},
		"metadata":   {Type: "object"},
		"spec":       {Properties: specProps, Type: "object"},
		"status": {
			Type:       "object",
			Properties: statusProps,
		},
	}

	crd.Spec.Validation = &apiextv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{Type: "object",
			Properties: validationProps,
		},
	}

	crd.Spec.Subresources = &apiextv1beta1.CustomResourceSubresources{Status: &apiextv1beta1.CustomResourceSubresourceStatus{}}
	return crd
}

// generateCrd provides a generic CRD object to be configured
func generateCrd(kind string, plural string) *apiextv1beta1.CustomResourceDefinition {
	plural = strings.ToLower(plural)
	name := plural + "." + group

	labels := generateLabels(map[string]string{"controller-tools.k8s.io": "1.0"})
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   group,
			Version: crdVersion,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural:     plural,
				Singular:   strings.ToLower(kind),
				ShortNames: nil,
				Kind:       kind,
			},
			Scope: "Namespaced",
		},
		Status: apiextv1beta1.CustomResourceDefinitionStatus{
			Conditions:     []apiextv1beta1.CustomResourceDefinitionCondition{},
			StoredVersions: []string{},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1beta1",
		},
	}
	// below is needed if we support 1.15 CRD in v1beta1, it is deprecated within the 1.15
	// for 1.16 it is removed and functions as if preserve == false
	// preserveFields := false
	// crd.Spec.PreserveUnknownFields = &preserveFields
	return crd
}

// KudoCrds represents custom resource definitions needed to run KUDO
type KudoCrds struct {
	Operator        *apiextv1beta1.CustomResourceDefinition
	OperatorVersion *apiextv1beta1.CustomResourceDefinition
	Instance        *apiextv1beta1.CustomResourceDefinition
}

// AsArray returns all CRDs as array of runtime objects
func (c KudoCrds) AsArray() []runtime.Object {
	return []runtime.Object{c.Operator, c.OperatorVersion, c.Instance}
}

// AsYaml returns crds as slice of strings
func (c KudoCrds) AsYaml() ([]string, error) {
	objs := c.AsArray()
	manifests := make([]string, len(objs))
	for i, obj := range objs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return []string{}, err
		}
		manifests[i] = string(o)
	}

	return manifests, nil
}

func (c KudoCrds) ValidateInstallation(client *kube.Client) error {
	if err := validateInstallation(client.ExtClient.ApiextensionsV1beta1(), c.Operator); err != nil {
		return err
	}
	if err := validateInstallation(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion); err != nil {
		return err
	}
	if err := validateInstallation(client.ExtClient.ApiextensionsV1beta1(), c.Instance); err != nil {
		return err
	}
	return nil
}

// CRDs returns the runtime.Object representation of all the CRDs KUDO requires
func CRDs() KudoCrds {
	return KudoCrds{
		Operator:        operatorCrd(),
		OperatorVersion: operatorVersionCrd(),
		Instance:        instanceCrd(),
	}
}
