package utils

// Contains methods helpful for interacting with and manipulating Kubernetes resources from YAML.

import (
	"bufio"
	"bytes"
	"context"
	ejson "encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis"
	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pmezard/go-difflib/difflib"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/scheme"
	coretesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateErrors accepts an error as its first argument and passes it to each function in the errValidationFuncs slice,
// if any of the methods returns true, the method returns nil, otherwise it returns the original error.
func ValidateErrors(err error, errValidationFuncs ...func(error) bool) error {
	for _, errFunc := range errValidationFuncs {
		if errFunc(err) {
			return nil
		}
	}

	return err
}

// Retry retries a method until the context expires or the method returns an unvalidated error.
func Retry(ctx context.Context, fn func(context.Context) error, errValidationFuncs ...func(error) bool) error {
	var lastErr error
	errCh := make(chan error)
	doneCh := make(chan struct{})

	// do { } while (err != nil): https://stackoverflow.com/a/32844744/10892393
	for ok := true; ok; ok = lastErr != nil {
		// run the function in a goroutine and close it once it is finished so that
		// we can use select to wait for both the function return and the context deadline.

		go func() {
			if err := fn(ctx); err != nil {
				errCh <- err
			} else {
				doneCh <- struct{}{}
			}
		}()

		select {
		// the callback finished
		case <-doneCh:
			lastErr = nil
		case err := <-errCh:
			// check if we tolerate the error, return it if not.
			if e := ValidateErrors(err, errValidationFuncs...); e != nil {
				return e
			}
			lastErr = err
		// timeout exceeded
		case <-ctx.Done():
			if lastErr == nil {
				// there's no previous error, so just return the timeout error
				return ctx.Err()
			}

			// return the most recent error
			return lastErr
		}
	}

	return lastErr
}

// Scheme returns an initialized Kubernetes Scheme.
func Scheme() *runtime.Scheme {
	apis.AddToScheme(scheme.Scheme)
	apiextensions.AddToScheme(scheme.Scheme)
	return scheme.Scheme
}

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

	resource, err := GetAPIResource(dClient, obj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return "", "", err
	}

	if !resource.Namespaced {
		return m.GetName(), "", nil
	}

	m.SetNamespace(namespace)
	return m.GetName(), namespace, nil
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
	case "TestSuite":
		converted = &kudo.TestSuite{}
	case "CustomResourceDefinition":
		converted = &apiextensions.CustomResourceDefinition{}
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
func InstallManifests(ctx context.Context, client client.Client, dClient discovery.DiscoveryInterface, manifestsDir string) ([]runtime.Object, error) {
	objects := []runtime.Object{}

	if manifestsDir == "" {
		return objects, nil
	}

	return objects, filepath.Walk(manifestsDir, func(path string, info os.FileInfo, err error) error {
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

			if err := CreateOrUpdate(ctx, client, obj, true); err != nil {
				return err
			}

			objects = append(objects, obj)
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

// WithLabels sets the labels on an object.
func WithLabels(obj runtime.Object, labels map[string]string) runtime.Object {
	obj = obj.DeepCopyObject()

	m, _ := meta.Accessor(obj)
	m.SetLabels(labels)

	return obj
}

// WithAnnotations sets the annotations on an object.
func WithAnnotations(obj runtime.Object, annotations map[string]string) runtime.Object {
	obj = obj.DeepCopyObject()

	m, _ := meta.Accessor(obj)
	m.SetAnnotations(annotations)

	return obj
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
func CreateOrUpdate(ctx context.Context, client client.Client, obj runtime.Object, retryOnError bool) error {
	orig := obj.DeepCopyObject()

	validators := []func(err error) bool{}

	if retryOnError {
		validators = append(validators, k8serrors.IsConflict, func(err error) bool {
			_, ok := err.(*ejson.SyntaxError)
			return ok
		})
	}

	return Retry(ctx, func(ctx context.Context) error {
		expected := orig.DeepCopyObject()
		actual := orig.DeepCopyObject()

		err := client.Get(ctx, ObjectKey(actual), actual)
		if err == nil {
			if err = PatchObject(actual, expected); err != nil {
				return err
			}
			err = client.Update(ctx, expected)
		} else if err != nil && k8serrors.IsNotFound(err) {
			err = client.Create(ctx, obj)
		}
		return err
	}, validators...)
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

// GetAPIResource returns the APIResource object for a specific GroupVersionKind.
func GetAPIResource(dClient discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (metav1.APIResource, error) {
	resourceTypes, err := dClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return metav1.APIResource{}, err
	}

	for _, resource := range resourceTypes.APIResources {
		if !strings.EqualFold(resource.Kind, gvk.Kind) {
			continue
		}

		return resource, nil
	}

	return metav1.APIResource{}, fmt.Errorf("resource type not found")
}

// WaitForCRDs waits for the provided CRD types to be available in the Kubernetes API.
func WaitForCRDs(dClient discovery.DiscoveryInterface, crds []runtime.Object) error {
	waitingFor := []schema.GroupVersionKind{}

	for _, crdObj := range crds {
		crd, ok := crdObj.(*apiextensions.CustomResourceDefinition)
		if !ok {
			continue
		}

		waitingFor = append(waitingFor, schema.GroupVersionKind{
			Group:   crd.Spec.Group,
			Version: crd.Spec.Version,
			Kind:    crd.Spec.Names.Kind,
		})
	}

	return wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (done bool, err error) {
		for _, resource := range waitingFor {
			_, err := GetAPIResource(dClient, resource)
			if err != nil {
				return false, nil
			}
		}

		return true, nil
	})
}
