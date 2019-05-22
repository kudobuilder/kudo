package test

// Contains methods helpful for interacting with and manipulating Kubernetes resources from YAML.

import (
	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"k8s.io/client-go/discovery"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"github.com/pmezard/go-difflib/difflib"
	"bytes"
	"io"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"bufio"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Return a human readable identifier indicating the object kind, name, and namespace.
func ResourceID(obj runtime.Object) string {
	m, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	gvk := obj.GetObjectKind().GroupVersionKind()

	return fmt.Sprintf("%s:%s/%s", gvk.Kind, m.GetNamespace(), m.GetName())
}

// Set the namespace on an object to namespace, if it is a namespace scoped resource.
// If the resource is cluster scoped, then it is ignored and the namespace is not set.
func Namespaced(obj runtime.Object, namespace string) (string, string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return "", "", err
	}

	gvk := obj.GetObjectKind().GroupVersionKind()

	dClient, err := discovery.NewDiscoveryClientForConfig(config.GetConfigOrDie())
	if err != nil {
		return "", "", err
	}

	resourceTypes, err := dClient.ServerPreferredResources()
	if err != nil {
		return "", "", err
	}

	for _, resourceType := range resourceTypes {
		for _, resource := range resourceType.APIResources {
			gv := ""
			if gvk.Group != "" {
				gv = gvk.Group + "/"
			}
			gv += gvk.Version

			if resource.Kind != gvk.Kind || resourceType.GroupVersion != gv {
				continue
			}

			if ! resource.Namespaced {
				return m.GetName(), "", nil
			}

			m.SetNamespace(namespace)
			return m.GetName(), namespace, nil
		}
	}

	return "", "", fmt.Errorf("Resource type not found.")
}

// Create a unified diff highlighting the differences between two Kubernetes resources
func PrettyDiff(expected runtime.Object, actual runtime.Object) (string, error) {
	expectedBuf := &bytes.Buffer{}
	actualBuf := &bytes.Buffer{}

	if err := MarshalObject(expected, expectedBuf); err != nil {
		return "", err
	}

	if err := MarshalObject(actual, actualBuf); err != nil {
		return "", err
	}

	diffed := difflib.UnifiedDiff{
		A: difflib.SplitLines(expectedBuf.String()),
		B: difflib.SplitLines(actualBuf.String()),
		FromFile: ResourceID(expected),
		ToFile: ResourceID(actual),
		Context: 3,
	}

	return difflib.GetUnifiedDiffString(diffed)
}

// Convert an unstructured object to the known struct. If the type is not known, then
// the unstructured object is returned unmodified.
func ConvertUnstructured(in runtime.Object) (runtime.Object, error) {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
	if err != nil {
		return nil, err
	}

	var converted runtime.Object

	switch in.GetObjectKind().GroupVersionKind().Kind {
	case "TestCase":
		converted = &kudo.TestCase{}
	case "TestAssert":
		converted = &kudo.TestAssert{}
	default:
		return in, nil
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct, converted)
	if err != nil {
		return nil, err
	}

	return converted, nil
}

// Marshal a Kubernetes object to a YAML string.
func MarshalObject(o runtime.Object, w io.Writer) error {
	encoder := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

	copied := o.DeepCopyObject()

	meta, err := meta.Accessor(copied)
	if err != nil {
		return err
	}

	meta.SetResourceVersion("")
	meta.SetCreationTimestamp(metav1.Time{})
	meta.SetSelfLink("")
	meta.SetUID(types.UID(""))
	meta.SetGeneration(0)

	annotations := meta.GetAnnotations()
	delete(annotations, "deployment.kubernetes.io/revision")

	if len(annotations) > 0 {
		meta.SetAnnotations(annotations)
	} else {
		meta.SetAnnotations(nil)
	}

	return encoder.Encode(copied, w)
}

// Load all objects from a YAML file.
func LoadYAML(path string) ([]runtime.Object, error) {
	opened, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	yamlReader := yaml.NewYAMLReader(bufio.NewReader(opened))

	objects := []runtime.Object{}

	for {
		data, err := yamlReader.Read()
		if err != nil {
			if err == io.EOF {
				return objects, nil
			}
			return nil, err
		}

		unstructuredObj := &unstructured.Unstructured{}
		decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))

		if err = decoder.Decode(unstructuredObj); err != nil {
			return nil, err
		}

		obj, err := ConvertUnstructured(unstructuredObj)
		if err != nil {
			return nil, err
		}

		objects = append(objects, obj)
	}

	return objects, nil
}
