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
	"strconv"
	"strings"

	"github.com/kudobuilder/kudo/pkg/util/kudo"

	apijson "k8s.io/apimachinery/pkg/util/json"

	kudoengine "github.com/kudobuilder/kudo/pkg/engine"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/util/health"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	ctx := context.TODO()
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
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			owners := a.Meta.GetOwnerReferences()
			requests := make([]reconcile.Request, 0)
			for _, owner := range owners {
				// if owner is an instance, we also want to queue up the PlanExecution
				// in the Status section
				inst := &kudov1alpha1.Instance{}
				err = mgr.GetClient().Get(ctx, client.ObjectKey{
					Name:      owner.Name,
					Namespace: a.Meta.GetNamespace(),
				}, inst)

				if err != nil {
					log.Printf("PlanExecutionController: Error getting instance object: %v", err)
				} else {
					log.Printf("PlanExecutionController: Adding \"%v\" to reconcile", inst.Status.ActivePlan)
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
			ToRequests: mapFn,
		},
		p)
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		p)
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		p)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &kudov1alpha1.Instance{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
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
// +kubebuilder:rbac:groups=kudo.k8s.io,resources=planexecutions;instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events;configmaps,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets;poddisruptionbudgets.policy,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcilePlanExecution) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	// Fetch the PlanExecution instance
	planExecution := &kudov1alpha1.PlanExecution{}
	err := r.Get(ctx, request.NamespacedName, planExecution)
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
	err = r.Get(ctx,
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
		return reconcile.Result{}, err
	}

	// Check for Suspend set.
	if planExecution.Spec.Suspend != nil && *planExecution.Spec.Suspend {
		planExecution.Status.State = kudov1alpha1.PhaseStateSuspend
		err = r.Update(ctx, planExecution)
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
	err = r.Get(ctx,
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

	executedPlan, ok := operatorVersion.Spec.Plans[planExecution.Spec.PlanName]
	if !ok {
		r.recorder.Event(planExecution, "Warning", "InvalidPlan", fmt.Sprintf("Could not find required plan (%v)", planExecution.Spec.PlanName))
		err = fmt.Errorf("could not find required plan (%v)", planExecution.Spec.PlanName)
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		return reconcile.Result{}, err
	}

	err = populatePlanExecution(executedPlan, planExecution, instance, operatorVersion, r.recorder)
	if err != nil {
		_, fatalError := err.(*fatalError)
		if fatalError {
			// do not retry
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// now we're actually starting with the execution of plan/phase/step
	for i, phase := range planExecution.Status.Phases {
		// If we still want to execute phases in this plan check if phase is healthy
		for j, s := range phase.Steps {
			planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateComplete

			for _, obj := range s.Objects {
				if s.Delete {
					log.Printf("PlanExecutionController: Step \"%v\" was marked to delete object %+v", s.Name, obj)
					err = r.Client.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
					if errors.IsNotFound(err) || err == nil {
						// This is okay
						log.Printf("PlanExecutionController: Object was already deleted or did not exist in step \"%v\"", s.Name)
					} else {
						log.Printf("PlanExecutionController: Error deleting object in step \"%v\": %v", s.Name, err)
						planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
						planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError
						return reconcile.Result{}, err
					}
					continue
				}

				// Make sure this object is applied to the cluster. Get back the instance from the
				// cluster so we can see if it's healthy or not
				if err = controllerutil.SetControllerReference(instance, obj.(metav1.Object), r.scheme); err != nil {
					return reconcile.Result{}, err
				}

				// Some objects don't update well. We capture the logic here to see if we need to
				// cleanup the current object
				err = r.Cleanup(obj)
				if err != nil {
					log.Printf("PlanExecutionController: Cleanup failed: %v", err)
				}

				//See if its present
				rawObj, _ := apijson.Marshal(obj)
				key, _ := client.ObjectKeyFromObject(obj)
				err := r.Client.Get(ctx, key, obj)
				if err == nil {
					log.Printf("PlanExecutionController: Object %v already exists for instance %v, going to apply patch", key, instance.Name)
					//update
					log.Printf("Going to apply patch\n%+v\n\n to object\n%s\n", string(rawObj), prettyPrint(obj))
					if err != nil {
						log.Printf("Error getting patch between truth and obj: %v\n", err)
					} else {
						err = r.Client.Patch(ctx, obj, client.ConstantPatch(types.StrategicMergePatchType, rawObj))
						if err != nil {
							// Right now applying a Strategic Merge Patch to custom resources does not work. There is
							// certain metadata needed, which when missing, leads to an invalid Content-Type Header and
							// causes the request to fail.
							// ( see https://github.com/kubernetes-sigs/kustomize/issues/742#issuecomment-458650435 )
							//
							// We temporarily solve this by checking for the specific error when a SMP is applied to
							// custom resources and handle it by defaulting to a Merge Patch.
							//
							// The error message for which we check is:
							// 		the body of the request was in an unknown format - accepted media types include:
							//			application/json-patch+json, application/merge-patch+json
							//
							// 		Reason: "UnsupportedMediaType" Code: 415
							if errors.IsUnsupportedMediaType(err) {
								err = r.Client.Patch(ctx, obj, client.ConstantPatch(types.MergePatchType, rawObj))
								if err != nil {
									log.Printf("PlanExecutionController: Error when applying merge patch to object %v for instance %v: %v", key, instance.Name, err)
								}
							} else {
								log.Printf("PlanExecutionController: Error when applying StrategicMergePatch to object %v for instance %v: %v", key, instance.Name, err)
							}
						}
					}
				} else {
					//create
					log.Printf("PlanExecutionController: Object %v does not exist, going to create new object for instance %v", key, instance.Name)
					err = r.Client.Create(ctx, obj)
					if err != nil {
						log.Printf("PlanExecutionController: Error when creating object %v: %v", s.Name, err)
						planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
						planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError

						return reconcile.Result{}, err
					}
				}

				err = health.IsHealthy(r.Client, obj)
				if err != nil {
					log.Printf("PlanExecutionController: Obj is NOT healthy: %s", prettyPrint(obj))
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateInProgress
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateInProgress
				}
			}
			log.Printf("PlanExecutionController: Phase \"%v\" has strategy %v", phase.Name, phase.Strategy)
			if phase.Strategy == kudov1alpha1.Serial {
				// We need to skip the rest of the steps if this step is unhealthy
				log.Printf("PlanExecutionController: Phase \"%v\" marked as serial", phase.Name)
				if planExecution.Status.Phases[i].Steps[j].State != kudov1alpha1.PhaseStateComplete {
					log.Printf("PlanExecutionController: Step \"%v\" isn't complete, skipping rest of steps in phase until it is", planExecution.Status.Phases[i].Steps[j].Name)
					break
				} else {
					log.Printf("PlanExecutionController: Step \"%v\" is healthy, so I can continue on", planExecution.Status.Phases[i].Steps[j].Name)
				}
			}

			log.Printf("PlanExecutionController: Looked at step \"%v\"", s.Name)
		}
		if health.IsPhaseHealthy(planExecution.Status.Phases[i]) {
			log.Printf("PlanExecutionController: Phase \"%v\" marked as healthy", phase.Name)
			planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateComplete
			continue
		}

		// This phase isn't quite ready yet. Let's see what needs to be done
		planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateInProgress

		// Don't keep going to other plans if we're flagged to perform the phases in serial
		if executedPlan.Strategy == kudov1alpha1.Serial {
			log.Printf("PlanExecutionController: Phase \"%v\" not healthy, and plan marked as serial, so breaking.", phase.Name)
			break
		}
		log.Printf("PlanExecutionController: Looked at phase \"%v\"", phase.Name)
	}

	if health.IsPlanHealthy(planExecution.Status) {
		r.recorder.Event(planExecution, "Normal", "PhaseStateComplete", fmt.Sprintf("Instances healthy, phase marked as COMPLETE"))
		r.recorder.Event(instance, "Normal", "PlanComplete", fmt.Sprintf("PlanExecution %v completed", planExecution.Name))
		planExecution.Status.State = kudov1alpha1.PhaseStateComplete
	} else {
		planExecution.Status.State = kudov1alpha1.PhaseStateInProgress
	}

	instance.Status.Status = planExecution.Status.State
	err = r.Client.Update(ctx, instance)
	if err != nil {
		log.Printf("PlanExecutionController: Error updating instance status to %v: %v\n", instance.Status.Status, err)
	}

	// Before returning from this function, update the status
	err = r.Update(ctx, planExecution)
	if err != nil {
		log.Printf("PlanExecutionController: Error updating planexecution: %v: err:%v", planExecution, err)
	}

	return reconcile.Result{}, nil
}

// fatalError is representing type of error that is non-recoverable (like bug in the template preventing rendering)
// we should not retry these errors
type fatalError struct {
	err error
}

func (e fatalError) Error() string {
	return fmt.Sprintf("Fatal error: %v", e.err)
}

// populatePlanExecution reads content of the Plan defined in operator version and populates PlanExecution with data from rendered templates
func populatePlanExecution(activePlan kudov1alpha1.Plan, planExecution *kudov1alpha1.PlanExecution, instance *kudov1alpha1.Instance, operatorVersion *kudov1alpha1.OperatorVersion, recorder record.EventRecorder) error {
	// Load parameters:

	// Create config map to hold all parameters for instantiation
	configs := make(map[string]interface{})

	// Default parameters from instance metadata
	configs["OperatorName"] = operatorVersion.Spec.Operator.Name
	configs["Name"] = instance.Name
	configs["Namespace"] = instance.Namespace

	params, err := getParameters(instance, operatorVersion)
	if err != nil {
		log.Printf("PlanExecutionController: %v", err)
		recorder.Event(planExecution, "Warning", "MissingParameter", err.Error())
		return err
	}

	configs["Params"] = params

	planExecution.Status.Name = planExecution.Spec.PlanName
	planExecution.Status.Strategy = activePlan.Strategy

	planExecution.Status.Phases = make([]kudov1alpha1.PhaseStatus, len(activePlan.Phases))
	for i, phase := range activePlan.Phases {
		// Populate the Status elements in instance.
		planExecution.Status.Phases[i].Name = phase.Name
		planExecution.Status.Phases[i].Strategy = phase.Strategy
		planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStatePending
		planExecution.Status.Phases[i].Steps = make([]kudov1alpha1.StepStatus, len(phase.Steps))
		for j, step := range phase.Steps {
			// Fetch OperatorVersion:
			//
			//   - Get the task name from the step
			//   - Get the task definition from the OV
			//   - Create the kustomize templates
			//   - Apply
			configs["PlanName"] = planExecution.Spec.PlanName
			configs["PhaseName"] = phase.Name
			configs["StepName"] = step.Name
			configs["StepNumber"] = strconv.FormatInt(int64(j), 10)

			var objs []runtime.Object

			engine := kudoengine.New()
			for _, t := range step.Tasks {
				// resolve task
				if taskSpec, ok := operatorVersion.Spec.Tasks[t]; ok {
					resources := make(map[string]string)

					for _, res := range taskSpec.Resources {
						if resource, ok := operatorVersion.Spec.Templates[res]; ok {
							templatedYaml, err := engine.Render(resource, configs)
							if err != nil {
								recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error expanding template: %v", err))
								log.Printf("PlanExecutionController: Error expanding template: %v", err)
								planExecution.Status.State = kudov1alpha1.PhaseStateError
								planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
								// returning error = nil so that we don't retry since this is non-recoverable
								return fatalError{err: err}
							}
							resources[res] = templatedYaml

						} else {
							recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding resource named %v for operator version %v", res, operatorVersion.Name))
							log.Printf("PlanExecutionController: Error finding resource named %v for operator version %v", res, operatorVersion.Name)
							return fatalError{err: fmt.Errorf("PlanExecutionController: Error finding resource named %v for operator version %v", res, operatorVersion.Name)}
						}
					}

					objsToAdd, err := applyConventionsToTemplates(resources, metadata{
						InstanceName:    instance.Name,
						Namespace:       instance.Namespace,
						OperatorName:    operatorVersion.Spec.Operator.Name,
						OperatorVersion: operatorVersion.Spec.Version,
						PlanExecution:   planExecution.Name,
						PlanName:        planExecution.Spec.PlanName,
						PhaseName:       phase.Name,
						StepName:        step.Name,
					})

					if err != nil {
						recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err))
						log.Printf("PlanExecutionController: Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err)
						return err
					}
					objs = append(objs, objsToAdd...)
				} else {
					recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding task named %s for operator version %s", taskSpec, operatorVersion.Name))
					log.Printf("PlanExecutionController: Error finding task named %s for operator version %s", taskSpec, operatorVersion.Name)
					return fatalError{err: err}
				}
			}

			planExecution.Status.Phases[i].Steps[j].Name = step.Name
			planExecution.Status.Phases[i].Steps[j].Objects = objs
			planExecution.Status.Phases[i].Steps[j].Delete = step.Delete
			log.Printf("PlanExecutionController: Phase \"%v\" Step \"%v\" of instance '%v' has %v object(s)", phase.Name, step.Name, instance.Name, len(objs))
		}
	}

	return nil
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
	ctx := context.TODO()
	switch obj := obj.(type) {
	case *batchv1.Job:
		// We need to see if there's a current job on the system that matches this exactly (with labels)
		log.Printf("PlanExecutionController.Cleanup: *batchv1.Job %v", obj.Name)

		present := &batchv1.Job{}
		key, _ := client.ObjectKeyFromObject(obj)
		err := r.Get(ctx, key, present)
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
				err = r.Delete(ctx, present)
				return err
			}
		}
		for k, v := range present.Labels {
			if v != obj.Labels[k] {
				// Need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, obj.Labels[k])
				err = r.Delete(ctx, present)
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
