package kube

import (
	"fmt"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

// Client provides access different K8S clients
type Client struct {
	KubeClient    kubernetes.Interface
	ExtClient     apiextensionsclient.Interface
	DynamicClient dynamic.Interface
}

// getConfig returns a Kubernetes client config for a given kubeconfig.
func getConfig(kubeconfig string) clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
}

func getRestConfig(kubeconfig string) (*rest.Config, error) {
	config, err := getConfig(kubeconfig).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config using configuration %q: %s", kubeconfig, err)
	}
	clog.V(4).Printf("configuration from %q finds host %v", kubeconfig, config.Host)
	return config, nil
}

// GetKubeClient provides k8s client for kubeconfig
func GetKubeClient(kubeconfig string) (*Client, error) {
	config, err := getRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	extClient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes dynamic client: %s", err)
	}

	return &Client{
		KubeClient:    client,
		ExtClient:     extClient,
		DynamicClient: dynamicClient}, nil
}
