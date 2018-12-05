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
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/record"
	"log"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
	"strconv"

	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/health"
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/template"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

const basePath = "/kustomize"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PlanExecution Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this maestro.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
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

	//Watch for Deployments, Jobs and StatefulSets
	// Define a mapping from the object in the event to one or more
	// objects to Reconcile.  Specifically this calls for
	// a reconsiliation of any objects "Owner".
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			owners := a.Meta.GetOwnerReferences()
			requests := make([]reconcile.Request, 0)
			for _, owner := range owners {
				//if owner is an instance, we also want to queue up the
				// PlanExecution in the Status section
				inst := &maestrov1alpha1.Instance{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{
					Name:      owner.Name,
					Namespace: a.Meta.GetNamespace(),
				}, inst)

				if err != nil {
					fmt.Printf("Error getting instance object: %v\n", err)
				} else {
					fmt.Printf("Adding %v to reconcile\n", inst.Status.ActivePlan.Name)
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

	// 'UpdateFunc' and 'CreateFunc' used to judge if a event about the object is
	// what we want. If that is true, the event will be processed by the reconciler.

	//PlanExecutions should be mostly immutable.  Updates should only

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
	}

	// Watch for changes to PlanExecution
	err = c.Watch(&source.Kind{Type: &maestrov1alpha1.PlanExecution{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch Deployments and trigger Reconciles for objects
	// mapped from the Deployment in the event
	err = c.Watch(
		&source.Kind{Type: &appsv1.StatefulSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		// Comment it if default predicate fun is used.
		p)
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		// Comment it if default predicate fun is used.
		p)
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		// Comment it if default predicate fun is used.
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
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maestro.k8s.io,resources=planexecutions,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcilePlanExecution) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the PlanExecution instance
	planExecution := &maestrov1alpha1.PlanExecution{}
	err := r.Get(context.TODO(), request.NamespacedName, planExecution)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("Could not find planExecution %v: %v\n", request.Name, err)
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//Check for Suspend set.
	if planExecution.Spec.Suspend != nil && *planExecution.Spec.Suspend {
		planExecution.Status.State = maestrov1alpha1.PhaseStateSuspend
		err = r.Update(context.TODO(), planExecution)
		return reconcile.Result{}, err
	}

	//See if this has already been proceeded
	if planExecution.Status.State == maestrov1alpha1.PhaseStateComplete {
		fmt.Printf("PlanExecution %v has already run to completion, not processing.\n", planExecution.Name)
	}

	//Get Instance Object
	instance := &maestrov1alpha1.Instance{}
	frameworkVersion := &maestrov1alpha1.FrameworkVersion{}
	//Before returning from this function, update the status
	defer r.Update(context.Background(), planExecution)

	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      planExecution.Spec.Instance.Name,
			Namespace: planExecution.Spec.Instance.Namespace,
		},
		instance)
	if err != nil {
		//TODO how to handle errors.
		//Can't find the instance.  Update sta
		r.recorder.Event(planExecution, "Warning", "InvalidInstance", fmt.Sprintf("Could not find required instance (%v)", planExecution.Spec.Instance.Name))
		planExecution.Status.State = maestrov1alpha1.PhaseStateError
		log.Printf("Error getting Instance %v in %v: %v\n",
			planExecution.Spec.Instance.Name,
			planExecution.Spec.Instance.Namespace,
			err)
		return reconcile.Result{}, err
	}

	//need to add ownerRefernce as the Instance

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
		fmt.Printf("Upate of instance with ActivePlan errored: %v\n", err)
		b, _ := json.MarshalIndent(instance, "", "\t")
		fmt.Println(string(b))
	}

	//TODO add the reference
	// gvk, err := apiutil.GVKForObject(planExecution.(runtime.Object), r.scheme)
	// if err != nil {
	// 	return err
	// }

	// // Create a new ref
	// ref := *v1.NewControllerRef(planExecution, schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind})

	// //see if we need to update the status of Instance
	// instance.Status.ActivePlan = ref
	// err = r.A

	//Get associated FrameworkVersion
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.FrameworkVersion.Name,
			Namespace: instance.Spec.FrameworkVersion.Namespace,
		},
		frameworkVersion)
	if err != nil {
		//TODO how to handle errors.
		//Can't find the instance.  Update sta
		planExecution.Status.State = maestrov1alpha1.PhaseStateError
		r.recorder.Event(planExecution, "Warning", "InvalidFrameworkVersion", fmt.Sprintf("Could not find FrameworkVersion %v", instance.Spec.FrameworkVersion.Name))
		log.Printf("Error getting FrameworkVersion %v in %v: %v\n",
			instance.Spec.FrameworkVersion.Name,
			instance.Spec.FrameworkVersion.Namespace,
			err)
		return reconcile.Result{}, err
	}

	//Load parameters:
	//Create configmap to hold all parameters for instantiation
	configs := make(map[string]string)
	//Default parameters from instance metadata
	configs["FRAMEWORK_NAME"] = frameworkVersion.Spec.Framework.Name
	configs["NAME"] = instance.Name
	configs["NAMESPACE"] = instance.Namespace
	//parameters from instance spec
	for k, v := range instance.Spec.Parameters {
		configs[k] = v
	}
	//merge defaults with customizations
	for k, v := range frameworkVersion.Spec.Defaults {
		_, ok := configs[k]
		if !ok { //not specified in params
			configs[k] = v
		}
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
		err = fmt.Errorf("Could not find required plan (%v)", planExecution.Spec.PlanName)
		planExecution.Status.State = maestrov1alpha1.PhaseStateError
		return reconcile.Result{}, err
	}

	planExecution.Status.Name = planExecution.Spec.PlanName
	planExecution.Status.Strategy = executedPlan.Strategy

	planExecution.Status.Phases = make([]maestrov1alpha1.PhaseStatus, len(executedPlan.Phases))
	for i, phase := range executedPlan.Phases {
		//populate the Status elements in instance
		planExecution.Status.Phases[i].Name = phase.Name
		planExecution.Status.Phases[i].Strategy = phase.Strategy
		planExecution.Status.Phases[i].State = maestrov1alpha1.PhaseStatePending
		planExecution.Status.Phases[i].Steps = make([]maestrov1alpha1.StepStatus, len(phase.Steps))
		for j, step := range phase.Steps {
			// fetch FrameworkVersion
			// get the task name from the step
			// get the task definition from the FV
			// create the kustomize templates
			// apply
			configs["PLAN_NAME"] = planExecution.Spec.PlanName
			configs["PHASE_NAME"] = phase.Name
			configs["STEP_NAME"] = step.Name
			configs["STEP_NUMBER"] = strconv.FormatInt(int64(j), 10)

			var objs []runtime.Object
			for _, t := range step.Tasks {
				// resolve task
				if taskSpec, ok := frameworkVersion.Spec.Tasks[t]; ok {
					var resources []string
					fsys := fs.MakeFakeFS()

					for _, res := range taskSpec.Resources {
						if resource, ok := frameworkVersion.Spec.Templates[res]; ok {
							templatedYaml, err := template.ExpandMustache(resource, configs)
							r.recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error expanding mustache: %v", err))
							if err != nil {
								log.Printf("Error expanding mustache: %v\n", err)
							}
							fsys.WriteFile(fmt.Sprintf("%s/%s", basePath, res), []byte(*templatedYaml))
							resources = append(resources, res)

						} else {
							r.recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding resource named %s for framework version %s", res, frameworkVersion.Name))
							log.Printf("Error finding resource named %s for framework version %s\n", res, frameworkVersion.Name)
							return reconcile.Result{}, err
						}
					}

					kustomization := &ktypes.Kustomization{
						NamePrefix: instance.Name + "-",
						Namespace:  instance.Namespace,
						CommonLabels: map[string]string{
							"heritage":      "maestro",
							"app":           frameworkVersion.Spec.Framework.Name,
							"version":       frameworkVersion.Spec.Version,
							"instance":      instance.Name,
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
						return reconcile.Result{}, err
					}

					fsys.WriteFile(fmt.Sprintf("%s/kustomization.yaml", basePath), yamlBytes)

					ldr, err := loader.NewLoader(basePath, fsys)
					if err != nil {
						return reconcile.Result{}, err
					}
					defer ldr.Cleanup()

					rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()))
					kt, err := target.NewKustTarget(ldr, fsys, rf, transformer.NewFactoryImpl())
					if err != nil {
						return reconcile.Result{}, err
					}

					allResources, err := kt.MakeCustomizedResMap()
					if err != nil {
						return reconcile.Result{}, err
					}

					res, err := allResources.EncodeAsYaml()
					if err != nil {
						return reconcile.Result{}, err
					}

					objsToAdd, err := template.ParseKubernetesObjects(string(res))
					if err != nil {
						r.recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err))
						log.Printf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err)
						return reconcile.Result{}, err
					}
					objs = append(objs, objsToAdd...)
				} else {
					r.recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding task named %s for framework version %s", taskSpec, frameworkVersion.Name))
					log.Printf("Error finding task named %s for framework version %s", taskSpec, frameworkVersion.Name)
					return reconcile.Result{}, err
				}
			}

			planExecution.Status.Phases[i].Steps[j].Name = step.Name
			planExecution.Status.Phases[i].Steps[j].Objects = objs
			fmt.Printf("Phase %v Step %v has %v objects\n", i, j, len(objs))
		}
	}

	for i, phase := range planExecution.Status.Phases {
		//If we still want to execute phases in this plan
		//check if phase is healthy
		for j, s := range phase.Steps {
			planExecution.Status.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateComplete

			for _, obj := range s.Objects {
				//Make sure this objet is applied to the cluster.  Get back the instance from
				// the cluster so we can see if it's healthy or not
				if err = controllerutil.SetControllerReference(instance, obj.(metav1.Object), r.scheme); err != nil {
					return reconcile.Result{}, err
				}

				//Some objects don't update well.  We capture the logic here to see if we need to cleanup the current object
				err = r.Cleanup(obj)
				if err != nil {
					fmt.Printf("Cleanup failed: %v\n", err)
				}
				result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, obj, func(runtime.Object) error { return nil })

				log.Printf("CreateOrUpdate resulted in: %v\n", result)
				if err != nil {
					log.Printf("Error CreateOrUpdate Object in step:%v: %v\n", s.Name, err)
					planExecution.Status.Phases[i].State = maestrov1alpha1.PhaseStateError
					planExecution.Status.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateError
					return reconcile.Result{}, err
				}
				// get the existing object meta
				metaObj := obj.(metav1.Object)

				// retrieve the existing object
				key := client.ObjectKey{
					Name:      metaObj.GetName(),
					Namespace: metaObj.GetNamespace(),
				}

				err = r.Client.Get(context.TODO(), key, obj)

				if err != nil {
					log.Printf("Error Getting new Object in step:%v: %v\n", s.Name, err)
					planExecution.Status.Phases[i].State = maestrov1alpha1.PhaseStateError
					planExecution.Status.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateError
					return reconcile.Result{}, err
				}
				err = health.IsHealthy(r.Client, obj)
				if err != nil {
					fmt.Printf("Obj is NOT healthy: %v\n", obj)
					planExecution.Status.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateInProgress
					planExecution.Status.Phases[i].State = maestrov1alpha1.PhaseStateInProgress
				}
			}
			fmt.Printf("Phase %v has strategy %v\n", phase.Name, phase.Strategy)
			if phase.Strategy == maestrov1alpha1.Serial {
				//we need to skip the rest of the steps if this step is unhealthy
				fmt.Printf("Phase %v marked as serial\n", phase.Name)
				if planExecution.Status.Phases[i].Steps[j].State != maestrov1alpha1.PhaseStateComplete {
					fmt.Printf("Step %v isn't complete, skipping rest of steps in phase until it is\n", planExecution.Status.Phases[i].Steps[j].Name)
					break //break step loop
				} else {
					fmt.Printf("Step %v is healthy, so I can continue on\n", planExecution.Status.Phases[i].Steps[j].Name)
				}
			}

			fmt.Printf("Step %v looked at\n", s.Name)
		}
		if health.IsPhaseHealthy(planExecution.Status.Phases[i]) {
			fmt.Printf("Phase %v marked as healthy\n", phase.Name)
			planExecution.Status.Phases[i].State = maestrov1alpha1.PhaseStateComplete
			continue
		}

		//This phase isn't quite ready yet.  Lets see what needs to be done
		planExecution.Status.Phases[i].State = maestrov1alpha1.PhaseStateInProgress

		//Don't keep goign to other plans if we're flagged to perform the phases in serial
		if executedPlan.Strategy == maestrov1alpha1.Serial {
			fmt.Printf("Phase %v not healthy, and plan marked as serial, so breaking.\n", phase.Name)
			break
		}
		fmt.Printf("Phase %v looked at\n", phase.Name)
	}

	if health.IsPlanHealthy(planExecution.Status) {
		r.recorder.Event(planExecution, "Normal", "PhaseStateComplete", fmt.Sprintf("Instances healthy, phase marked as COMPLETE"))
		planExecution.Status.State = maestrov1alpha1.PhaseStateComplete
	} else {
		planExecution.Status.State = maestrov1alpha1.PhaseStateInProgress
	}

	return reconcile.Result{}, nil
}

