package prereq

import (
	apiextensionfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
)

func getFakeClient() *kube.Client {
	client := fake.NewSimpleClientset()

	extClient := apiextensionfake.NewSimpleClientset()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	return &kube.Client{
		KubeClient:    client,
		ExtClient:     extClient,
		DynamicClient: dynamicClient}
}
