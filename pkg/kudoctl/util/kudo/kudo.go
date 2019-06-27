package kudo

import (
	"fmt"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is a KUDO Client providing access to a clientset
type Client struct {
	clientset versioned.Interface
}

// NewClient creates new KUDO Client
func NewClient(namespace, kubeConfigPath string) (*Client, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// set default configs
	config.Timeout = time.Second * 3

	// create the clientset
	kudoClientset, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	_, err = kudoClientset.KudoV1alpha1().Frameworks(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "frameworks")
	}
	_, err = kudoClientset.KudoV1alpha1().FrameworkVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "frameworkversions")
	}
	_, err = kudoClientset.KudoV1alpha1().Instances(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "instances")
	}
	_, err = kudoClientset.KudoV1alpha1().PlanExecutions(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "planexecutions")
	}

	return &Client{
		clientset: kudoClientset,
	}, nil
}

// CRDsInstalled checks for essential CRDs of KUDO to be installed
func (c *Client) CRDsInstalled(namespace string) error {
	_, err := c.clientset.KudoV1alpha1().Frameworks(namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "frameworks")
	}
	_, err = c.clientset.KudoV1alpha1().FrameworkVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "frameworkversions")
	}
	_, err = c.clientset.KudoV1alpha1().Instances(namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "instances")
	}
	_, err = c.clientset.KudoV1alpha1().PlanExecutions(namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "planexecutions")
	}
	return nil
}

// FrameworkExistsInCluster checks if a given Framework object is installed on the current k8s cluster
func (c *Client) FrameworkExistsInCluster(name, namespace string) bool {
	framework, err := c.clientset.KudoV1alpha1().Frameworks(namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return false
	}
	fmt.Printf("framework.kudo.k8s.io/%s unchanged\n", framework.Name)
	return true
}

// AnyFrameworkVersionExistsInCluster checks if any FrameworkVersion object matches to the given Framework name
// in the cluster
func (c *Client) AnyFrameworkVersionExistsInCluster(framework string, namespace string) bool {
	fv, err := c.clientset.KudoV1alpha1().FrameworkVersions(namespace).List(v1.ListOptions{})
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

// InstanceExistsInCluster checks if any FrameworkVersion object matches to the given Framework name
// in the cluster.
// An Instance has two identifiers:
// 		1) Spec.FrameworkVersion.Name
// 		spec:
//    		frameworkVersion:
//      		name: kafka-2.11-2.4.0
// 		2) LabelSelector
// 		metadata:
//    		creationTimestamp: "2019-02-28T14:39:20Z"
//    		generation: 1
//    		labels:
//      		controller-tools.k8s.io: "1.0"
//      		framework: kafka
// This function also just returns true if the Instance matches a specific FrameworkVersion of a Framework
func (c *Client) InstanceExistsInCluster(name, namespace, version, instanceName string) (bool, error) {
	instances, err := c.clientset.KudoV1alpha1().Instances(namespace).List(v1.ListOptions{LabelSelector: "framework=" + name})
	if err != nil {
		return false, err
	}
	if len(instances.Items) < 1 {
		return false, nil
	}

	// TODO: check function that actual checks for the FrameworkVersion named e.g. "test-1.0" to exist
	var i int
	for _, v := range instances.Items {
		if v.Spec.FrameworkVersion.Name == name+"-"+version && v.ObjectMeta.Name == instanceName {
			i++
		}
	}

	// No instance exist with this name and FV exists
	if i == 0 {
		return false, nil
	}
	return true, nil
}

// FrameworkVersionInClusterOutOfSync checks if any FrameworkVersion object matches a given Framework name and
// if not it returns false. False means that for the given Framework the most recent official FrameworkVersion
// is not installed in the cluster or an error occurred.
func (c *Client) FrameworkVersionInClusterOutOfSync(framework, mostRecentVersion, namespace string) bool {
	fv, err := c.clientset.KudoV1alpha1().FrameworkVersions(namespace).List(v1.ListOptions{})
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
	return !(i < 1)
}

// InstallFrameworkObjToCluster expects a valid Framework obj to install
func (c *Client) InstallFrameworkObjToCluster(obj *v1alpha1.Framework, namespace string) (*v1alpha1.Framework, error) {
	createdObj, err := c.clientset.KudoV1alpha1().Frameworks(namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing Framework")
	}
	return createdObj, nil
}

// InstallFrameworkVersionObjToCluster expects a valid Framework obj to install
func (c *Client) InstallFrameworkVersionObjToCluster(obj *v1alpha1.FrameworkVersion, namespace string) (*v1alpha1.FrameworkVersion, error) {
	createdObj, err := c.clientset.KudoV1alpha1().FrameworkVersions(namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing FrameworkVersion")
	}
	return createdObj, nil
}

// InstallInstanceObjToCluster expects a valid Instance obj to install
func (c *Client) InstallInstanceObjToCluster(obj *v1alpha1.Instance, namespace string) (*v1alpha1.Instance, error) {
	createdObj, err := c.clientset.KudoV1alpha1().Instances(namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing Instance")
	}
	return createdObj, nil
}
