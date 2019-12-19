//Defines the CRDs that the KUDO manager implements and watches.
package crd

import (
	"fmt"
	"os"
	"strings"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
)

const (
	group      = "kudo.dev"
	crdVersion = "v1beta1"
)

// Ensure kudoinit.InitStep is implemented
var _ kudoinit.InitStep = &Initializer{}

// Initializer represents custom resource definitions needed to run KUDO
type Initializer struct {
	Operator        *apiextv1beta1.CustomResourceDefinition
	OperatorVersion *apiextv1beta1.CustomResourceDefinition
	Instance        *apiextv1beta1.CustomResourceDefinition
}

// CRDs returns the runtime.Object representation of all the CRDs KUDO requires
func NewInitializer() Initializer {
	return Initializer{
		Operator:        operatorCrd(),
		OperatorVersion: operatorVersionCrd(),
		Instance:        instanceCrd(),
	}
}

// AsArray returns all CRDs as array of runtime objects
func (c Initializer) AsArray() []runtime.Object {
	return []runtime.Object{c.Operator, c.OperatorVersion, c.Instance}
}

// AsYamlManifests returns crds as slice of strings
func (c Initializer) AsYamlManifests() ([]string, error) {
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

// Install uses Kubernetes client to install KUDO Crds.
func (c Initializer) Install(client *kube.Client) error {
	if err := c.install(client.ExtClient.ApiextensionsV1beta1(), c.Operator); err != nil {
		return err
	}
	if err := c.install(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion); err != nil {
		return err
	}
	if err := c.install(client.ExtClient.ApiextensionsV1beta1(), c.Instance); err != nil {
		return err
	}
	return nil
}

func (c Initializer) ValidateInstallation(client *kube.Client) error {
	if err := c.validateInstallation(client.ExtClient.ApiextensionsV1beta1(), c.Operator); err != nil {
		return err
	}
	if err := c.validateInstallation(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion); err != nil {
		return err
	}
	if err := c.validateInstallation(client.ExtClient.ApiextensionsV1beta1(), c.Instance); err != nil {
		return err
	}
	return nil
}

func (c Initializer) validateInstallation(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition) error {
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

func (c Initializer) install(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition) error {
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
		"default":     {Type: "string", Description: "Default is a default value if no parameter is provided by the instance."},
		"description": {Type: "string", Description: "Description captures a longer description of how the parameter will be used."},
		"displayName": {Type: "string", Description: "DisplayName can be used by UIs."},
		"name":        {Type: "string", Description: "Name is the string that should be used in the template file for example, if `name: COUNT` then using the variable in a spec like: \n spec:   replicas:  {{ .Params.COUNT }}"},
		"required":    {Type: "boolean", Description: "Required specifies if the parameter is required to be provided by all instances, or whether a default can suffice."},
		"trigger":     {Type: "string", Description: "Trigger identifies the plan that gets executed when this parameter changes in the Instance object. Default is `update` if a plan with that name exists, otherwise it's `deploy`."},
	}
	taskProps := map[string]apiextv1beta1.JSONSchemaProps{
		"name": {Type: "string"},
		"kind": {Type: "string"},
		"spec": {Type: "object"},
	}
	specProps := map[string]apiextv1beta1.JSONSchemaProps{
		"appVersion":       {Type: "string"},
		"connectionString": {Type: "string", Description: "ConnectionString defines a templated string that can be used to connect to an instance of the Operator."},
		"operator":         {Type: "object"},
		"parameters": {
			Type: "array",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Properties: paramProps,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"plans": {Type: "object", Description: "Plans maps a plan name to a plan."},
		"tasks": {
			Type:        "array",
			Description: "List of all tasks available in this OperatorVersion.",
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{
				Type:       "object",
				Properties: taskProps,
			}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"templates": {Type: "object", Description: "Templates is a list of references to YAML templates located in the templates folder and later referenced from tasks."},
		"upgradableFrom": {
			Type:        "array",
			Description: "UpgradableFrom lists all OperatorVersions that can upgrade to this OperatorVersion.",
			Items:       &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{Type: "object"}, JSONSchemas: []apiextv1beta1.JSONSchemaProps{}},
		},
		"version": {Type: "string"},
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
		"operatorVersion": {Type: "object", Description: "OperatorVersion specifies a reference to a specific OperatorVersion object."},
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

	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
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
