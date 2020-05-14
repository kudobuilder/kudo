package diagnostics

import (
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	kudoutil "github.com/kudobuilder/kudo/pkg/util/kudo"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceFuncsConfig - a wrapper for Kube and Kudo clients and common invocation parameters
// for loading Kube and Kudo resources
type ResourceFuncsConfig struct {
	c           *kube.Client
	kc          *kudo.Client
	ns          string
	instanceObj *v1beta1.Instance
	opts        metav1.ListOptions
	logOpts     corev1.PodLogOptions
}

// NewInstanceResources is a configuration for Instance-related resources
func NewInstanceResources(opts *Options, s *env.Settings) (*ResourceFuncsConfig, error) {

	kc, err := kudo.NewClient(s.KubeConfig, s.RequestTimeout, s.Validate)
	if err != nil {
		return nil, fmt.Errorf("failed to create kudo client: %v", err)
	}
	c, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kube client: %v", err)
	}
	instance, err := kc.GetInstance(opts.Instance, s.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s/%s: %v", s.Namespace, opts.Instance, err)
	}
	if instance == nil {
		return nil, fmt.Errorf("instance %s/%s not found", s.Namespace, opts.Instance)
	}
	return &ResourceFuncsConfig{
		c:           c,
		kc:          kc,
		ns:          s.Namespace,
		instanceObj: instance,
		opts:        metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", kudoutil.OperatorLabel, instance.Labels[kudoutil.OperatorLabel])},
		logOpts:     corev1.PodLogOptions{SinceSeconds: &opts.LogSince},
	}, nil
}

// NewKudoResources is a configuration for Kudo controller related resources
// panics if used to load Kudo CRDs (e.g. Instance etc.)
func NewKudoResources(s *env.Settings) (*ResourceFuncsConfig, error) {
	c, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", appKudoManager)}
	ns, err := c.KubeClient.CoreV1().Namespaces().List(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get kudo system namespace: %v", err)
	}
	if ns == nil || len(ns.Items) == 0 {
		return nil, fmt.Errorf("kudo system namespace not found")
	}
	return &ResourceFuncsConfig{
		c:       c,
		ns:      ns.Items[0].Name,
		opts:    opts,
		logOpts: corev1.PodLogOptions{},
	}, nil
}

// Object implements runtime.Object and
// metav1.Object interfaces.
// copied from K8 internal type metaRuntimeInterface
type Object interface {
	runtime.Object
	metav1.Object
}

func isKudoCR(o runtime.Object) bool {
	kind := o.GetObjectKind().GroupVersionKind().Kind
	return kind == "Instance" || kind == "Operator" || kind == "OperatorVersion"
}

type stringGetter func() string

func (r *ResourceFuncsConfig) Instance() (runtime.Object, error) {
	return r.instanceObj, nil
}

func (r *ResourceFuncsConfig) OperatorVersion(name stringGetter) func() (runtime.Object, error) {
	return func() (runtime.Object, error) {
		return r.kc.GetOperatorVersion(name(), r.ns)
	}
}

func (r *ResourceFuncsConfig) Operator(name stringGetter) func() (runtime.Object, error) {
	return func() (runtime.Object, error) {
		return r.kc.GetOperator(name(), r.ns)
	}
}

func (r *ResourceFuncsConfig) Deployments() (runtime.Object, error) {
	obj, err := r.c.KubeClient.AppsV1().Deployments(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) Pods() (runtime.Object, error) {
	obj, err := r.c.KubeClient.CoreV1().Pods(r.ns).List(r.opts)
	//return obj, err
	_, _ = obj, err
	return nil, fmt.Errorf("fake err")
}

func (r *ResourceFuncsConfig) Services() (runtime.Object, error) {
	obj, err := r.c.KubeClient.CoreV1().Services(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) ReplicaSets() (runtime.Object, error) {
	obj, err := r.c.KubeClient.AppsV1().ReplicaSets(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) StatefulSets() (runtime.Object, error) {
	obj, err := r.c.KubeClient.AppsV1().StatefulSets(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) ServiceAccounts() (runtime.Object, error) {
	obj, err := r.c.KubeClient.CoreV1().ServiceAccounts(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) ClusterRoleBindings() (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().ClusterRoleBindings().List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) RoleBindings() (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().RoleBindings(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) ClusterRoles() (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().ClusterRoles().List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) Roles() (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().Roles(r.ns).List(r.opts)
	return obj, err
}

func (r *ResourceFuncsConfig) Log(podName string) (io.ReadCloser, error) {
	return r.c.KubeClient.CoreV1().Pods(r.ns).GetLogs(podName, &r.logOpts).Stream()
}
