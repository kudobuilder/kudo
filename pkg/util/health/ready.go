package health

import (
	"context"
	"fmt"
	"log"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//IsHealthy returns whether an object is healthy.  Must be implemented for each type
func IsHealthy(c client.Client, obj runtime.Object) error {

	switch obj.(type) {
	case *appsv1.StatefulSet:
		ss := obj.(*appsv1.StatefulSet)
		if ss.Spec.Replicas == nil {
			return fmt.Errorf("replicas not set, so can't be healthy")
		}
		if ss.Status.ReadyReplicas == *ss.Spec.Replicas {
			log.Printf("Statefulset %v is marked healthy\n", ss.Name)
			return nil
		}
		log.Printf("HealthUtil: Statefulset %v is NOT healthy. Not enough ready replicas: %v/%v", ss.Name, ss.Status.ReadyReplicas, ss.Status.Replicas)
		return fmt.Errorf("ready replicas (%v) does not equal requested replicas (%v)", ss.Status.ReadyReplicas, ss.Status.Replicas)
	case *appsv1.Deployment:
		d := obj.(*appsv1.Deployment)
		if d.Spec.Replicas != nil && d.Status.ReadyReplicas == *d.Spec.Replicas {
			log.Printf("HealthUtil: Deployment %v is marked healthy", d.Name)
			return nil
		}
		log.Printf("HealthUtil: Deployment %v is NOT healthy. Not enough ready replicas: %v/%v", d.Name, d.Status.ReadyReplicas, *d.Spec.Replicas)
		return fmt.Errorf("ready replicas (%v) does not equal requested replicas (%v)", d.Status.ReadyReplicas, *d.Spec.Replicas)
	case *batchv1.Job:
		job := obj.(*batchv1.Job)
		// job.Status.

		if job.Status.Succeeded == int32(1) {
			//done!

			log.Printf("HealthUtil: Job \"%v\" is marked healthy", job.Name)
			return nil
		}
		return fmt.Errorf("job \"%v\" still running or failed", job.Name)
	case *kudov1alpha1.Instance:
		i := obj.(*kudov1alpha1.Instance)
		//Instances are healthy when their Active Plan has succeeded
		plan := &kudov1alpha1.PlanExecution{}
		err := c.Get(context.TODO(), client.ObjectKey{
			Name:      i.Status.ActivePlan.Name,
			Namespace: i.Status.ActivePlan.Namespace,
		}, plan)
		if err != nil {
			log.Printf("Error getting PlaneExecution %v/%v: %v\n", i.Status.ActivePlan.Name, i.Status.ActivePlan.Namespace, err)
			return fmt.Errorf("instance active plan not found: %v", err)
		}
		log.Printf("HealthUtil: Instance %v is in state %v", i.Name, plan.Status.State)

		if plan.Status.State == kudov1alpha1.PhaseStateComplete {
			return nil
		}
		return fmt.Errorf("instance's active plan is in state %v", plan.Status.State)

	//unless we build logic for what a healthy object is, assume its healthy when created
	default:
		log.Printf("HealthUtil: Unknown type is marked healthy by default")
		return nil
	}
}

//IsStepHealthy returns whether each object in the given step is healthy
func IsStepHealthy(c client.Client, step kudov1alpha1.StepStatus) bool {
	for _, obj := range step.Objects {
		if e := IsHealthy(c, obj); e != nil {
			log.Printf("HealthUtil: Step %v is not healthy", step.Name)
			return false
		}
	}
	return true
}

//IsPhaseHealthy returns whether each step in the phase is healthy.  See IsStepHealthy for step health
func IsPhaseHealthy(phase kudov1alpha1.PhaseStatus) bool {
	for _, step := range phase.Steps {
		if step.State != kudov1alpha1.PhaseStateComplete {
			log.Printf("HealthUtil: Phase %v is not healthy b/c step %v is not healthy", phase.Name, step.Name)
			return false
		}
	}
	log.Printf("HealthUtil: Phase %v is healthy", phase.Name)
	return true
}

//IsPlanHealthy returns whether each Phase in the plan is healthy.  See IsPhaseHealthy for phase health
func IsPlanHealthy(plan kudov1alpha1.PlanExecutionStatus) bool {
	for _, phase := range plan.Phases {
		if !IsPhaseHealthy(phase) {
			return false
		}
	}
	return true
}
