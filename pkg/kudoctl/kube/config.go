package kube

import (
	"fmt"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig returns a Kubernetes client config for a given kubeconfig.
func GetConfig(kubeconfig string) clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
}

func getRestConfig(kubeconfig string) (*rest.Config, error) {
	config, err := GetConfig(kubeconfig).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config using configuration %q: %s", kubeconfig, err)
	}
	return config, nil
}

// GetKubeClient provides k8s client for kubeconfig
func GetKubeClient(kubeconfig string) (kubernetes.Interface, error) {
	config, err := getRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return client, nil
}

// GetKubeAPIExtClient provides a k8s api extension client (required for CRDs)
func GetKubeAPIExtClient(kubeconfig string) (*apiextensionsclient.Clientset, error) {
	config, err := getRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}

	return client, nil
}
