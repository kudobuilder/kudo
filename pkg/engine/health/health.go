package health

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/polymorphichelpers"

	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

func isJobTerminallyFailed(job *batchv1.Job) (bool, string) {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			log.Printf("HealthUtil: Job %q has failed: %s", job.Name, c.Message)
			return true, c.Message
		}
	}
	return false, ""
}

// IsHealthy returns whether an object is healthy. Must be implemented for each type.
func IsHealthy(obj runtime.Object) error {
	if obj == nil {
		return nil
	}
	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}

	objUnstructured := &unstructured.Unstructured{Object: unstructMap}
	switch obj := obj.(type) {
	case *appsv1.StatefulSet:
		statusViewer := &polymorphichelpers.StatefulSetStatusViewer{}
		msg, done, err := statusViewer.Status(objUnstructured, 0)
		if err != nil {
			return err
		}
		if !done {
			log.Printf("HealthUtil: Statefulset %v is NOT healthy. %s", obj.Name, msg)
			return errors.New(msg)
		}
		log.Printf("Statefulset %v is marked healthy\n", obj.Name)
		return nil
	case *appsv1.Deployment:
		statusViewer := &polymorphichelpers.DeploymentStatusViewer{}
		msg, done, err := statusViewer.Status(objUnstructured, 0)
		if err != nil {
			return err
		}
		if !done {
			log.Printf("HealthUtil: Deployment %v is NOT healthy. %s", obj.Name, msg)
			return errors.New(msg)
		}
		clog.V(2).Printf("Deployment %v is marked healthy\n", obj.Name)
		return nil
	case *batchv1.Job:

		if obj.Status.Succeeded == int32(1) {
			// Done!
			log.Printf("HealthUtil: Job %q is marked healthy", obj.Name)
			return nil
		}
		if terminal, msg := isJobTerminallyFailed(obj); terminal {
			return fmt.Errorf("%wHealthUtil: Job %q has failed terminally: %s", engine.ErrFatalExecution, obj.Name, msg)
		}

		return fmt.Errorf("job %q still running or failed", obj.Name)
	case *kudov1beta1.Instance:
		// if there is no scheduled plan, then we're done
		if obj.Spec.PlanExecution.PlanName == "" {
			return nil
		}

		return fmt.Errorf("instance %s/%s active plan is in state %v", obj.Namespace, obj.Name, obj.Spec.PlanExecution.Status)

	case *corev1.Pod:
		if obj.Status.Phase == corev1.PodRunning {
			return nil
		}
		return fmt.Errorf("pod %q is not running yet: %s", obj.Name, obj.Status.Phase)

	// unless we build logic for what a healthy object is, assume it's healthy when created.
	default:
		log.Printf("HealthUtil: Unknown type %s is marked healthy by default", reflect.TypeOf(obj))
		return nil
	}
}
