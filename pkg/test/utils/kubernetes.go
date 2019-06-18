package utils

// Contains methods helpful for interacting with and manipulating Kubernetes resources from YAML.

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pmezard/go-difflib/difflib"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	coretesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceID returns a human readable identifier indicating the object kind, name, and namespace.
func ResourceID(obj runtime.Object) string {
	m, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	gvk := obj.GetObjectKind().GroupVersionKind()

	return fmt.Sprintf("%s:%s/%s", gvk.Kind, m.GetNamespace(), m.GetName())
}

// Namespaced sets the namespace on an object to namespace, if it is a namespace scoped resource.
// If the resource is cluster scoped, then it is ignored and the namespace is not set.
func Namespaced(dClient discovery.DiscoveryInterface, obj runtime.Object, namespace string) (string, string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return "", "", err
	}

	gvk := obj.GetObjectKind().GroupVersionKind()

	resourceTypes, err := dClient.ServerResources()
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

			if !resource.Namespaced {
				return m.GetName(), "", nil
			}

			m.SetNamespace(namespace)
			return m.GetName(), namespace, nil
		}
	}

	return "", "", fmt.Errorf("resource type not found")
}

// PrettyDiff creates a unified diff highlighting the differences between two Kubernetes resources
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
		A:        difflib.SplitLines(expectedBuf.String()),
		B:        difflib.SplitLines(actualBuf.String()),
		FromFile: ResourceID(expected),
		ToFile:   ResourceID(actual),
		Context:  3,
	}

	return difflib.GetUnifiedDiffString(diffed)
}

// ConvertUnstructured converts an unstructured object to the known struct. If the type is not known, then
// the unstructured object is returned unmodified.
func ConvertUnstructured(in runtime.Object) (runtime.Object, error) {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
	if err != nil {
		return nil, err
	}

	var converted runtime.Object

	switch in.GetObjectKind().GroupVersionKind().Kind {
	case "TestStep":
		converted = &kudo.TestStep{}
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

// PatchObject updates expected with the Resource Version from actual.
// In the future, PatchObject may perform a strategic merge of actual into expected.
func PatchObject(actual, expected runtime.Object) error {
	actualMeta, err := meta.Accessor(actual)
	if err != nil {
		return err
	}

	expectedMeta, err := meta.Accessor(expected)
	if err != nil {
		return err
	}

	expectedMeta.SetResourceVersion(actualMeta.GetResourceVersion())
	return nil
}

// MarshalObject marshals a Kubernetes object to a YAML string.
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

// LoadYAML loads all objects from a YAML file.
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
				break
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

// InstallManifests recurses over ManifestsDir to install all resources defined in YAML manifests.
func InstallManifests(ctx context.Context, client client.Client, dClient discovery.DiscoveryInterface, manifestsDir string) error {
	if manifestsDir == "" {
		return nil
	}

	return filepath.Walk(manifestsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		extensions := map[string]bool{
			".yaml": true,
			".yml":  true,
			".json": true,
		}
		if !extensions[filepath.Ext(path)] {
			return nil
		}

		objs, err := LoadYAML(path)
		if err != nil {
			return err
		}

		for _, obj := range objs {
			objectKey := ObjectKey(obj)
			if objectKey.Namespace == "" {
				if _, _, err := Namespaced(dClient, obj, "default"); err != nil {
					return err
				}
			}

			if err := client.Create(ctx, obj); err != nil {
				return err
			}
		}

		return nil
	})
}

// ObjectKey returns an instantiated ObjectKey for the provided object.
func ObjectKey(obj runtime.Object) client.ObjectKey {
	m, _ := meta.Accessor(obj)
	return client.ObjectKey{
		Name:      m.GetName(),
		Namespace: m.GetNamespace(),
	}
}

// NewResource generates a Kubernetes object using the provided apiVersion, kind, name, and namespace.
func NewResource(apiVersion, kind, name, namespace string) runtime.Object {
	meta := map[string]interface{}{
		"name": name,
	}

	if namespace != "" {
		meta["namespace"] = namespace
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata":   meta,
		},
	}
}

// NewPod creates a new pod object.
func NewPod(name, namespace string) runtime.Object {
	return NewResource("v1", "Pod", name, namespace)
}

// WithNamespace naively applies the namespace to the object. Used mainly in tests, otherwise
// use Namespaced.
func WithNamespace(obj runtime.Object, namespace string) runtime.Object {
	obj = obj.DeepCopyObject()

	m, _ := meta.Accessor(obj)
	m.SetNamespace(namespace)

	return obj
}

// WithSpec applies the provided spec to the Kubernetes object.
func WithSpec(obj runtime.Object, spec map[string]interface{}) runtime.Object {
	return WithKeyValue(obj, "spec", spec)
}

// WithStatus applies the provided status to the Kubernetes object.
func WithStatus(obj runtime.Object, status map[string]interface{}) runtime.Object {
	return WithKeyValue(obj, "status", status)
}

// WithKeyValue sets key in the provided object to value.
func WithKeyValue(obj runtime.Object, key string, value map[string]interface{}) runtime.Object {
	obj = obj.DeepCopyObject()

	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return obj
	}

	content[key] = value

	runtime.DefaultUnstructuredConverter.FromUnstructured(content, obj)
	return obj.DeepCopyObject()
}

// FakeDiscoveryClient returns a fake discovery client that is populated with some types for use in
// unit tests.
func FakeDiscoveryClient() discovery.DiscoveryInterface {
	return &fakediscovery.FakeDiscovery{
		Fake: &coretesting.Fake{
			Resources: []*metav1.APIResourceList{
				{
					GroupVersion: corev1.SchemeGroupVersion.String(),
					APIResources: []metav1.APIResource{
						{Name: "pods", Namespaced: true, Kind: "Pod"},
						{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
					},
				},
			},
		},
	}
}

// CreateOrUpdate will create obj if it does not exist and update if it it does.
func CreateOrUpdate(ctx context.Context, client client.Client, obj runtime.Object, retryOnConflict bool) error {
	orig := obj.DeepCopyObject()
	actual := obj.DeepCopyObject()

	err := client.Get(ctx, ObjectKey(actual), actual)
	if err == nil {
		if err = PatchObject(actual, obj); err != nil {
			return err
		}
		err = client.Update(ctx, obj)
		if err != nil && k8serrors.IsConflict(err) && retryOnConflict {
			return CreateOrUpdate(ctx, client, orig, retryOnConflict)
		}
	} else if err != nil && k8serrors.IsNotFound(err) {
		err = client.Create(ctx, obj)
	}

	return err
}

// SetAnnotation sets the given key and value in the object's annotations, returning a copy.
func SetAnnotation(obj runtime.Object, key, value string) runtime.Object {
	obj = obj.DeepCopyObject()

	meta, _ := meta.Accessor(obj)

	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
	meta.SetAnnotations(annotations)

	return obj
}
