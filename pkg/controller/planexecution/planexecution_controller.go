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
	"strconv"

	"github.com/kudobuilder/kudo/pkg/util/health"
	"github.com/kudobuilder/kudo/pkg/util/template"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

	"github.com/kudobuilder/kudo/pkg/engine"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
				ToRequests: reconcileRequestsMapperFunc(mgr),
			},
			planEventPredicateFunc())
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
	// Fetch the PlanExecution Instance
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

	// Get Instance Object
	instance := &kudov1alpha1.Instance{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      planExecution.Spec.Instance.Name,
			Namespace: planExecution.Spec.Instance.Namespace,
		},
		instance)
	if err != nil {
		// Can't find the Instance. Update status.
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		r.recorder.Event(planExecution, "Warning", "InvalidInstance", fmt.Sprintf("Could not find required Instance (%v)", planExecution.Spec.Instance.Name))
		log.Printf("PlanExecutionController: Error getting Instance %v in %v: %v",
			planExecution.Spec.Instance.Name,
			planExecution.Spec.Instance.Namespace,
			err)
		return reconcile.Result{}, err
	}

	// See if this has already been processed
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
		log.Printf("PlanExecutionController: Update of Instance with ActivePlan errored: %v", err)
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

	configs, err := engine.ParseConfig(instance, frameworkVersion, func(eventtype, reason, message string) {
		r.recorder.Event(planExecution, eventtype, reason, message)
	})
	if err != nil {
		log.Printf("PlanExecutionController: %v", err)
		return reconcile.Result{}, err
	}

	// Get Plan from FrameworkVersion:
	// Right now must match exactly.  In the future have defaults/backups:
	//  e.g. if no "upgrade", call "update"
	//  if no "update" call "deploy"
	//  When we have this we'll have to keep the active plan in the status since
	//  that might not match the "requested" plan.
	executedPlan, ok := frameworkVersion.Spec.Plans[planExecution.Spec.PlanName]
	if !ok {
		r.recorder.Event(planExecution, "Warning", "InvalidPlan", fmt.Sprintf("Could not find required plan (%v)", planExecution.Spec.PlanName))
		planExecution.Status.State = kudov1alpha1.PhaseStateError
		err = fmt.Errorf("could not find required plan (%v)", planExecution.Spec.PlanName)
		return reconcile.Result{}, err
	}

	planExecution.Status.Name = planExecution.Spec.PlanName
	planExecution.Status.Strategy = executedPlan.Strategy

	err = PopulatePlanExecutionPhases(basePath, &executedPlan, planExecution, instance, frameworkVersion, configs, r.recorder)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = RunPhases(&executedPlan, planExecution, instance, r.Client, r.scheme)
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
		log.Printf("Error updating Instance status to %v: %v\n", instance.Status.Status, err)
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

// Watch for Deployments, Jobs and StatefulSets
// Define a mapping from the object in the event to one or more
// objects to Reconcile.  Specifically this calls for
// a reconciliation of any objects "Owner".
func reconcileRequestsMapperFunc(mgr manager.Manager) handler.ToRequestsFunc {
	return func(a handler.MapObject) []reconcile.Request {
		owners := a.Meta.GetOwnerReferences()
		requests := make([]reconcile.Request, 0)
		for _, owner := range owners {
			// if owner is an Instance, we also want to queue up the
			// PlanExecution in the Status section
			inst := &kudov1alpha1.Instance{}
			err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
				Name:      owner.Name,
				Namespace: a.Meta.GetNamespace(),
			}, inst)

			if err != nil {
				log.Printf("Error getting Instance object: %v", err)
			} else {
				log.Printf("Adding \"%v\" to reconcile", inst.Status.ActivePlan.Name)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      inst.Status.ActivePlan.Name,
						Namespace: inst.Status.ActivePlan.Namespace,
					},
				})
			}
		}
		return requests
	}
}

func planEventPredicateFunc() predicate.Funcs {
	// 'UpdateFunc' and 'CreateFunc' used to judge if a event about the object is
	// what we want. If that is true, the event will be processed by the reconciler.
	// PlanExecutions should be mostly immutable.  Updates should only
	msg := "PlanEventPredicate: Received update event for an Instance named: %v"
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.Printf(msg, e.MetaNew.GetName())
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf(msg, e.Meta.GetName())
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// TODO send event for Instance that plan was deleted
			log.Printf(msg, e.Meta.GetName())
			return true
		},
	}
}

