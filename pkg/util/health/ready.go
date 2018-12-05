package health

import (
	"context"
	"fmt"
	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
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
		if ss.Status.ReadyReplicas == ss.Status.Replicas {
			fmt.Printf("Statefulset %v is marked healthy\n", ss.Name)
			return nil
		}
		fmt.Printf("Statefulset %v is NOT healthy.  Not enough ready replicas: %v/%v\n", ss.Name, ss.Status.ReadyReplicas, ss.Status.Replicas)
		return fmt.Errorf("Ready Replicas (%v) does not equal Requested Replicas (%v)", ss.Status.ReadyReplicas, ss.Status.Replicas)
	case *appsv1.Deployment:
		d := obj.(*appsv1.Deployment)
		fmt.Printf("Deployment %v is marked healthy\n", d.Name)
		return nil
	case *batchv1.Job:
		job := obj.(*batchv1.Job)
		// job.Status.

		if job.Status.Succeeded == int32(1) {
			//done!

			fmt.Printf("Job %v is marked healthy\n", job.Name)
			return nil
		}
		return fmt.Errorf("Job %v still running or failed", job.Name)
	case *maestrov1alpha1.Instance:
		i := obj.(*maestrov1alpha1.Instance)
		//Instances are healthy when their Active Plan has succeeded
		plan := &maestrov1alpha1.PlanExecution{}
		c.Get(context.TODO(), client.ObjectKey{
			Name:      i.Status.ActivePlan.Name,
			Namespace: i.Status.ActivePlan.Namespace,
		}, plan)
		fmt.Printf("Instance %v is in state %v\n", i.Name, plan.Status.State)
		if plan.Status.State == maestrov1alpha1.PhaseStateComplete {
			return nil
		}
		return fmt.Errorf("instance's active plan is in state %v", plan.Status.State)

	//unless we build logic for what a healthy object is, assume its healthy when created
	default:
		fmt.Printf("Unkonwn type is marked healthy by default\n")
		return nil
	}
}

func IsStepHealthy(c client.Client, step maestrov1alpha1.StepStatus) bool {
	for _, obj := range step.Objects {
		if e := IsHealthy(c, obj); e != nil {
			fmt.Printf("Step %v is not healthy\n", step.Name)
			return false
		}
	}
	return true
}

func IsPhaseHealthy(phase maestrov1alpha1.PhaseStatus) bool {
	for _, step := range phase.Steps {
		if step.State != maestrov1alpha1.PhaseStateComplete {
			fmt.Printf("Phase %v is not healthy b/c step %v is not healthy\n", phase.Name, step.Name)
			return false
		}
	}
	fmt.Printf("Phase %v is healthy\n", phase.Name)
	return true
}

func IsPlanHealthy(plan maestrov1alpha1.PlanExecutionStatus) bool {
	for _, phase := range plan.Phases {
		if !IsPhaseHealthy(phase) {
			return false
		}
	}
	return true
}
