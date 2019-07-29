package utils

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNamespaced(t *testing.T) {
	fake := FakeDiscoveryClient()

	for _, test := range []struct {
		testName    string
		resource    runtime.Object
		namespace   string
		shouldError bool
	}{
		{
			testName:  "namespaced resource",
			resource:  NewPod("hello", ""),
			namespace: "set-the-namespace",
		},
		{
			testName:  "namespace already set",
			resource:  NewPod("hello", "other"),
			namespace: "other",
		},
		{
			testName:  "not-namespaced resource",
			resource:  NewResource("v1", "Namespace", "hello", ""),
			namespace: "",
		},
		{
			testName:    "non-existent resource",
			resource:    NewResource("v1", "Blah", "hello", ""),
			shouldError: true,
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			m, _ := meta.Accessor(test.resource)

			actualName, actualNamespace, err := Namespaced(fake, test.resource, "set-the-namespace")

			if test.shouldError {
				assert.NotNil(t, err)
				assert.Equal(t, "", actualName)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, m.GetName(), actualName)
			}

			assert.Equal(t, test.namespace, actualNamespace)
			assert.Equal(t, test.namespace, m.GetNamespace())
		})
	}
}

func TestGETAPIResource(t *testing.T) {
	fake := FakeDiscoveryClient()

	apiResource, err := GetAPIResource(fake, schema.GroupVersionKind{
		Kind:    "Pod",
		Version: "v1",
	})
	assert.Nil(t, err)
	assert.Equal(t, apiResource.Kind, "Pod")

	apiResource, err = GetAPIResource(fake, schema.GroupVersionKind{
		Kind:    "NonExistentResourceType",
		Version: "v1",
	})
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "resource type not found")
}

func TestRetry(t *testing.T) {
	index := 0

	assert.Nil(t, Retry(context.TODO(), func(context.Context) error {
		index++
		if index == 1 {
			return errors.New("ignore this error")
		}
		return nil
	}, func(err error) bool { return false }, func(err error) bool {
		return err.Error() == "ignore this error"
	}))

	assert.Equal(t, 2, index)
}

func TestRetryWithUnexpectedError(t *testing.T) {
	index := 0

	assert.Equal(t, errors.New("bad error"), Retry(context.TODO(), func(context.Context) error {
		index++
		if index == 1 {
			return errors.New("bad error")
		}
		return nil
	}, func(err error) bool { return false }, func(err error) bool {
		return err.Error() == "ignore this error"
	}))
	assert.Equal(t, 1, index)
}

func TestRetryWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	assert.Equal(t, errors.New("error"), Retry(ctx, func(context.Context) error {
		return errors.New("error")
	}, func(err error) bool { return true }))
}

func TestLoadYAML(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test.yaml")
	assert.Nil(t, err)
	defer tmpfile.Close()

	ioutil.WriteFile(tmpfile.Name(), []byte(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.7.9
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: nginx
  name: hello
spec:
  containers:
  - name: nginx
    image: nginx:1.7.9
`), 0644)

	objs, err := LoadYAML(tmpfile.Name())
	assert.Nil(t, err)

	assert.Equal(t, &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"app": "nginx",
				},
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"image": "nginx:1.7.9",
						"name":  "nginx",
					},
				},
			},
		},
	}, objs[0])

	assert.Equal(t, &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"app": "nginx",
				},
				"name": "hello",
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"image": "nginx:1.7.9",
						"name":  "nginx",
					},
				},
			},
		},
	}, objs[1])
}

func TestMatchesKind(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test.yaml")
	assert.Nil(t, err)
	defer tmpfile.Close()

	ioutil.WriteFile(tmpfile.Name(), []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: hello
spec:
  containers:
  - name: nginx
    image: nginx:1.7.9
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: hello
`), 0644)

	objs, err := LoadYAML(tmpfile.Name())
	assert.Nil(t, err)

	crd := NewResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "", "")
	pod := NewResource("v1", "Pod", "", "")
	svc := NewResource("v1", "Service", "", "")

	assert.False(t, MatchesKind(objs[0], crd))
	assert.True(t, MatchesKind(objs[0], pod))
	assert.True(t, MatchesKind(objs[0], pod, crd))
	assert.True(t, MatchesKind(objs[0], crd, pod))
	assert.False(t, MatchesKind(objs[0], crd, svc))

	assert.True(t, MatchesKind(objs[1], crd))
	assert.False(t, MatchesKind(objs[1], pod))
	assert.True(t, MatchesKind(objs[1], pod, crd))
	assert.True(t, MatchesKind(objs[1], crd, pod))
	assert.False(t, MatchesKind(objs[1], svc, pod))
}