func PopulatePlanExecutionPhases(basePath string, executedPlan *kudov1alpha1.Plan, planExecution *kudov1alpha1.PlanExecution, instance *kudov1alpha1.Instance, frameworkVersion *kudov1alpha1.FrameworkVersion, configs map[string]interface{}, recorder record.EventRecorder) error {
	planExecution.Status.Phases = make([]kudov1alpha1.PhaseStatus, len(executedPlan.Phases))
	var err error
	for i, phase := range executedPlan.Phases {
		planExecution.Status.Phases[i].Name = phase.Name
		planExecution.Status.Phases[i].Strategy = phase.Strategy
		planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStatePending
		planExecution.Status.Phases[i].Steps = make([]kudov1alpha1.StepStatus, len(phase.Steps))
		for j, step := range phase.Steps {
			// fetch FrameworkVersion
			// get the task name from the step
			// get the task definition from the FV
			// create the kustomize templates
			// apply
			configs["PlanName"] = planExecution.Spec.PlanName
			configs["PhaseName"] = phase.Name
			configs["StepName"] = step.Name
			configs["StepNumber"] = strconv.FormatInt(int64(j), 10)

			var objs []runtime.Object
			engine := engine.New()

			for _, t := range step.Tasks {
				// resolve task
				if taskSpec, ok := frameworkVersion.Spec.Tasks[t]; ok {
					var resources []string
					fsys := fs.MakeFakeFS()

					for _, res := range taskSpec.Resources {
						if resource, ok := frameworkVersion.Spec.Templates[res]; ok {
							templatedYaml, err := engine.Render(resource, configs)
							if err != nil {
								recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error expanding template: %v", err))
								log.Printf("PlanExecutionController: Error expanding template: %v", err)
							}
							fsys.WriteFile(fmt.Sprintf("%s/%s", basePath, res), []byte(templatedYaml))
							resources = append(resources, res)

						} else {
							recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding resource named %v for framework version %v", res, frameworkVersion.Name))
							log.Printf("PlanExecutionController: Error finding resource named %v for framework version %v", res, frameworkVersion.Name)
							return err
						}
					}

					kustomization := &ktypes.Kustomization{
						NamePrefix: instance.Name + "-",
						Namespace:  instance.Namespace,
						CommonLabels: map[string]string{
							"heritage":      "kudo",
							"app":           frameworkVersion.Spec.Framework.Name,
							"version":       frameworkVersion.Spec.Version,
							"Instance":      instance.Name,
							"planexecution": planExecution.Name,
							"plan":          planExecution.Spec.PlanName,
							"phase":         phase.Name,
							"step":          step.Name,
						},
						GeneratorOptions: &ktypes.GeneratorOptions{
							DisableNameSuffixHash: true,
						},
						Resources:             resources,
						PatchesStrategicMerge: []patch.StrategicMerge{},
					}

					yamlBytes, err := yaml.Marshal(kustomization)
					if err != nil {
						return err
					}

					fsys.WriteFile(fmt.Sprintf("%s/kustomization.yaml", basePath), yamlBytes)

					ldr, err := loader.NewLoader(basePath, fsys)
					if err != nil {
						return err
					}
					defer ldr.Cleanup()

					rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()))
					kt, err := target.NewKustTarget(ldr, fsys, rf, transformer.NewFactoryImpl())
					if err != nil {
						return err
					}

					allResources, err := kt.MakeCustomizedResMap()
					if err != nil {
						return err
					}

					res, err := allResources.EncodeAsYaml()
					if err != nil {
						return err
					}

					objsToAdd, err := template.ParseKubernetesObjects(string(res))
					if err != nil {
						recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err))
						log.Printf("PlanExecutionController: Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err)
						return err
					}
					objs = append(objs, objsToAdd...)
				} else {
					recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding task named %s for framework version %s", taskSpec, frameworkVersion.Name))
					log.Printf("PlanExecutionController: Error finding task named %s for framework version %s", taskSpec, frameworkVersion.Name)
					return nil
				}
			}

			planExecution.Status.Phases[i].Steps[j].Name = step.Name
			planExecution.Status.Phases[i].Steps[j].Objects = objs
			planExecution.Status.Phases[i].Steps[j].Delete = step.Delete
			log.Printf("PlanExecutionController: Phase \"%v\" Step \"%v\" has %v object(s)", phase.Name, step.Name, len(objs))
		}
	}
	return nil
}

func MutateFn(oldObj runtime.Object) controllerutil.MutateFn {
	return func(newObj runtime.Object) error {
		//TODO Clean this up.  I don't like having to do a switch here
		switch t := newObj.(type) {
		case *appsv1.StatefulSet:
			log.Printf("PlanExecutionController: CreateOrUpdate: StatefulSet %+v", t.Name)

			newSs := newObj.(*appsv1.StatefulSet)
			ss, ok := oldObj.(*appsv1.StatefulSet)
			if !ok {
				return fmt.Errorf("object passed in doesn't match expected StatefulSet type")
			}

			// We need some specialized logic in there.  We can't just copy the Spec since there are other values
			// like spec.updateState, spec.volumeClaimTemplates, etc that are all
			// generated from the object by the k8s controller.  We just want to update things we can change
			newSs.Spec.Replicas = ss.Spec.Replicas

			return nil
		case *appsv1.Deployment:
			newD := newObj.(*appsv1.Deployment)
			d, ok := oldObj.(*appsv1.Deployment)
			if !ok {
				return fmt.Errorf("object passed in doesn't match expected deployment type")
			}
			newD.Spec.Replicas = d.Spec.Replicas
			return nil
		case *v1beta1.Deployment:
			newD := newObj.(*v1beta1.Deployment)
			d, ok := oldObj.(*v1beta1.Deployment)
			if !ok {
				return fmt.Errorf("object passed in doesn't match expected deployment type")
			}
			newD.Spec.Replicas = d.Spec.Replicas
			return nil

		case *batchv1.Job:
			// job := oldObj.(*batchv1.Job)

		case *kudov1alpha1.Instance:
			// i := oldObj.(*kudov1alpha1.Instance)

		//unless we build logic for what a healthy object is, assume its healthy when created
		default:
			log.Print("PlanExecutionController: CreateOrUpdate: Type is not implemented yet")
			return nil
		}

		return nil

	}
}

