package diagnostics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	kudoutil "github.com/kudobuilder/kudo/pkg/util/kudo"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// resourceFuncsConfig - a wrapper for Kube and Kudo clients and common invocation parameters
// for loading Kube and Kudo resources
type resourceFuncsConfig struct {
	c           *kudo.Client
	ns          string
	instanceObj *kudoapi.Instance
	opts        metav1.ListOptions
	logOpts     corev1.PodLogOptions
}

// newInstanceResources is a configuration for instance-related resources
func newInstanceResources(instanceName string, options *Options, c *kudo.Client, s *env.Settings) (*resourceFuncsConfig, error) {
	instance, err := c.GetInstance(instanceName, s.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s/%s: %v", s.Namespace, instanceName, err)
	}
	if instance == nil {
		return nil, fmt.Errorf("instance %s/%s not found", s.Namespace, instanceName)
	}
	return &resourceFuncsConfig{
		c:           c,
		ns:          s.Namespace,
		instanceObj: instance,
		opts:        metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", kudoutil.OperatorLabel, instance.Labels[kudoutil.OperatorLabel])},
		logOpts:     corev1.PodLogOptions{SinceSeconds: options.LogSince},
	}, nil
}

// newKudoResources is a configuration for Kudo controller related resources
// panics if used to load Kudo CRDs (e.g. instance etc.)
func newKudoResources(options *Options, c *kudo.Client) (*resourceFuncsConfig, error) {
	opts := metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", kudoinit.DefaultKudoLabel)}
	ns, err := c.KubeClientset.CoreV1().Namespaces().List(context.TODO(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get kudo system namespace: %v", err)
	}
	if ns == nil || len(ns.Items) == 0 {
		return nil, fmt.Errorf("kudo system namespace not found")
	}
	return &resourceFuncsConfig{
		c:       c,
		ns:      ns.Items[0].Name,
		opts:    opts,
		logOpts: corev1.PodLogOptions{SinceSeconds: options.LogSince},
	}, nil
}

func newDependenciesResources(instanceName string, options *Options, c *kudo.Client, s *env.Settings) ([]*resourceFuncsConfig, error) {
	instance, err := c.GetInstance(instanceName, s.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s/%s: %v", s.Namespace, instanceName, err)
	}
	if instance == nil {
		return nil, fmt.Errorf("instance %s/%s not found", s.Namespace, instanceName)
	}

	children, err := c.GetChildInstances(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get children of instance %s/%s: %v", s.Namespace, instanceName, err)
	}

	configs := make([]*resourceFuncsConfig, 0, len(children))

	for _, child := range children {
		child := child

		configs = append(configs, &resourceFuncsConfig{
			c:           c,
			ns:          s.Namespace,
			instanceObj: &child,
			opts:        metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", kudoutil.OperatorLabel, child.Labels[kudoutil.OperatorLabel])},
			logOpts:     corev1.PodLogOptions{SinceSeconds: options.LogSince},
		})
	}

	return configs, nil
}

type stringGetter func() string

func (r *resourceFuncsConfig) instance() (runtime.Object, error) {
	return r.instanceObj, nil
}

func (r *resourceFuncsConfig) operatorVersion(name stringGetter) func() (runtime.Object, error) {
	return func() (runtime.Object, error) {
		return r.c.GetOperatorVersion(name(), r.ns)
	}
}

func (r *resourceFuncsConfig) operator(name stringGetter) func() (runtime.Object, error) {
	return func() (runtime.Object, error) {
		return r.c.GetOperator(name(), r.ns)
	}
}

func (r *resourceFuncsConfig) deployments() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.AppsV1().Deployments(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) pods() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.CoreV1().Pods(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) services() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.CoreV1().Services(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) replicaSets() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.AppsV1().ReplicaSets(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) statefulSets() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.AppsV1().StatefulSets(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) serviceAccounts() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.CoreV1().ServiceAccounts(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) clusterRoleBindings() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.RbacV1().ClusterRoleBindings().List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) roleBindings() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.RbacV1().RoleBindings(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) clusterRoles() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.RbacV1().ClusterRoles().List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) roles() (runtime.Object, error) {
	obj, err := r.c.KubeClientset.RbacV1().Roles(r.ns).List(context.TODO(), r.opts)
	return obj, err
}

func (r *resourceFuncsConfig) log(podName, containerName string) (io.ReadCloser, error) {
	req := r.c.KubeClientset.CoreV1().Pods(r.ns).GetLogs(podName, &corev1.PodLogOptions{SinceSeconds: r.logOpts.SinceSeconds, Container: containerName})
	// a hack for tests: fake client returns rest.Request{} for GetLogs and Stream panics with null-pointer
	if reflect.DeepEqual(*req, rest.Request{}) {
		return ioutil.NopCloser(&bytes.Buffer{}), nil
	}
	return req.Stream(context.TODO())
}
