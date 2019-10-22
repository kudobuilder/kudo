package init

import (
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

//Defines the CRDs that the KUDO manager implements and watches.

// Install uses Kubernetes client to install KUDO Crds.
func installCrds(client apiextensionsclient.Interface) error {
	if err := installOperator(client.ApiextensionsV1beta1()); err != nil {
		return err
	}
	if err := installOperatorVersion(client.ApiextensionsV1beta1()); err != nil {
		return err
	}
	if err := installInstance(client.ApiextensionsV1beta1()); err != nil {
		return err
	}
	return nil
}

func installOperator(client v1beta1.CustomResourceDefinitionsGetter) error {
	o := generateOperator()
	_, err := client.CustomResourceDefinitions().Create(o)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("crd %v already exists", o.Name)
		return nil
	}
	return err

}

func installOperatorVersion(client v1beta1.CustomResourceDefinitionsGetter) error {
	ov := generateOperatorVersion()
	_, err := client.CustomResourceDefinitions().Create(ov)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("crd %v already exists", ov.Name)
		return nil
	}
	return err
}

func installInstance(client v1beta1.CustomResourceDefinitionsGetter) error {
	instance := generateInstance()
	_, err := client.CustomResourceDefinitions().Create(instance)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("crd %v already exists", instance.Name)
		return nil
	}
	return err
}

// operatorCrd provides the Operator CRD manifest for printing
func operatorCrd() *apiextv1beta1.CustomResourceDefinition {
	crd := generateOperator()
	crd.TypeMeta = metav1.TypeMeta{
		Kind:       "CustomResourceDefinition",
		APIVersion: "apiextensions.k8s.io/v1beta1",
	}
	return crd
}

func generateOperator() *apiextv1beta1.CustomResourceDefinition {

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
		"meta":       {Type: "object"},
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

// operatorVersionCrd provides the OperatorVersion CRD manifest for printing
func operatorVersionCrd() *apiextv1beta1.CustomResourceDefinition {
	crd := generateOperatorVersion()
	crd.TypeMeta = metav1.TypeMeta{
		Kind:       "CustomResourceDefinition",
		APIVersion: "apiextensions.k8s.io/v1beta1",
	}
	return crd
}

func generateOperatorVersion() *apiextv1beta1.CustomResourceDefinition {
	crd := generateCrd("OperatorVersion", "operatorversions")
	dependProps := map[string]apiextv1beta1.JSONSchemaProps{
		"referenceName": {Type: "string", Description: "Name specifies the name of the dependency.  Referenced via this in defaults.config"},
		"crdVersion":    {Type: "string", Description: "Version captures the requirements for what versions of the above object are allowed Example: ^3.1.4"},
	}
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
		"dependencies": {
			Type: "array",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Required:   []string{"referenceName", "crdVersion"},
				Properties: dependProps,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"operator": {Type: "object"},
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
		"meta":       {Type: "object"},
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

// InstanceCrd provides the Instance CRD manifest for printing
func InstanceCrd() *apiextv1beta1.CustomResourceDefinition {
	crd := generateInstance()
	crd.TypeMeta = metav1.TypeMeta{
		Kind:       "CustomResourceDefinition",
		APIVersion: "apiextensions.k8s.io/v1beta1",
	}
	return crd
}

func generateInstance() *apiextv1beta1.CustomResourceDefinition {
	crd := generateCrd("Instance", "instances")
	dependProps := map[string]apiextv1beta1.JSONSchemaProps{
		"referenceName": {Type: "string", Description: "Name specifies the name of the dependency.  Referenced via this in defaults.config"},
		"crdVersion":    {Type: "string", Description: "Version captures the requirements for what versions of the above object are allowed Example: ^3.1.4"},
	}
	specProps := map[string]apiextv1beta1.JSONSchemaProps{
		"dependencies": {
			Type:        "array",
			Description: "Dependency references specific",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Required:   []string{"referenceName", "crdVersion"},
				Properties: dependProps,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
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
		"meta":       {Type: "object"},
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
	}
	// below is needed if we support 1.15 CRD in v1beta1, it is deprecated within the 1.15
	// for 1.16 it is removed and functions as if preserve == false
	// preserveFields := false
	// crd.Spec.PreserveUnknownFields = &preserveFields
	return crd
}

// CRDManifests provides a slice of strings for each CRD manifest
func CRDManifests() ([]string, error) {
	objs := CRDs()
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

// CRDs returns the slice of crd objects for KUDO
func CRDs() []runtime.Object {
	o := operatorCrd()
	ov := operatorVersionCrd()
	i := InstanceCrd()

	return []runtime.Object{o, ov, i}
}
