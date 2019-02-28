package k8s

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

type K2oClient struct {
	client *versioned.Clientset
}

// Create new k8s client
func NewK2oClient() (*K2oClient, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", vars.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K2oClient{
		client: clientset,
	}, nil
}

// CRDsInstalled checks for essential CRDs of KUDO to be installed
func (k *K2oClient) CRDsInstalled() error {
	_, err := k.client.KudoV1alpha1().Frameworks(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "frameworks")
	}
	_, err = k.client.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "frameworkversions")
	}
	_, err = k.client.KudoV1alpha1().Instances(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "instances")
	}
	_, err = k.client.KudoV1alpha1().PlanExecutions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "planexecutions")
	}
	return nil
}

// FrameworkExistsInCluster checks if a given Framework object is installed on the current k8s cluster
func (k *K2oClient) FrameworkExistsInCluster(name string) bool {
	framework, err := k.client.KudoV1alpha1().Frameworks(vars.Namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return false
	}
	fmt.Printf("framework.kudo.k8s.io/%s unchanged\n", framework.Name)
	return true
}

// AnyFrameworkVersionExistsInCluster checks if any FrameworkVersion object matches to the given Framework name
// in the cluster
func (k *K2oClient) AnyFrameworkVersionExistsInCluster(framework string) bool {
	fv, err := k.client.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return false
	}
	if len(fv.Items) < 1 {
		return false
	}

	var i int
	for _, v := range fv.Items {
		if strings.HasPrefix(v.Name, framework) {
			i++
		}
	}
	if i < 1 {
		return false
	}
	fmt.Printf("frameworkversion.kudo.k8s.io/%s unchanged\n", framework)
	return true
}

// AnyFrameworkVersionInClusterOutOfSync checks if any FrameworkVersion object matches to the given Framework name and
// if not it returns false. False means that for the given Framework the most recent official FrameworkVersion
// is not installed in the cluster.
func (k *K2oClient) FrameworkVersionInClusterOutOfSync(framework, mostRecentVersion string) bool {
	fv, err := k.client.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return false
	}
	if len(fv.Items) < 1 {
		return false
	}

	var i int
	for _, v := range fv.Items {
		if strings.HasPrefix(v.Name, framework) {
			if v.Spec.Version == mostRecentVersion {
				i++
			}
		}
	}
	if i < 1 {
		return false
	}
	return true
}

// InstallFrameworkYamlToCluster expects a valid Framework obj to install
func (k *K2oClient) InstallFrameworkYamlToCluster(obj *v1alpha1.Framework) (*v1alpha1.Framework, error) {
	createdObj, err := k.client.KudoV1alpha1().Frameworks(vars.Namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing Framework")
	}
	return createdObj, nil
}

// InstallFrameworkVersionYamlToCluster expects a valid Framework obj to install
func (k *K2oClient) InstallFrameworkVersionYamlToCluster(obj *v1alpha1.FrameworkVersion) (*v1alpha1.FrameworkVersion, error) {
	createdObj, err := k.client.KudoV1alpha1().FrameworkVersions(vars.Namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing FrameworkVersion")
	}
	return createdObj, nil
}
