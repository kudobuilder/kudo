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
	"time"

	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/health"
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/template"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

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
	return &ReconcilePlanExecution{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
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
				//TODO maybe check the owner is the correct type?
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      owner.Name,
						Namespace: a.Meta.GetNamespace(),
					},
				})
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
	scheme *runtime.Scheme
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
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
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
		planExecution.Status.State = maestrov1alpha1.PhaseStateError
		log.Printf("Error getting Instance %v in %v: %v\n",
			planExecution.Spec.Instance.Name,
			planExecution.Spec.Instance.Namespace,
			err)
		return reconcile.Result{}, err
	}

	gvk, err := apiutil.GVKForObject(planExecution.(runtime.Object), r.scheme)
	if err != nil {
		return err
	}

	// Create a new ref
	ref := *v1.NewControllerRef(planExecution, schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind})

	//see if we need to update the status of Instance
	instance.Status.ActivePlan = ref
	err = r.A

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
		log.Printf("Error getting FrameworkVersion %v in %v: %v\n",
			instance.Spec.FrameworkVersion.Name,
			instance.Spec.FrameworkVersion.Namespace,
			err)
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
		err = fmt.Errorf("Could not find required plan (%v)", planExecution.Spec.PlanName)
		planExecution.Status.State = maestrov1alpha1.PhaseStateError
		return reconcile.Result{}, err
	}

	planExecution.Status.ActivePlan = planName
	planExecution.Status.PlanStatus = maestrov1alpha1.PlanStatus{
		Name:     planName,
		Strategy: executedPlan.Strategy,
	}

	planExecution.Status.PlanStatus.Phases = make([]maestrov1alpha1.PhaseStatus, len(executedPlan.Phases))
	for i, phase := range executedPlan.Phases {
		//populate the Status elements in instance
		planExecution.Status.PlanStatus.Phases[i].Name = phase.Name
		planExecution.Status.PlanStatus.Phases[i].Strategy = phase.Strategy
		planExecution.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStatePending
		planExecution.Status.PlanStatus.Phases[i].Steps = make([]maestrov1alpha1.StepStatus, len(phase.Steps))
		for j, step := range phase.Steps {
			appliedYaml, err := template.ExpandMustache(step.Mustache, configs)
			if err != nil {
				log.Printf("Error applying configs to step %v in phase %v of plan %v: %v", step.Name, phase.Name, planName, err)
				return reconcile.Result{}, err
			}
			objs, err := template.ParseKubernetesObjects(*appliedYaml)
			if err != nil {
				log.Printf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planName, err)
				return reconcile.Result{}, err
			}
			planExecution.Status.PlanStatus.Phases[i].Steps[j].Name = step.Name
			planExecution.Status.PlanStatus.Phases[i].Steps[j].Objects = objs
		}
	}

	for i, phase := range planExecution.Status.PlanStatus.Phases {
		//If we still want to execute phases in this plan
		//check if phase is healthy
		for j, s := range phase.Steps {
			planExecution.Status.PlanStatus.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateComplete

			for _, obj := range s.Objects {
				//Make sure this objet is applied to the cluster.  Get back the instance from
				// the cluster so we can see if it's healthy or not
				if err = controllerutil.SetControllerReference(instance, obj, r.scheme); err != nil {
					return nil, err
				}
				obj, err = r.ApplyObject(obj)
				if err != nil {
					log.Printf("Error applying Object in step:%v: %v\n", s.Name, err)
					planExecution.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateError
					planExecution.Status.PlanStatus.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateError
					return reconcile.Result{}, err
				}
				err = health.IsHealthy(obj)
				if err != nil {
					fmt.Printf("Obj is NOT healthy: %v\n", obj)
					planExecution.Status.PlanStatus.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateInProgress
					planExecution.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateInProgress
				}
			}
			fmt.Printf("Phase %v has strategy %v\n", phase.Name, phase.Strategy)
			if phase.Strategy == maestrov1alpha1.Serial {
				//we need to skip the rest of the steps if this step is unhealthy
				fmt.Printf("Phase %v marked as serial\n", phase.Name)
				if planExecution.Status.PlanStatus.Phases[i].Steps[j].State != maestrov1alpha1.PhaseStateComplete {
					fmt.Printf("Step %v isn't complete, skipping rest of steps in phase until it is\n", instance.Status.PlanStatus.Phases[i].Steps[j].Name)
					break //break step loop
				} else {
					fmt.Printf("Step %v is healthy, so I can continue on\n", planExecution.Status.PlanStatus.Phases[i].Steps[j].Name)
				}
			}

			fmt.Printf("Step %v looked at\n", s.Name)
		}
		if health.IsPhaseHealthy(planExecution.Status.PlanStatus.Phases[i]) {
			fmt.Printf("Phase %v marked as healthy\n", phase.Name)
			planExecution.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateComplete
			continue
		}

		//This phase isn't quite ready yet.  Lets see what needs to be done
		planExecution.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateInProgress

		//Don't keep goign to other plans if we're flagged to perform the phases in serial
		if executedPlan.Strategy == maestrov1alpha1.Serial {
			fmt.Printf("Phase %v not healthy, and plan marked as serial, so breaking.\n", phase.Name)
			break
		}
		fmt.Printf("Phase %v looked at\n", phase.Name)
	}

	if health.IsPlanHealthy(planExecution.Status.PlanStatus) {
		planExecution.Status.PlanStatus.State = maestrov1alpha1.PhaseStateComplete
	} else {
		planExecution.Status.PlanStatus.State = maestrov1alpha1.PhaseStateInProgress
	}

	return reconcile.Result{}, nil
}

