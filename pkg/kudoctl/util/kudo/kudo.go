package kudo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// NewClientFromK8s creates KUDO client from kubernetes client interface
func NewClientFromK8s(client versioned.Interface) *Client {
	result := Client{}
	result.clientset = client
	return &result
}

// OperatorExistsInCluster checks if a given Operator object is installed on the current k8s cluster
func (c *Client) OperatorExistsInCluster(name, namespace string) bool {
	operator, err := c.clientset.KudoV1alpha1().Operators(namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return false
	}
	fmt.Printf("operator.kudo.dev/%s unchanged\n", operator.Name)
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
//      		kudo.dev/operator: kafka
// This function also just returns true if the Instance matches a specific OperatorVersion of a Operator
func (c *Client) InstanceExistsInCluster(operatorName, namespace, version, instanceName string) (bool, error) {
	instances, err := c.clientset.KudoV1alpha1().Instances(namespace).List(v1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", kudo.OperatorLabel, operatorName)})
	if err != nil {
		return false, err
	}
	if len(instances.Items) < 1 {
		return false, nil
	}

	// TODO: check function that actual checks for the OperatorVersion named e.g. "test-1.0" to exist
	var i int
	for _, v := range instances.Items {
		if v.Spec.OperatorVersion.Name == operatorName+"-"+version && v.ObjectMeta.Name == instanceName {
			i++
		}
	}

	// No instance exist with this operatorName and OV exists
	if i == 0 {
		return false, nil
	}
	return true, nil
}

// GetInstance queries kubernetes api for instance of given name in given namespace
// returns error for error conditions. Instance not found is not considered an error and will result in 'nil, nil'
func (c *Client) GetInstance(name, namespace string) (*v1alpha1.Instance, error) {
	instance, err := c.clientset.KudoV1alpha1().Instances(namespace).Get(name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return instance, err
}

// GetOperatorVersion queries kubernetes api for operatorversion of given name in given namespace
// returns error for all other errors that not found, not found is treated as result being 'nil, nil'
func (c *Client) GetOperatorVersion(name, namespace string) (*v1alpha1.OperatorVersion, error) {
	ov, err := c.clientset.KudoV1alpha1().OperatorVersions(namespace).Get(name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return ov, err
}

//  patchStringValue specifies a patch operation for a string.
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// UpdateInstance updates operatorversion on instance
func (c *Client) UpdateInstance(instanceName, namespace, operatorVersionName string, parameters map[string]string) error {
	instancePatch := []patchStringValue{
		{
			Op: "replace",
			Path: "/spec/operatorVersion/name",
			Value: operatorVersionName,
		},
	}
	if parameters != nil {
		for n, v := range parameters {
			instancePatch = append(instancePatch, patchStringValue{
				Op: "replace",
				Path: fmt.Sprintf("/spec/parameters/%s", n),
				Value: v,
			})
		}
	}
	serializedPatch, err := json.Marshal(instancePatch)
	if err != nil {
		return err
	}
	_, err = c.clientset.KudoV1alpha1().Instances(namespace).Patch(instanceName, types.JSONPatchType, serializedPatch)
	return err
}

// ListInstances lists all instances of given operator installed in the cluster in a given ns
func (c *Client) ListInstances(namespace string) ([]string, error) {
	instances, err := c.clientset.KudoV1alpha1().Instances(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	existingInstances := []string{}

	for _, v := range instances.Items {
		existingInstances = append(existingInstances, v.Name)
	}
	return existingInstances, nil
}

// OperatorVersionsInstalled lists all the versions of given operator installed in the cluster in given ns
func (c *Client) OperatorVersionsInstalled(operatorName, namespace string) ([]string, error) {
	ov, err := c.clientset.KudoV1alpha1().OperatorVersions(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	existingVersions := []string{}

	for _, v := range ov.Items {
		if strings.HasPrefix(v.Name, operatorName) {
			existingVersions = append(existingVersions, v.Spec.Version)
		}
	}
	return existingVersions, nil
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
