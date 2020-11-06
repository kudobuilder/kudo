package status

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util/podutils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/resource"
)

func isJobTerminallyFailed(job *batchv1.Job) (bool, string, error) {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true, fmt.Sprintf("job %q has failed terminally: %s", job.Name, c.Message), nil
		}
	}
	return false, "", nil
}

// IsTerminallyFailed returns true if a resource will never become healthy anymore and a corresponding message
// The returned msg is optional and should reflect the terminal state in a human readable form and potential reason.
// If the returned error is non-nil, the other returned values can be undefined and should not be used.
// This is a generic function and works on all resource types.
func IsDeleted(client client.Client, discovery discovery.CachedDiscoveryInterface, obj runtime.Object) (deleted bool, msg string, err error) {
	key, err := resource.ObjectKeyFromObject(obj, discovery)
	if err != nil {
		return false, "", fmt.Errorf("failed to get object key from object: %v", err)
	}
	newObj := obj.DeepCopyObject()
	err = client.Get(context.TODO(), key, newObj)
	if err == nil {
		// Object was retrieved without error - not deleted
		return false, fmt.Sprintf("%s/%s is not deleted", key.Namespace, key.Name), nil
	}
	if apierrors.IsNotFound(err) {
		// Object was not found - deleted
		return true, fmt.Sprintf("%s/%s is deleted", key.Namespace, key.Name), nil
	}

	// We got a different error - it's an error
	return false, "", fmt.Errorf("%s/%s is not deleted: %v", key.Namespace, key.Name, err)
}

// IsTerminallyFailed returns true if a resource will never become healthy anymore and a corresponding message
// The returned msg is optional and should reflect the terminal state in a human readable form and may include
// a potential reason.
// If the returned error is non-nil, the other returned values can be undefined and should not be used.
// Must be implemented for each type; all unimplemented resources are considered non-terminal by default.
func IsTerminallyFailed(obj runtime.Object) (terminal bool, msg string, err error) {
	if obj == nil {
		return true, "", nil
	}

	if obj, ok := obj.(*batchv1.Job); ok {
		return isJobTerminallyFailed(obj)
	}
	return false, "", nil
}