//ApplyObject takes the object provided and either creates or updates it depending on whether the object
// exixts or not
func (r *ReconcileInstance) ApplyObject(obj runtime.Object) (runtime.Object, error) {
	nnn, _ := client.ObjectKeyFromObject(obj)
	switch o := obj.(type) {
	//Service
	case *corev1.Service:
		svc := &corev1.Service{}
		err := r.Get(context.TODO(), nnn, svc)
		if err != nil && errors.IsNotFound(err) {
			svc = obj.(*corev1.Service)
			err = r.Create(context.TODO(), svc)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			//This gets autogetnerated, so don't overwrite it with a blank
			//value
			obj.(*corev1.Service).Spec.ClusterIP = svc.Spec.ClusterIP
			svc.Spec = obj.(*corev1.Service).Spec
			svc.Labels = obj.(*corev1.Service).Labels
			svc.Annotations = obj.(*corev1.Service).Annotations
			err = r.Update(context.TODO(), svc)
		}
		if err != nil {
			return nil, err
		}

		//Sleep to wait for the obejct to show up?
		time.Sleep(1 * time.Second)

		//get the copy from the cluster now that things have been applied:
		err = r.Get(context.TODO(), nnn, svc)
		return svc, err

	case *appsv1.StatefulSet:
		ss := &appsv1.StatefulSet{}
		err := r.Get(context.TODO(), nnn, ss)
		if err != nil && errors.IsNotFound(err) {
			ss = obj.(*appsv1.StatefulSet)
			err = r.Create(context.TODO(), ss)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			ss.Spec = obj.(*appsv1.StatefulSet).Spec
			ss.Labels = obj.(*appsv1.StatefulSet).Labels
			ss.Annotations = obj.(*appsv1.StatefulSet).Annotations
			err = r.Update(context.TODO(), ss)
		}
		if err != nil {
			return nil, err
		}

		//Sleep to wait for the obejct to show up?
		time.Sleep(1 * time.Second)
		//get the copy from the cluster now that things have been applied:
		err = r.Get(context.TODO(), nnn, ss)
		return ss, err
	case *policyv1beta1.PodDisruptionBudget:
		pdb := &policyv1beta1.PodDisruptionBudget{}
		err := r.Get(context.TODO(), nnn, pdb)
		if err != nil && errors.IsNotFound(err) {
			pdb = obj.(*policyv1beta1.PodDisruptionBudget)
			err = r.Create(context.TODO(), pdb)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			pdb.Spec = obj.(*policyv1beta1.PodDisruptionBudget).Spec
			pdb.Labels = obj.(*policyv1beta1.PodDisruptionBudget).Labels
			pdb.Annotations = obj.(*policyv1beta1.PodDisruptionBudget).Annotations
			err = r.Update(context.TODO(), pdb)
		}
		if err != nil {
			return nil, err
		}

		//Sleep to wait for the obejct to show up?
		time.Sleep(1 * time.Second)
		//get the copy from the cluster now that things have been applied:
		err = r.Get(context.TODO(), nnn, pdb)
		return pdb, err
	case *corev1.ConfigMap:
		cm := &corev1.ConfigMap{}
		err := r.Get(context.TODO(), nnn, cm)
		if err != nil && errors.IsNotFound(err) {
			cm = obj.(*corev1.ConfigMap)
			if err := controllerutil.SetControllerReference(parent, cm, r.scheme); err != nil {
				return nil, err
			}
			err = r.Create(context.TODO(), cm)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			cm.Data = obj.(*corev1.ConfigMap).Data
			cm.Labels = obj.(*corev1.ConfigMap).Labels
			cm.Annotations = obj.(*corev1.ConfigMap).Annotations
			err = r.Update(context.TODO(), cm)
		}
		if err != nil {
			return nil, err
		}
		//Sleep to wait for the obejct to show up?
		time.Sleep(1 * time.Second)

		//get the copy from the cluster now that things have been applied:
		err = r.Get(context.TODO(), nnn, cm)
		return cm, err
	case *batchv1.Job:
		job := &batchv1.Job{}
		err := r.Get(context.TODO(), nnn, job)
		if err != nil && errors.IsNotFound(err) {
			job = obj.(*batchv1.Job)
			err = r.Create(context.TODO(), job)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		//Don't update Jobs if they're already running.

		//Sleep to wait for the obejct to show up?
		time.Sleep(1 * time.Second)

		//get the copy from the cluster now that things have been applied:
		err = r.Get(context.TODO(), nnn, job)
		return job, err
	default:
		return nil, fmt.Errorf("I dont know how to update types %v.  Please implement", o)

	}
}