func TestGetKubectlArgs(t *testing.T) {
	for _, test := range []struct {
		testName  string
		namespace string
		args      string
		expected  []string
	}{
		{
			testName:  "namespace long, combined already set at end is not modified",
			namespace: "default",
			args:      "kudo test --namespace=test-canary",
			expected: []string{
				"kubectl", "kudo", "test", "--namespace=test-canary",
			},
		},
		{
			testName:  "namespace long already set at end is not modified",
			namespace: "default",
			args:      "kudo test --namespace test-canary",
			expected: []string{
				"kubectl", "kudo", "test", "--namespace", "test-canary",
			},
		},
		{
			testName:  "namespace short, combined already set at end is not modified",
			namespace: "default",
			args:      "kudo test -n=test-canary",
			expected: []string{
				"kubectl", "kudo", "test", "-n=test-canary",
			},
		},
		{
			testName:  "namespace short already set at end is not modified",
			namespace: "default",
			args:      "kudo test -n test-canary",
			expected: []string{
				"kubectl", "kudo", "test", "-n", "test-canary",
			},
		},
		{
			testName:  "namespace long, combined already set at beginning is not modified",
			namespace: "default",
			args:      "--namespace=test-canary kudo test",
			expected: []string{
				"kubectl", "--namespace=test-canary", "kudo", "test",
			},
		},
		{
			testName:  "namespace long already set at beginning is not modified",
			namespace: "default",
			args:      "--namespace test-canary kudo test",
			expected: []string{
				"kubectl", "--namespace", "test-canary", "kudo", "test",
			},
		},
		{
			testName:  "namespace short, combined already set at beginning is not modified",
			namespace: "default",
			args:      "-n=test-canary kudo test",
			expected: []string{
				"kubectl", "-n=test-canary", "kudo", "test",
			},
		},
		{
			testName:  "namespace short already set at beginning is not modified",
			namespace: "default",
			args:      "-n test-canary kudo test",
			expected: []string{
				"kubectl", "-n", "test-canary", "kudo", "test",
			},
		},
		{
			testName:  "namespace long, combined already set in middle is not modified",
			namespace: "default",
			args:      "kudo --namespace=test-canary test",
			expected: []string{
				"kubectl", "kudo", "--namespace=test-canary", "test",
			},
		},
		{
			testName:  "namespace long already set in middle is not modified",
			namespace: "default",
			args:      "kudo --namespace test-canary test",
			expected: []string{
				"kubectl", "kudo", "--namespace", "test-canary", "test",
			},
		},
		{
			testName:  "namespace short, combined already set in middle is not modified",
			namespace: "default",
			args:      "kudo -n=test-canary test",
			expected: []string{
				"kubectl", "kudo", "-n=test-canary", "test",
			},
		},
		{
			testName:  "namespace short already set in middle is not modified",
			namespace: "default",
			args:      "kudo -n test-canary test",
			expected: []string{
				"kubectl", "kudo", "-n", "test-canary", "test",
			},
		},
		{
			testName:  "namespace not set is appended",
			namespace: "default",
			args:      "kudo test",
			expected: []string{
				"kubectl", "kudo", "test", "--namespace", "default",
			},
		},
		{
			testName:  "unknown arguments do not break parsing with namespace is not set",
			namespace: "default",
			args:      "kudo test --config kudo-test.yaml",
			expected: []string{
				"kubectl", "kudo", "test", "--config", "kudo-test.yaml", "--namespace", "default",
			},
		},
		{
			testName:  "unknown arguments do not break parsing if namespace is set at beginning",
			namespace: "default",
			args:      "--namespace=test-canary kudo test --config kudo-test.yaml",
			expected: []string{
				"kubectl", "--namespace=test-canary", "kudo", "test", "--config", "kudo-test.yaml",
			},
		},
		{
			testName:  "unknown arguments do not break parsing if namespace is set at middle",
			namespace: "default",
			args:      "kudo --namespace=test-canary test --config kudo-test.yaml",
			expected: []string{
				"kubectl", "kudo", "--namespace=test-canary", "test", "--config", "kudo-test.yaml",
			},
		},
		{
			testName:  "unknown arguments do not break parsing if namespace is set at end",
			namespace: "default",
			args:      "kudo test --config kudo-test.yaml --namespace=test-canary",
			expected: []string{
				"kubectl", "kudo", "test", "--config", "kudo-test.yaml", "--namespace=test-canary",
			},
		},
		{
			testName:  "quotes are respected when parsing",
			namespace: "default",
			args:      "kudo \"test quoted\"",
			expected: []string{
				"kubectl", "kudo", "test quoted", "--namespace", "default",
			},
		},
		{
			testName:  "kubectl is not pre-pended if it is already present",
			namespace: "default",
			args:      "kubectl kudo test",
			expected: []string{
				"kubectl", "kudo", "test", "--namespace", "default",
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			args, err := GetKubectlArgs(test.args, test.namespace)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, args)
		})
	}
}
