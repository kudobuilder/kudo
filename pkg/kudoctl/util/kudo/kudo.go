package kudo

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"time"
)

type KudoClient struct {
	clientset versioned.Interface
}

// Create new Kudo client
func NewKudoClient() (*KudoClient, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", vars.KubeConfigPath)
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

	_, err = kudoClientset.KudoV1alpha1().Frameworks(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "frameworks")
	}
	_, err = kudoClientset.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "frameworkversions")
	}
	_, err = kudoClientset.KudoV1alpha1().Instances(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "instances")
	}
	_, err = kudoClientset.KudoV1alpha1().PlanExecutions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.WithMessage(err, "planexecutions")
	}

	return &KudoClient{
		clientset: kudoClientset,
	}, nil
}

// CRDsInstalled checks for essential CRDs of KUDO to be installed
func (k *KudoClient) CRDsInstalled() error {
	_, err := k.clientset.KudoV1alpha1().Frameworks(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "frameworks")
	}
	_, err = k.clientset.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "frameworkversions")
	}
	_, err = k.clientset.KudoV1alpha1().Instances(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "instances")
	}
	_, err = k.clientset.KudoV1alpha1().PlanExecutions(vars.Namespace).List(v1.ListOptions{})
	if err != nil {
		return errors.WithMessage(err, "planexecutions")
	}
	return nil
}

// FrameworkExistsInCluster checks if a given Framework object is installed on the current k8s cluster
func (k *KudoClient) FrameworkExistsInCluster(name string) bool {
	framework, err := k.clientset.KudoV1alpha1().Frameworks(vars.Namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return false
	}
	fmt.Printf("framework.kudo.k8s.io/%s unchanged\n", framework.Name)
	return true
}

// AnyFrameworkVersionExistsInCluster checks if any FrameworkVersion object matches to the given Framework name
// in the cluster
func (k *KudoClient) AnyFrameworkVersionExistsInCluster(framework string) bool {
	fv, err := k.clientset.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
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

// AnyInstanceExistsInCluster checks if any FrameworkVersion object matches to the given Framework name
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
func (k *KudoClient) AnyInstanceExistsInCluster(name, version string) bool {
	instances, err := k.clientset.KudoV1alpha1().Instances(vars.Namespace).List(v1.ListOptions{LabelSelector: "framework=" + name})
	if err != nil {
		return false
	}
	if len(instances.Items) < 1 {
		return false
	}

	// Todo: check function that actual checks for the FrameworkVersion named e.g. "test-1.0" to exist
	var i int
	for _, v := range instances.Items {
		if v.Spec.FrameworkVersion.Name == name+"-"+version {
			i++
		}
	}

	// This is when we don't find the version we are looking for
	if i < 1 {
		return false
	}
	fmt.Printf("instance.kudo.k8s.io/%s unchanged\n", name)
	return true
}

// AnyFrameworkVersionInClusterOutOfSync checks if any FrameworkVersion object matches a given Framework name and
// if not it returns false. False means that for the given Framework the most recent official FrameworkVersion
// is not installed in the cluster or an error occurred.
func (k *KudoClient) FrameworkVersionInClusterOutOfSync(framework, mostRecentVersion string) bool {
	fv, err := k.clientset.KudoV1alpha1().FrameworkVersions(vars.Namespace).List(v1.ListOptions{})
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

// InstallFrameworkObjToCluster expects a valid Framework obj to install
func (k *KudoClient) InstallFrameworkObjToCluster(obj *v1alpha1.Framework) (*v1alpha1.Framework, error) {
	createdObj, err := k.clientset.KudoV1alpha1().Frameworks(vars.Namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing Framework")
	}
	return createdObj, nil
}

// InstallFrameworkVersionObjToCluster expects a valid Framework obj to install
func (k *KudoClient) InstallFrameworkVersionObjToCluster(obj *v1alpha1.FrameworkVersion) (*v1alpha1.FrameworkVersion, error) {
	createdObj, err := k.clientset.KudoV1alpha1().FrameworkVersions(vars.Namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing FrameworkVersion")
	}
	return createdObj, nil
}

// InstallInstanceObjToCluster expects a valid Instance obj to install
func (k *KudoClient) InstallInstanceObjToCluster(obj *v1alpha1.Instance) (*v1alpha1.Instance, error) {
	createdObj, err := k.clientset.KudoV1alpha1().Instances(vars.Namespace).Create(obj)
	if err != nil {
		return nil, errors.WithMessage(err, "installing Instance")
	}
	return createdObj, nil
}
