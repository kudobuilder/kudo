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
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/kudobuilder/kudo/pkg/util/kudo"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new PlanExecution Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	log.Printf("PlanExecutionController: Registering planexecution controller.")

	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePlanExecution{Client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetEventRecorderFor("planexecution-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("planexecution-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for Deployments, Jobs and StatefulSets
	//
	// Define a mapping from the object in the event to one or more objects to
	// Reconcile. Specifically this calls for a reconciliation of any owned
	// objects.
	mapToOwningInstanceActivePlan := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			owners := a.Meta.GetOwnerReferences()
			requests := make([]reconcile.Request, 0)
			for _, owner := range owners {
				// if owner is an instance, we also want to queue up the PlanExecution
				// in the Status section
				inst := &kudov1alpha1.Instance{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{
					Name:      owner.Name,
					Namespace: a.Meta.GetNamespace(),
				}, inst)

				if err != nil {
					log.Printf("PlanExecutionController: Error getting instance object: %v", err)
				} else {
					log.Printf("PlanExecutionController: Adding \"%v\" to reconcile", inst.Status.ActivePlan.Name)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      inst.Status.ActivePlan.Name,
							Namespace: inst.Status.ActivePlan.Namespace,
						},
					})
				}
			}
			return requests
		})

	// 'UpdateFunc' and 'CreateFunc' are used to judge if a event about the object is what
	// we want. If return true, the event will be processed by the reconciler.
	//
	// PlanExecutions should be mostly immutable.
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.Printf("PlanExecutionController: Received update event for an instance named: %v", e.MetaNew.GetName())
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf("PlanExecutionController: Received create event for an instance named: %v", e.Meta.GetName())
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// TODO: send event for Instance that plan was deleted
			log.Printf("PlanExecutionController: Received delete event for an instance named: %v", e.Meta.GetName())
			return true
		},
	}

	// Watch for changes to PlanExecution
	err = c.Watch(&source.Kind{Type: &kudov1alpha1.PlanExecution{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch Deployments and trigger Reconciles for objects mapped from the Deployment in the event
	err = c.Watch(
		&source.Kind{Type: &appsv1.StatefulSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapToOwningInstanceActivePlan,
		},
		p)
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapToOwningInstanceActivePlan,
		},
		p)
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapToOwningInstanceActivePlan,
		},
		p)
	if err != nil {
		return err
	}

	// for instances we're interested in updates of instances owned by some planexecution (instance was created as part of PE)
	// but also root instances of an operator that might have been updated with new activeplan
	err = c.Watch(
		&source.Kind{Type: &kudov1alpha1.Instance{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(
				func(a handler.MapObject) []reconcile.Request {
					requests := mapToOwningInstanceActivePlan(a)
					if len(requests) == 0 {
						inst := &kudov1alpha1.Instance{}
						err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{
							Name:      a.Meta.GetName(),
							Namespace: a.Meta.GetNamespace(),
						}, inst)

						if err == nil {
							// for every updated/added instance also trigger reconcile for its active plan
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Name:      inst.Status.ActivePlan.Name,
									Namespace: inst.Status.ActivePlan.Namespace,
								},
							})
						} else {
							log.Printf("PlanExecutionController: received event from Instance %s/%s but instance of that name does not exist", a.Meta.GetNamespace(), a.Meta.GetName())
						}
					}
					return requests
				}),
		},
		p)
	if err != nil {
		return err
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
//
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.dev,resources=planexecutions;instances,verbs=get;list;watch;create;update;patch;delete
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
		return reconcile.Result{}, err
	}

	instance := &kudov1alpha1.Instance{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      planExecution.Spec.Instance.Name,
			Namespace: planExecution.Spec.Instance.Namespace,
		},
		instance)
	if err != nil {
		// Can't find the instance.
		r.recorder.Event(planExecution, "Warning", "InvalidInstance", fmt.Sprintf("Could not find required instance (%v)", planExecution.Spec.Instance.Name))
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		log.Printf("PlanExecutionController: Error getting Instance %v in %v: %v",
			planExecution.Spec.Instance.Name,
			planExecution.Spec.Instance.Namespace,
			err)

		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if instance.Status.ActivePlan.Name != planExecution.Name || instance.Status.ActivePlan.Namespace != planExecution.Namespace {
		// this can happen for newly created PlanExecution where ActivePlan was not yet set to point to this instance
		// this will get retried thanks to a watch set up for instance updates
		log.Printf("instance %s does not have ActivePlan pointing to PlanExecution %s, %s. Instead %s, %s", instance.Name, planExecution.Name, planExecution.Namespace, instance.Status.ActivePlan.Name, instance.Status.ActivePlan.Namespace)
		return reconcile.Result{}, nil
	}

	// Check for Suspend set.
	if planExecution.Spec.Suspend != nil && *planExecution.Spec.Suspend {
		planExecution.Status.State = kudov1alpha1.PhaseStateSuspend
		err = r.Update(context.TODO(), planExecution)
		r.recorder.Event(instance, "Normal", "PlanSuspend", fmt.Sprintf("PlanExecution %v suspended", planExecution.Name))
		return reconcile.Result{}, err
	}

	// See if this has already been processed
	if planExecution.Status.State == kudov1alpha1.PhaseStateComplete {
		log.Printf("PlanExecutionController: PlanExecution \"%v\" has already run to completion, not processing.", planExecution.Name)
		return reconcile.Result{}, nil
	}

	// Get associated OperatorVersion
	operatorVersion := &kudov1alpha1.OperatorVersion{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.OperatorVersion.Name,
			Namespace: instance.GetOperatorVersionNamespace(),
		},
		operatorVersion)
	if err != nil {
		// Can't find the OperatorVersion.
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		r.recorder.Event(planExecution, "Warning", "InvalidOperatorVersion", fmt.Sprintf("Could not find OperatorVersion %v", instance.Spec.OperatorVersion.Name))
		log.Printf("PlanExecutionController: Error getting OperatorVersion %v in %v: %v",
			instance.Spec.OperatorVersion.Name,
			instance.GetOperatorVersionNamespace(),
			err)
		return reconcile.Result{}, err
	}

	params, err := getParameters(instance, operatorVersion)
	if err != nil {
		log.Printf("PlanExecutionController: %v", err)
		r.recorder.Event(planExecution, "Warning", "MissingParameter", err.Error())
		return reconcile.Result{}, nil // do not retry this error
	}

	executedPlan, ok := operatorVersion.Spec.Plans[planExecution.Spec.PlanName]
	if !ok {
		r.recorder.Event(planExecution, "Warning", "InvalidPlan", fmt.Sprintf("Could not find required plan (%v)", planExecution.Spec.PlanName))
		err = fmt.Errorf("could not find required plan (%v)", planExecution.Spec.PlanName)
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		return reconcile.Result{}, err
	}

	planExecution = planExecution.DeepCopy()
	activePlan := &activePlan{
		Name: planExecution.Spec.PlanName,
		Spec: &executedPlan,
		State: &planExecution.Status,
		Tasks: operatorVersion.Spec.Tasks,
		Templates: operatorVersion.Spec.Templates,
	}
	initializePlanStatus(&planExecution.Status, activePlan)

	log.Printf("PlanExecutionController: Going to execute plan %s for instance %s", planExecution.Name, instance.Name)
	newState, err := executePlan(activePlan, planExecution.Name, instance, params, operatorVersion, r.Client, r.scheme)
	if newState != nil {
		planExecution.Status = *newState
	}

	log.Printf("PlanExecutionStatus for %s: %s", instance.Name, prettyPrint(planExecution.Status))

	if err != nil {
		log.Printf("PlanExecutionController: error when executing plan for instance %s: %v", instance.Name, err)

		err = r.Client.Update(context.TODO(), planExecution)
		if err != nil {
			log.Printf("PlanExecutionController: Error when updating planExecution state. %v", err)
			return reconcile.Result{}, err
		}

		if _, ok := err.(*fatalError); ok {
			// do not retry
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err = r.Client.Update(context.TODO(), planExecution)
	if err != nil {
		log.Printf("PlanExecutionController: Error when updating planExecution state. %v", err)
		return reconcile.Result{}, err
	}

	// update instance state
	// TODO this should not be done in this controller, we should address it in another iteration of refactoring
	instance.Status.Status = planExecution.Status.State
	err = r.Client.Update(context.TODO(), instance)
	if err != nil {
		log.Printf("Error updating instance status to %v: %v\n", instance.Status.Status, err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// getPlanState constructs the current plan execution summary by consulting current state of PE CRD and selected plan from OV
func initializePlanStatus(status *kudov1alpha1.PlanExecutionStatus, plan *activePlan) {
	if plan.Name == status.Name && status.State != kudov1alpha1.PhaseStateComplete {
		// nothing to do, plan is already in progress and was populated in previous iteration
		return
	}

	status.State = kudov1alpha1.PhaseStateInProgress
	status.Name = plan.Name
	status.Strategy = plan.Spec.Strategy
	status.Phases = make([]kudov1alpha1.PhaseStatus, 0)

	// plan execution might not yet be initialized, make sure we have all phases and steps covered
	for _, p := range plan.Spec.Phases {
		phaseState := &kudov1alpha1.PhaseStatus{
			Name:     p.Name,
			State:    kudov1alpha1.PhaseStatePending,
			Strategy: p.Strategy,
			Steps:    make([]kudov1alpha1.StepStatus, 0),
		}

		for _, s := range p.Steps {
			stepState := &kudov1alpha1.StepStatus{
				Name:  s.Name,
				State: kudov1alpha1.PhaseStatePending,
			}
			phaseState.Steps = append(phaseState.Steps, *stepState)
		}

		status.Phases = append(status.Phases, *phaseState)
	}
}

// fatalError is representing type of error that is non-recoverable (like bug in the template preventing rendering)
// we should not retry these errors
type fatalError struct {
	err error
}

func (e fatalError) Error() string {
	return fmt.Sprintf("Fatal error: %v", e.err)
}

func getParameters(instance *kudov1alpha1.Instance, operatorVersion *kudov1alpha1.OperatorVersion) (map[string]string, error) {
	params := make(map[string]string)

	for k, v := range instance.Spec.Parameters {
		params[k] = v
	}

	missingRequiredParameters := make([]string, 0)
	// Merge defaults with customizations
	for _, param := range operatorVersion.Spec.Parameters {
		_, ok := params[param.Name]
		if !ok && param.Required && param.Default == nil {
			// instance does not define this parameter and there is no default while the parameter is required -> error
			missingRequiredParameters = append(missingRequiredParameters, param.Name)

		} else if !ok {
			params[param.Name] = kudo.StringValue(param.Default)
		}
	}

	if len(missingRequiredParameters) != 0 {
		return nil, fmt.Errorf("parameters are missing when evaluating template: %s", strings.Join(missingRequiredParameters, ","))
	}

	return params, nil
}

// Cleanup modifies objects on the cluster to allow for the provided obj to get CreateOrApply.
// Currently only needs to clean up Jobs that get run from multiplePlanExecutions
func (r *ReconcilePlanExecution) Cleanup(obj runtime.Object) error {

	switch obj := obj.(type) {
	case *batchv1.Job:
		// We need to see if there's a current job on the system that matches this exactly (with labels)
		log.Printf("PlanExecutionController.Cleanup: *batchv1.Job %v", obj.Name)

		present := &batchv1.Job{}
		key, _ := client.ObjectKeyFromObject(obj)
		err := r.Get(context.TODO(), key, present)
		if errors.IsNotFound(err) {
			// This is fine, its good to go
			log.Printf("PlanExecutionController: Could not find job \"%v\" in cluster. Good to make a new one.", key)
			return nil
		}
		if err != nil {
			// Something else happened
			return err
		}
		// See if the job in the cluster has the same labels as the one we're looking to add.
		for k, v := range obj.Labels {
			if v != present.Labels[k] {
				// Need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, present.Labels[k])
				err = r.Delete(context.TODO(), present)
				return err
			}
		}
		for k, v := range present.Labels {
			if v != obj.Labels[k] {
				// Need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, obj.Labels[k])
				err = r.Delete(context.TODO(), present)
				return err
			}
		}
		return nil
	}

	return nil
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
}
