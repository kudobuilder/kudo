package status

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
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
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

func isJobTerminallyFailed(job *batchv1.Job) (bool, string, error) {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true, fmt.Sprintf("job %q has failed terminally: %s", job.Name, c.Message), nil
		}
	}
	return false, "", nil
}

func IsDeleted(client client.Client, discovery discovery.CachedDiscoveryInterface, objs []runtime.Object) error {
	for _, obj := range objs {
		key, err := resource.ObjectKeyFromObject(obj, discovery)
		if err != nil {
			return err
		}
		newObj := obj.DeepCopyObject()
		err = client.Get(context.TODO(), key, newObj)
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("%s/%s is not deleted", key.Namespace, key.Name)
		}
	}
	return nil
}

// IsTerminallyFailed returns true if a resource will never become healthy anymore
func IsTerminallyFailed(obj runtime.Object) (bool, string, error) {
	if obj == nil {
		return true, "", nil
	}

	switch obj := obj.(type) {
	case *batchv1.Job:
		return isJobTerminallyFailed(obj)
	}
	return false, "", nil
}

// IsHealthy returns whether an object is healthy and a corresponding message
// Must be implemented for each type.
func IsHealthy(obj runtime.Object) (bool, string, error) {
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

	case *appsv1.StatefulSet:
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
			return true, fmt.Sprintf("statefulset %q is marked healthy", obj.Name), nil
		}
		return false, fmt.Sprintf("statefulset %q is not healthy: %s", obj.Name, msg), nil

	case *appsv1.Deployment:
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
			return true, fmt.Sprintf("deployment %v is marked healthy", obj.Name), nil
		}
		clog.V(2).Printf("deployment %v is NOT healthy. %s", obj.Name, msg)
		return false, msg, nil

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

	// unless we build logic for what a healthy object is, assume it's healthy when created.
	default:
		return true, fmt.Sprintf("unknown type %s is marked healthy by default", reflect.TypeOf(obj)), nil
	}
}

func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: unstructMap}, nil
}
