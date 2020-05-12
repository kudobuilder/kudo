package diagnostics

// set of resource provide funcs
// some require information from the context that should be put by some other funcs, therefore the correct order of calling is required

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

type ResourceFuncsConfig struct {
	c           *kube.Client
	kc          *kudo.Client
	ns          string
	instanceObj *v1beta1.Instance
	opts        metav1.ListOptions
	logOpts     corev1.PodLogOptions
}

// instanceObj related resources
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

	return &ResourceFuncsConfig{
		c:           c,
		kc:          kc,
		ns:          s.Namespace,
		instanceObj: instance,
		opts:        metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", kudoutil.OperatorLabel, instance.Labels[kudoutil.OperatorLabel])},
		logOpts:     corev1.PodLogOptions{SinceSeconds: &opts.LogSince},
	}, nil
}

// kudo controller related resources
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

// Instance - updates context with Instance name and OperatroVersion name
func Instance(r *ResourceFuncsConfig, ctx *processingContext) (runtime.Object, error) {
	ctx.instanceName = r.instanceObj.Name
	ctx.operatorVersionName = r.instanceObj.Spec.OperatorVersion.Name
	return r.instanceObj, nil
}

// OperatorVersion - depends on Instance, updates context with Operator name
func OperatorVersion(r *ResourceFuncsConfig, ctx *processingContext) (runtime.Object, error) {
	ovName := ctx.operatorVersionName
	obj, err := r.kc.GetOperatorVersion(ovName, r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to get operatorversion %s/%s: %v", r.ns, ovName, err)
	}
	if obj == nil {
		return nil, fmt.Errorf("operator version not found")
	}
	ctx.operatorName = obj.Spec.Operator.Name
	return obj, err
}

func Operator(r *ResourceFuncsConfig, ctx *processingContext) (runtime.Object, error) {
	opName := ctx.operatorName
	obj, err := r.kc.GetOperator(opName, r.ns)
	return obj, err
}

func Deployments(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.AppsV1().Deployments(r.ns).List(r.opts)
	return obj, err
}

// Pods - updates context with pod names
func Pods(r *ResourceFuncsConfig, ctx *processingContext) (runtime.Object, error) {
	obj, err := r.c.KubeClient.CoreV1().Pods(r.ns).List(r.opts)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	for _, pod := range obj.Items {
		ctx.podNames = append(ctx.podNames, pod.Name)
	}
	return obj, err
}

func Services(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.CoreV1().Services(r.ns).List(r.opts)
	return obj, err
}

func ReplicaSets(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.AppsV1().ReplicaSets(r.ns).List(r.opts)
	return obj, err
}

func StatefulSets(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.AppsV1().StatefulSets(r.ns).List(r.opts)
	return obj, err
}

func ServiceAccounts(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.CoreV1().ServiceAccounts(r.ns).List(r.opts)
	return obj, err
}

func ClusterRoleBindings(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().ClusterRoleBindings().List(r.opts)
	return obj, err
}

func RoleBindings(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().RoleBindings(r.ns).List(r.opts)
	return obj, err
}

func ClusterRoles(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().ClusterRoles().List(r.opts)
	return obj, err
}

func Roles(r *ResourceFuncsConfig) (runtime.Object, error) {
	obj, err := r.c.KubeClient.RbacV1().Roles(r.ns).List(r.opts)
	return obj, err
}

// Log - depends on Pods
func Log(r *ResourceFuncsConfig, podName string) (io.ReadCloser, error) {
	return r.c.KubeClient.CoreV1().Pods(r.ns).GetLogs(podName, &r.logOpts).Stream()
}