//Cleanup modfies objects on the cluster to allow for the provided obj to get CreateOrApply.  Currently
//only needs to clean up Jobs that get run from multiple PlanExecutions
func (r *ReconcilePlanExecution) Cleanup(obj runtime.Object) error {
	switch obj.(type) {
	case *batchv1.Job:
		//We need to see if there's a current job on the system that matches this exactly (with labels)
		job := obj.(*batchv1.Job)
		fmt.Printf("PlanExecutionController.Cleanup: *batchv1.Job %v\n", job.Name)

		present := &batchv1.Job{}
		key, _ := client.ObjectKeyFromObject(obj)
		err := r.Get(context.TODO(), key, present)
		if errors.IsNotFound(err) {
			//this is fine, its good to go
			fmt.Printf("Could not find job %v on cluster.  Good to make a new one:\n", key)
			return nil
		}
		if err != nil {
			//Something else happened
			return err
		}
		//see if thie job on the cluster has the same labels as the one we're looking to add.
		for k, v := range job.Labels {
			if v != present.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				fmt.Printf("Different values for key %v: %v and %v\n", k, v, present.Labels[k])
				err = r.Delete(context.TODO(), present)
				return err
			}
		}
		for k, v := range present.Labels {
			if v != job.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				fmt.Printf("Different values for key %v: %v and %v\n", k, v, job.Labels[k])
				err = r.Delete(context.TODO(), present)
				return err
			}
		}
		return nil
	}

	return nil
}
