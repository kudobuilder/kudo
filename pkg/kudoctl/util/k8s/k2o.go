package k8s

import (
	"fmt"
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

// FrameworkExists checks if a given Framework object is installed
func (k *K2oClient) FrameworkExists(name string) bool {
	_, err := k.client.KudoV1alpha1().Frameworks(vars.Namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

// AnyFrameworkVersionExists checks if any FrameworkVersion object matches to the given Framework name
func (k *K2oClient) AnyFrameworkVersionExists(framework string) bool {
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

	return true
}

// AnyFrameworkVersionOutdated checks if any FrameworkVersion object matches to the given Framework name and
// if not it returns false. False means that for the given Framework the most recent official FrameworkVersion
// is not installed.
func (k *K2oClient) FrameworkVersionOutdated(framework, mostRecentVersion string) bool {
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
			fmt.Println("v.Spec.Version:", v.Spec.Version, "mostRecentVersion:", mostRecentVersion)
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

// FrameworkVersion returns the current version of the Framework object installed
func (k *K2oClient) GetVersionOfFramework(name string) (string, error) {
	framework, err := k.client.KudoV1alpha1().Frameworks(vars.Namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return "", errors.WithMessage(err, "getting framework")
	}
	fmt.Println(framework)
	return "", nil
}
