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
	if err := install(client.ApiextensionsV1beta1(), operatorCrd()); err != nil {
		return err
	}
	if err := install(client.ApiextensionsV1beta1(), operatorVersionCrd()); err != nil {
		return err
	}
	if err := install(client.ApiextensionsV1beta1(), InstanceCrd()); err != nil {
		return err
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

// InstanceCrd provides definition of the instance CRD
func InstanceCrd() *apiextv1beta1.CustomResourceDefinition {
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
