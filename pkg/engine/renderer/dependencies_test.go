package renderer

import (
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestGetResources(t *testing.T) {
	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "configmap",
			Namespace:   "namespace",
			Annotations: map[string]string{},
		},
		Data: map[string]string{
			"key": "value",
		},
	}
	cmString, _ := runtime.Encode(unstructured.UnstructuredJSONScheme, &cm)
	cm.Annotations[kudo.LastAppliedConfigAnnotation] = string(cmString)

	cmUnstructuredData, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&cm)
	cmUnstructured := unstructured.Unstructured{Object: cmUnstructuredData}

	// Test retrieval from api-server
	testClient := fake.NewFakeClientWithScheme(scheme.Scheme, &cm)
	dc := newDependencyCalculator(testClient, []*unstructured.Unstructured{})
	obj, err := dc.resourceDependency(resourceDependency{gvk: typeConfigMap, name: "configmap", namespace: "namespace"})
	assert.NilError(t, err, "resourceDependency return error")
	cmResult := &v1.ConfigMap{}
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), cmResult)
	assert.DeepEqual(t, cm.Data, cmResult.Data)

	// Test retrieval from taskObjects list
	testClient = fake.NewFakeClientWithScheme(scheme.Scheme)
	dc = newDependencyCalculator(testClient, []*unstructured.Unstructured{&cmUnstructured})
	obj, err = dc.resourceDependency(resourceDependency{gvk: typeConfigMap, name: "configmap", namespace: "namespace"})
	assert.NilError(t, err, "resourceDependency return error")
	cmResult = &v1.ConfigMap{}
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), cmResult)
	assert.DeepEqual(t, cm.Data, cmResult.Data)

}
