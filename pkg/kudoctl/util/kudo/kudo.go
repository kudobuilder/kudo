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

	_, err = kudoClientset.KudoV1alpha1().Operators(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "operators")
	}
	_, err = kudoClientset.KudoV1alpha1().OperatorVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "operatorversions")
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
	_, err := c.clientset.KudoV1alpha1().Operators(namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "operators")
	}
	_, err = c.clientset.KudoV1alpha1().OperatorVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "operatorversions")
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

// OperatorExistsInCluster checks if a given Operator object is installed on the current k8s cluster
func (c *Client) OperatorExistsInCluster(name, namespace string) bool {
	operator, err := c.clientset.KudoV1alpha1().Operators(namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return false
	}
	fmt.Printf("operator.kudo.k8s.io/%s unchanged\n", operator.Name)
	return true
}

// AnyOperatorVersionExistsInCluster checks if any OperatorVersion object matches to the given Operator name
// in the cluster
func (c *Client) AnyOperatorVersionExistsInCluster(operator string, namespace string) bool {
	fv, err := c.clientset.KudoV1alpha1().OperatorVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return false
	}
	if len(fv.Items) < 1 {
		return false
	}

	var i int
	for _, v := range fv.Items {
		if strings.HasPrefix(v.Name, operator) {
			i++
		}
	}
	if i < 1 {
		return false
	}
	fmt.Printf("operatorversion.kudo.k8s.io/%s unchanged\n", operator)
	return true
}

// InstanceExistsInCluster checks if any OperatorVersion object matches to the given Operator name
// in the cluster.
// An Instance has two identifiers:
// 		1) Spec.OperatorVersion.Name
// 		spec:
//    		operatorVersion:
//      		name: kafka-2.11-2.4.0
// 		2) LabelSelector
// 		metadata:
//    		creationTimestamp: "2019-02-28T14:39:20Z"
//    		generation: 1
//    		labels:
//      		controller-tools.k8s.io: "1.0"
//      		operator: kafka
// This function also just returns true if the Instance matches a specific OperatorVersion of a Operator
func (c *Client) InstanceExistsInCluster(name, namespace, version, instanceName string) (bool, error) {
	instances, err := c.clientset.KudoV1alpha1().Instances(namespace).List(v1.ListOptions{LabelSelector: "operator=" + name})
	if err != nil {
		return false, err
	}
	if len(instances.Items) < 1 {
		return false, nil
	}

	// TODO: check function that actual checks for the OperatorVersion named e.g. "test-1.0" to exist
	var i int
	for _, v := range instances.Items {
		if v.Spec.OperatorVersion.Name == name+"-"+version && v.ObjectMeta.Name == instanceName {
			i++
		}
	}

	// No instance exist with this name and FV exists
	if i == 0 {
		return false, nil
	}
	return true, nil
}

// OperatorVersionInClusterOutOfSync checks if any OperatorVersion object matches a given Operator name and
// if not it returns false. False means that for the given Operator the most recent official OperatorVersion
// is not installed in the cluster or an error occurred.
func (c *Client) OperatorVersionInClusterOutOfSync(operator, mostRecentVersion, namespace string) bool {
	fv, err := c.clientset.KudoV1alpha1().OperatorVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return false
	}
	if len(fv.Items) < 1 {
		return false
	}

	var i int
	for _, v := range fv.Items {
		if strings.HasPrefix(v.Name, operator) {
			if v.Spec.Version == mostRecentVersion {
				i++
			}
		}
	}
	return !(i < 1)
}

// InstallOperatorObjToCluster expects a valid Operator obj to install
func (c *Client) InstallOperatorObjToCluster(obj *v1alpha1.Operator, namespace string) (*v1alpha1.Operator, error) {
	createdObj, err := c.clientset.KudoV1alpha1().Operators(namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing Operator")
	}
	return createdObj, nil
}

// InstallOperatorVersionObjToCluster expects a valid Operator obj to install
func (c *Client) InstallOperatorVersionObjToCluster(obj *v1alpha1.OperatorVersion, namespace string) (*v1alpha1.OperatorVersion, error) {
	createdObj, err := c.clientset.KudoV1alpha1().OperatorVersions(namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing OperatorVersion")
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
