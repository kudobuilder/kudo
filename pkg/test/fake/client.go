package fake

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	coretesting "k8s.io/client-go/testing"
)

type CachedDiscovery struct {
	fakediscovery.FakeDiscovery
}

func (d *CachedDiscovery) Fresh() bool {
	return true
}

func (d *CachedDiscovery) Invalidate() {

}

func CustomCachedDiscoveryClient(additionalResources ...*metav1.APIResourceList) discovery.CachedDiscoveryInterface {
	commonResources := []*metav1.APIResourceList{
		{
			GroupVersion: corev1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "pod", Namespaced: true, Kind: "Pod"},
				{Name: "namespace", Namespaced: false, Kind: "Namespace"},
				{Name: "service", Namespaced: true, Kind: "Service"},
				{Name: "configmap", Namespaced: true, Kind: "ConfigMap"},
				{Name: "secret", Namespaced: true, Kind: "Secret"},
			},
		},
		{
			GroupVersion: appsv1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "statefulset", Namespaced: true, Kind: "StatefulSet"},
				{Name: "deployment", Namespaced: true, Kind: "Deployment"},
			},
		},
		{
			GroupVersion: batchv1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "job", Namespaced: true, Kind: "Job"},
			},
		},
		{
			GroupVersion: batchv1beta1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "job", Namespaced: true, Kind: "CronJob"},
			},
		},
		{
			GroupVersion: apiextv1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "customresourcedefinitions", Namespaced: false, Kind: "CustomResourceDefinition"},
			},
		},
	}

	resources := append(commonResources, additionalResources...)

	return &CachedDiscovery{
		FakeDiscovery: fakediscovery.FakeDiscovery{
			Fake: &coretesting.Fake{
				Resources: resources,
			},
		},
	}
}

// CachedDiscoveryClient returns a fake discovery client that is populated with some types for use in
// unit tests.
func CachedDiscoveryClient() discovery.CachedDiscoveryInterface {
	return CustomCachedDiscoveryClient()
}
