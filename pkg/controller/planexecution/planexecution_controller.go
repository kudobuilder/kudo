/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package planexecution

import (
	"context"
	"fmt"
	"log"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/health"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const basePath = "/kustomize"

// Add creates a new PlanExecution Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	log.Printf("PlanExecutionController: Registering planexecution controller.")
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePlanExecution{Client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetRecorder("planexecution-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("planexecution-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	// Watch for changes to PlanExecution,
	err = c.Watch(&source.Kind{Type: &kudov1alpha1.PlanExecution{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch Deployments and trigger Reconciles for objects mapped from the Deployment in the event
	for _, e := range [...]runtime.Object{
		&appsv1.StatefulSet{},
		&appsv1.Deployment{},
		&batchv1.Job{},
		&kudov1alpha1.Instance{},
	} {
		err = c.Watch(
			&source.Kind{Type: e},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: engine.ReconcileRequestsMapperFunc(mgr),
			},
			engine.PlanEventPredicateFunc())
		if err != nil {
			return err
		}
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcilePlanExecution{}

// ReconcilePlanExecution reconciles a PlanExecution object
type ReconcilePlanExecution struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a PlanExecution object and makes changes based on the state read
// and what is in the PlanExecution.Spec
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.k8s.io,resources=planexecutions;instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events;configmaps,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets;poddisruptionbudgets.policy,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcilePlanExecution) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the PlanExecution instance
	planExecution := &kudov1alpha1.PlanExecution{}
	err := r.Get(context.TODO(), request.NamespacedName, planExecution)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("PlanExecutionController: Error finding planexecution \"%v\": %v", request.Name, err)
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{Requeue: true}, err
	}

	//Get Instance Object
	instance := &kudov1alpha1.Instance{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      planExecution.Spec.Instance.Name,
			Namespace: planExecution.Spec.Instance.Namespace,
		},
		instance)
	if err != nil {
		// Can't find the instance. Update status.
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		r.recorder.Event(planExecution, "Warning", "InvalidInstance", fmt.Sprintf("Could not find required instance (%v)", planExecution.Spec.Instance.Name))
		log.Printf("PlanExecutionController: Error getting Instance %v in %v: %v",
			planExecution.Spec.Instance.Name,
			planExecution.Spec.Instance.Namespace,
			err)
		return reconcile.Result{}, err
	}

	//See if this has already been processed
	if planExecution.Status.State == kudov1alpha1.PhaseStateComplete {
		log.Printf("PlanExecutionController: PlanExecution \"%v\" has already run to completion, not processing.", planExecution.Name)
		return reconcile.Result{}, nil
	}

	//Before returning from this function, update the status
	defer r.Update(context.Background(), planExecution)

	//Check for Suspend set.
	if planExecution.Spec.Suspend != nil && *planExecution.Spec.Suspend {
		planExecution.Status.State = kudov1alpha1.PhaseStateSuspend
		r.recorder.Event(instance, "Normal", "PlanSuspend", fmt.Sprintf("PlanExecution %v suspended", planExecution.Name))
		return reconcile.Result{}, err
	}

	//need to add ownerReference as the Instance
	instance.Status.ActivePlan = corev1.ObjectReference{
		Name:       planExecution.Name,
		Kind:       planExecution.Kind,
		Namespace:  planExecution.Namespace,
		APIVersion: planExecution.APIVersion,
		UID:        planExecution.UID,
	}
	err = r.Update(context.TODO(), instance)
	if err != nil {
		r.recorder.Event(planExecution, "Warning", "UpdateError", fmt.Sprintf("Could not update the ActivePlan for (%v): %v", planExecution.Spec.Instance.Name, err))
		log.Printf("PlanExecutionController: Update of instance with ActivePlan errored: %v", err)
	}

	//Get associated FrameworkVersion
	frameworkVersion := &kudov1alpha1.FrameworkVersion{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.FrameworkVersion.Name,
			Namespace: instance.Spec.FrameworkVersion.Namespace,
		},
		frameworkVersion)
	if err != nil {
		//Can't find the FrameworkVersion. Update status
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		r.recorder.Event(planExecution, "Warning", "InvalidFrameworkVersion", fmt.Sprintf("Could not find FrameworkVersion %v", instance.Spec.FrameworkVersion.Name))
		log.Printf("PlanExecutionController: Error getting FrameworkVersion %v in %v: %v",
			instance.Spec.FrameworkVersion.Name,
			instance.Spec.FrameworkVersion.Namespace,
			err)
		return reconcile.Result{}, err
	}

	configs, err := engine.ParseConfig(planExecution, instance, frameworkVersion, r.recorder)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Get Plan from FrameworkVersion:
	//Right now must match exactly.  In the future have defaults/backups:
	// e.g. if no "upgrade", call "update"
	// if no "update" call "deploy"
	// When we have this we'll have to keep the active plan in the status since
	// that might not match the "requested" plan.
	executedPlan, ok := frameworkVersion.Spec.Plans[planExecution.Spec.PlanName]
	if !ok {
		r.recorder.Event(planExecution, "Warning", "InvalidPlan", fmt.Sprintf("Could not find required plan (%v)", planExecution.Spec.PlanName))
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		err = fmt.Errorf("could not find required plan (%v)", planExecution.Spec.PlanName)
		return reconcile.Result{}, err
	}

	planExecution.Status.Name = planExecution.Spec.PlanName
	planExecution.Status.Strategy = executedPlan.Strategy

	err = engine.PopulatePlanExecutionPhases(basePath, &executedPlan, planExecution, instance, frameworkVersion, configs, r.recorder)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = engine.RunPhases(&executedPlan, planExecution, instance, r.Client, r.scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	if health.IsPlanHealthy(planExecution.Status) {
		r.recorder.Event(planExecution, "Normal", "PhaseStateComplete", fmt.Sprintf("Instances healthy, phase marked as COMPLETE"))
		r.recorder.Event(instance, "Normal", "PlanComplete", fmt.Sprintf("PlanExecution %v completed", planExecution.Name))
		planExecution.Status.State = kudov1alpha1.PhaseStateComplete
	} else {
		planExecution.Status.State = kudov1alpha1.PhaseStateInProgress
	}

	instance.Status.Status = planExecution.Status.State
	err = r.Client.Update(context.TODO(), instance)
	if err != nil {
		log.Printf("Error updating instance status to %v: %v\n", instance.Status.Status, err)
	}

	return reconcile.Result{}, nil
}

//Cleanup modifies objects on the cluster to allow for the provided obj to get CreateOrApply.  Currently
//only needs to clean up Jobs that get run from multiple PlanExecutions
func (r *ReconcilePlanExecution) Cleanup(obj runtime.Object) error {

	switch obj := obj.(type) {
	case *batchv1.Job:
		//We need to see if there's a current job on the system that matches this exactly (with labels)
		log.Printf("PlanExecutionController.Cleanup: *batchv1.Job %v", obj.Name)

		present := &batchv1.Job{}
		key, _ := client.ObjectKeyFromObject(obj)
		err := r.Get(context.TODO(), key, present)
		if errors.IsNotFound(err) {
			//this is fine, its good to go
			log.Printf("PlanExecutionController: Could not find job \"%v\" in cluster. Good to make a new one.", key)
			return nil
		}
		if err != nil {
			//Something else happened
			return err
		}
		//see if the job in the cluster has the same labels as the one we're looking to add.
		for k, v := range obj.Labels {
			if v != present.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, present.Labels[k])
				err = r.Delete(context.TODO(), present)
				return err
			}
		}
		for k, v := range present.Labels {
			if v != obj.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, obj.Labels[k])
				err = r.Delete(context.TODO(), present)
				return err
			}
		}
		return nil
	}

	return nil
}