func Cleanup(c client.Client, obj runtime.Object) error {
	switch obj := obj.(type) {
	case *batchv1.Job:
		//We need to see if there's a current job on the system that matches this exactly (with labels)
		log.Printf("PlanExecutionController.Cleanup: *batchv1.Job %v", obj.Name)

		present := &batchv1.Job{}
		key, _ := client.ObjectKeyFromObject(obj)
		err := c.Get(context.TODO(), key, present)
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
				err = c.Delete(context.TODO(), present)
				return err
			}
		}
		for k, v := range present.Labels {
			if v != obj.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, obj.Labels[k])
				err = c.Delete(context.TODO(), present)
				return err
			}
		}
		return nil
	}

	return nil
}

func RunPhases(executedPlan *kudov1alpha1.Plan, planExecution *kudov1alpha1.PlanExecution, instance *kudov1alpha1.Instance, c client.Client, scheme *runtime.Scheme) error {
	var err error
	for i, phase := range planExecution.Status.Phases {
		//If we still want to execute phases in this plan
		//check if phase is healthy
		for j, s := range phase.Steps {
			planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateComplete

			for _, obj := range s.Objects {
				if s.Delete {
					log.Printf("PlanExecutionController: Step \"%v\" was marked to delete object %+v", s.Name, obj)
					err = c.Delete(context.TODO(), obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
					if errors.IsNotFound(err) || err == nil {
						//This is okay
						log.Printf("PlanExecutionController: Object was already deleted or did not exist in step \"%v\"", s.Name)
					}
					if err != nil {
						log.Printf("PlanExecutionController: Error deleting object in step \"%v\": %v", s.Name, err)
						planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
						planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError
						return err
					}
					continue
				}

				// Make sure this object is applied to the cluster. Get back the Instance from
				// the cluster so we can see if it's healthy or not
				if err = controllerutil.SetControllerReference(instance, obj.(metav1.Object), scheme); err != nil {
					return err
				}

				//Some objects don't update well.  We capture the logic here to see if we need to cleanup the current object
				err = Cleanup(c, obj)
				if err != nil {
					log.Printf("PlanExecutionController: Cleanup failed: %v", err)
				}

				arg := obj.DeepCopyObject()
				result, err := controllerutil.CreateOrUpdate(context.TODO(), c, arg, MutateFn(obj))

				if err != nil {
					log.Printf("PlanExecutionController: Error CreateOrUpdate Object in step \"%v\": %v", s.Name, err)
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError

					return err
				}
				log.Printf("PlanExecutionController: CreateOrUpdate resulted in: %v", result)

				// get the existing object meta
				metaObj := obj.(metav1.Object)

				// retrieve the existing object
				key := client.ObjectKey{
					Name:      metaObj.GetName(),
					Namespace: metaObj.GetNamespace(),
				}

				err = c.Get(context.TODO(), key, obj)

				if err != nil {
					log.Printf("PlanExecutionController: Error getting new object in step \"%v\": %v", s.Name, err)
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError
					return err
				}
				err = health.IsHealthy(c, obj)
				if err != nil {
					log.Printf("PlanExecutionController: Obj is NOT healthy: %+v", obj)
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateInProgress
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateInProgress
				}
			}
			log.Printf("PlanExecutionController: Phase \"%v\" has strategy %v", phase.Name, phase.Strategy)
			if phase.Strategy == kudov1alpha1.Serial {
				//we need to skip the rest of the steps if this step is unhealthy
				log.Printf("PlanExecutionController: Phase \"%v\" marked as serial", phase.Name)
				if planExecution.Status.Phases[i].Steps[j].State != kudov1alpha1.PhaseStateComplete {
					log.Printf("PlanExecutionController: Step \"%v\" isn't complete, skipping rest of steps in phase until it is", planExecution.Status.Phases[i].Steps[j].Name)
					break //break step loop
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

		//This phase isn't quite ready yet.  Lets see what needs to be done
		planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateInProgress

		//Don't keep going to other plans if we're flagged to perform the phases in serial
		if executedPlan.Strategy == kudov1alpha1.Serial {
			log.Printf("PlanExecutionController: Phase \"%v\" not healthy, and plan marked as serial, so breaking.", phase.Name)
			break
		}
		log.Printf("PlanExecutionController: Looked at phase \"%v\"", phase.Name)
	}
	return nil
}

type FIPP struct {
	FrameworkVersion *kudov1alpha1.FrameworkVersion
	Instance         *kudov1alpha1.Instance
	PlanExecution    *kudov1alpha1.PlanExecution
	Plan             *kudov1alpha1.Plan
}

