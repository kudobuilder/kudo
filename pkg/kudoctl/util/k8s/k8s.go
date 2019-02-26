package k8s

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	client *kubernetes.Clientset
}

// Create new k8s client
func NewK8sClient() (*K8sClient, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", vars.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sClient{
		client: clientset,
	}, nil
}

func (k *K8sClient) FrameworkExists(name string) error {

	return nil
}
