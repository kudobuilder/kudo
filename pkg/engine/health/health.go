package health

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/polymorphichelpers"

	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

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
		log.Printf("Deployment %v is marked healthy\n", obj.Name)
		return nil
	case *batchv1.Job:

		if obj.Status.Succeeded == int32(1) {
			// Done!
			log.Printf("HealthUtil: Job \"%v\" is marked healthy", obj.Name)
			return nil
		}
		return fmt.Errorf("job \"%v\" still running or failed", obj.Name)
	case *kudov1beta1.Instance:
		log.Printf("HealthUtil: Instance %v is in state %v", obj.Name, obj.Status.AggregatedStatus.Status)

		if obj.Status.AggregatedStatus.Status.IsFinished() {
			return nil
		}
		return fmt.Errorf("instance's active plan is in state %v", obj.Status.AggregatedStatus.Status)

	// unless we build logic for what a healthy object is, assume it's healthy when created.
	default:
		log.Printf("HealthUtil: Unknown type %s is marked healthy by default", reflect.TypeOf(obj))
		return nil
	}
}
