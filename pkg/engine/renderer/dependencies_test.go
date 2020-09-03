package renderer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestGetResources(t *testing.T) {
	cm := corev1.ConfigMap{
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
	cmWithoutAnnotation := cm.DeepCopy()
	cmString, _ := runtime.Encode(unstructured.UnstructuredJSONScheme, &cm)
	cm.Annotations[kudo.LastAppliedConfigAnnotation] = string(cmString)

	cmUnstructuredData, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&cm)
	cmUnstructured := unstructured.Unstructured{Object: cmUnstructuredData}

	tests := []struct {
		name        string
		taskObjects []*unstructured.Unstructured
		client      client.Client
	}{
		{name: "from api server", taskObjects: []*unstructured.Unstructured{}, client: fake.NewFakeClientWithScheme(scheme.Scheme, &cm)},
		{name: "from task objects", taskObjects: []*unstructured.Unstructured{&cmUnstructured}, client: fake.NewFakeClientWithScheme(scheme.Scheme)},
		{name: "from api server without annotation", taskObjects: []*unstructured.Unstructured{}, client: fake.NewFakeClientWithScheme(scheme.Scheme, cmWithoutAnnotation)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dc := newDependencyCalculator(tt.client, tt.taskObjects)
			obj, err := dc.resourceDependency(resourceDependency{gvk: typeConfigMap, name: "configmap", namespace: "namespace"})
			assert.NoError(t, err, "resourceDependency return error")
			cmResult := &corev1.ConfigMap{}
			_ = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), cmResult)
			assert.Equal(t, cm.Data, cmResult.Data)
		})
	}
}

func configMapVolume(name string, objectRef string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: objectRef},
			},
		},
	}
}

func secretVolume(name string, secretName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

func TestSetDependenciesHash(t *testing.T) {
	tests := []struct {
		name   string
		obj    runtime.Object
		assert func(us *unstructured.Unstructured)
	}{
		{name: "statefulset", obj: statefulSet("somename", "namespace"), assert: func(us *unstructured.Unstructured) {
			sts := &v1.StatefulSet{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(us.UnstructuredContent(), &sts)
			assert.NoError(t, err)
			assert.Equal(t, "fancyHash", sts.Spec.Template.Annotations[kudo.DependenciesHashAnnotation])
		}},
		{name: "cronjob", obj: cronjob("somename", "namespace"), assert: func(us *unstructured.Unstructured) {
			c := &v1beta1.CronJob{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(us.UnstructuredContent(), &c)
			assert.NoError(t, err)
			assert.Equal(t, "fancyHash", c.Spec.JobTemplate.Spec.Template.Annotations[kudo.DependenciesHashAnnotation])
		}},
		{name: "no change in pod", obj: pod("somename", "namespace"), assert: func(us *unstructured.Unstructured) {
			p := &corev1.Pod{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(us.UnstructuredContent(), &p)
			assert.NoError(t, err)
			assert.Equal(t, pod("somename", "namespace"), p)
		}},
		{name: "no change in unstructured CRD", obj: unstructuredCrd("crd", "namespace"), assert: func(us *unstructured.Unstructured) {
			assert.Equal(t, unstructuredCrd("crd", "namespace"), us)
		}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cmUnstructuredData, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.obj)
			cmUnstructured := &unstructured.Unstructured{Object: cmUnstructuredData}

			err := setTemplateHash(cmUnstructured, "fancyHash")
			assert.NoError(t, err)

			tt.assert(cmUnstructured)
		})
	}
}

func TestCalculateDependencies(t *testing.T) {
	namespace := "default"
	ssBase := statefulSet("statefulset", namespace)

	tests := []struct {
		name     string
		modify   func(set *v1.StatefulSet)
		expected resourceDependencies
	}{
		{name: "no-dependencies", modify: func(sts *v1.StatefulSet) {}, expected: resourceDependencies{}},
		{name: "configmap", modify: func(sts *v1.StatefulSet) {
			sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, configMapVolume("ConfigMap", "configmap"))
		}, expected: resourceDependencies{
			resourceDependency{
				gvk:       typeConfigMap,
				name:      "configmap",
				namespace: namespace,
			},
		}},
		{name: "secret", modify: func(sts *v1.StatefulSet) {
			sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, secretVolume("MySecret", "secret"))
		}, expected: resourceDependencies{
			resourceDependency{
				gvk:       typeSecret,
				name:      "secret",
				namespace: namespace,
			},
		}},
		{name: "pullSecret", modify: func(sts *v1.StatefulSet) {
			sts.Spec.Template.Spec.ImagePullSecrets = append(sts.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: "pullsecret"})
		}, expected: resourceDependencies{
			resourceDependency{
				gvk:       typeSecret,
				name:      "pullsecret",
				namespace: namespace,
			},
		}},
	}

	for _, test := range tests {
		ss := ssBase.DeepCopy()
		test.modify(ss)

		cmUnstructuredData, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(ss)
		cmUnstructured := &unstructured.Unstructured{Object: cmUnstructuredData}

		deps, err := calculateResourceDependencies(cmUnstructured)
		assert.NoError(t, err)

		assert.Equal(t, len(test.expected), len(deps))
		assert.Equal(t, test.expected, deps, cmp.AllowUnexported(resourceDependency{}))
	}
}