// IsHealthy returns whether an object is healthy and a corresponding message.
// The message is optional and may be empty in any case; it is human readable and should reflect
// the returned healthy status and an optional reason.
// When the returned error is non-nil, all other parameters can have undefined values and should not
// be used.
// Must be implemented for each type; all unimplemented resources are considered healthy by default.
func IsHealthy(obj runtime.Object) (healthy bool, msg string, err error) {
	if obj == nil {
		return true, "", nil
	}
	switch obj := obj.(type) {
	case *apiextv1beta1.CustomResourceDefinition:
		for _, c := range obj.Status.Conditions {
			if c.Type == apiextv1beta1.Established && c.Status == apiextv1beta1.ConditionTrue {
				return true, fmt.Sprintf("CRD %s is now healthy", obj.Name), nil
			}
		}
		return false, fmt.Sprintf("CRD %s is not healthy ( Conditions: %v )", obj.Name, obj.Status.Conditions), nil

	case *apiextv1.CustomResourceDefinition:
		for _, c := range obj.Status.Conditions {
			if c.Type == apiextv1.Established && c.Status == apiextv1.ConditionTrue {
				return true, fmt.Sprintf("CRD %s is now healthy", obj.Name), nil
			}
		}
		return false, fmt.Sprintf("CRD %s is not healthy ( Conditions: %v )", obj.Name, obj.Status.Conditions), nil

	case *appsv1.StatefulSet, *appsv1beta1.StatefulSet, *appsv1beta2.StatefulSet:
		objUnstructured, err := toUnstructured(obj)
		if err != nil {
			return false, "", err
		}
		statusViewer := &polymorphichelpers.StatefulSetStatusViewer{}
		msg, done, err := statusViewer.Status(objUnstructured, 0)
		if err != nil {
			return false, "", err
		}
		if done {
			return true, fmt.Sprintf("statefulset %q is marked healthy", objUnstructured.GetName()), nil
		}
		return false, fmt.Sprintf("statefulset %q is not healthy: %s", objUnstructured.GetName(), msg), nil

	case *appsv1.Deployment, *appsv1beta1.Deployment, *appsv1beta2.Deployment:
		objUnstructured, err := toUnstructured(obj)
		if err != nil {
			return false, "", err
		}
		statusViewer := &polymorphichelpers.DeploymentStatusViewer{}
		msg, done, err := statusViewer.Status(objUnstructured, 0)
		if err != nil {
			return false, "", err
		}
		if done {
			return true, fmt.Sprintf("deployment %v is marked healthy", objUnstructured.GetName()), nil
		}
		return false, fmt.Sprintf("deployment %q is not healthy: %s", objUnstructured.GetName(), msg), nil

	case *batchv1.Job:
		if obj.Status.Succeeded == int32(1) {
			return true, fmt.Sprintf("job %q is marked healthy", obj.Name), nil
		}
		return false, fmt.Sprintf("job %q still running or failed", obj.Name), nil

	case *kudoapi.Instance:
		// if there is no scheduled plan, then we're done
		if obj.Spec.PlanExecution.PlanName == "" {
			return true, fmt.Sprintf("instance %s/%s is marked healthy", obj.Namespace, obj.Name), nil
		}
		return false, fmt.Sprintf("instance %s/%s active plan is in state %v", obj.Namespace, obj.Name, obj.Spec.PlanExecution.Status), nil

	case *corev1.Pod:
		if obj.Status.Phase == corev1.PodRunning && podutils.IsPodReady(obj) {
			return true, "", nil
		}
		return false, fmt.Sprintf("pod %s/%s is not running yet: %s", obj.Namespace, obj.Name, obj.Status.Phase), nil

	case *corev1.Namespace:
		if obj.Status.Phase == corev1.NamespaceActive {
			return true, "", nil
		}
		return false, fmt.Sprintf("namespace %s is not active: %s", obj.Name, obj.Status.Phase), nil

	case *corev1.Service:
		return isServiceHealthy(obj)

	// unless we build logic for what a healthy object is, assume it's healthy when created.
	default:
		return true, fmt.Sprintf("unknown type %s is marked healthy by default", reflect.TypeOf(obj)), nil
	}
}

// Service health depends on the ingress type.
// To be considered healthy, a service needs to be accessible by its cluster IP.
// If the service is load-balanced, the balancer need to have an ingress defined.
// Adapted from https://github.com/helm/helm/blob/v3.3.4/pkg/kube/wait.go#L185.
func isServiceHealthy(obj *corev1.Service) (healthy bool, msg string, err error) {
	// ExternalName services are external to cluster. KUDO shouldn't be checking to see if they're 'ready' (i.e. have an IP set).
	if obj.Spec.Type == corev1.ServiceTypeExternalName {
		return true, fmt.Sprintf("external name service %s/%s is marked healthy", obj.Namespace, obj.Name), nil
	}

	if obj.Spec.ClusterIP == "" {
		return false, fmt.Sprintf("service %s/%s does not have cluster IP address", obj.Namespace, obj.Name), nil
	}

	// Check if the service has a LoadBalancer and that balancer has an Ingress defined.
	if obj.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if len(obj.Spec.ExternalIPs) > 0 {
			return true, fmt.Sprintf("service %s/%s has external IP addresses (%v), marked healthy", obj.Namespace, obj.Name, obj.Spec.ExternalIPs), nil
		}

		if obj.Status.LoadBalancer.Ingress == nil {
			return false, fmt.Sprintf("service %s/%s does not have load balancer ingress IP address", obj.Namespace, obj.Name), nil
		}
	}

	// If none of the above conditions are met, we can assume that the service is healthy.
	return true, fmt.Sprintf("service %s/%s is marked healthy", obj.Namespace, obj.Name), nil
}

func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: unstructMap}, nil
}
