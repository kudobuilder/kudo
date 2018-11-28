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

package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/health"
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/template"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
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

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Instance Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this maestro.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInstance{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("instance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Instance
	err = c.Watch(&source.Kind{Type: &maestrov1alpha1.Instance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	//Watch for Deployments, Jobs and StatefulSets
	// Define a mapping from the object in the event to one or more
	// objects to Reconcile
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
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// The object doesn't contain label "foo", so the event will be
			// ignored.
			fmt.Printf("Recieved UpdateEvent for event %v/%v:\n", e.ObjectNew.GetObjectKind(), e.MetaNew.GetName())
			fmt.Printf("NewSpec:\n")
			fmt.Printf("%v\n", e.ObjectNew)
			fmt.Printf("OldSpec:\n")
			fmt.Printf("%v\n", e.ObjectOld)
			if _, ok := e.MetaOld.GetLabels()["foo"]; !ok {
				return false
			}
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			fmt.Printf("Recieved CreateEvent for event %v/%v:\n", e.Object.GetObjectKind(), e.Meta.GetName())
			if _, ok := e.Meta.GetLabels()["foo"]; !ok {
				return false
			}
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			fmt.Printf("Recieved DeleteEvent for event %v/%v:\n", e.Object.GetObjectKind(), e.Meta.GetName())
			return true

		},
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

var _ reconcile.Reconciler = &ReconcileInstance{}

// ReconcileInstance reconciles a Instance object
type ReconcileInstance struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Instance object and makes changes based on the state read
// and what is in the Instance.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maestro.k8s.io,resources=instances,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Instance instance
	instance := &maestrov1alpha1.Instance{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log.Printf("InstanceController: Recieved Reconcile request for %v\n", request.Name)

	//Create configmap to hold all parameters for instantiation
	configs := make(map[string]string)
	//Default parameters from instance metadata
	configs["FRAMEWORK_NAME"] = instance.Name
	configs["NAME"] = instance.Name
	configs["NAMESPACE"] = instance.Namespace
	//parameters from instance spec
	for k, v := range instance.Spec.Parameters {
		configs[k] = v
	}
	//grab Framework instance (TODO Switch to FrameworkVersion)
	frameworkVersion := &maestrov1alpha1.FrameworkVersion{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Name:      instance.Spec.FrameworkVersion.Name,
		Namespace: instance.Spec.FrameworkVersion.Namespace,
	}, frameworkVersion)

	if err != nil {
		log.Printf("InstanceController: Could not find FrameworkVersion with name %v: %v\n", instance.Spec.FrameworkVersion.Name, err)
		return reconcile.Result{}, err
	}

	//merge defaults with customizations
	for k, v := range frameworkVersion.Spec.Defaults {
		_, ok := configs[k]
		if !ok { //not specified in params
			configs[k] = v
		}
	}

	if true {
		b, _ := json.MarshalIndent(frameworkVersion, "", "\t")
		fmt.Println(string(b))
	}

	//Now we need to see what plan should be executed:
	//TODO actually figure it out, for now assume deploy == update
	planName := "deploy"
	executedPlan, ok := frameworkVersion.Spec.Plans[planName]
	if !ok {
		err = fmt.Errorf("Could not find required plan (%v)", planName)
		return reconcile.Result{}, err
	}

	//populate the correct objects inside of the instance
	//Clean start:
	instance.Status.ActivePlan = planName
	instance.Status.PlanStatus = maestrov1alpha1.PlanStatus{
		Name:     planName,
		Strategy: executedPlan.Strategy,
	}
	instance.Status.PlanStatus.Phases = make([]maestrov1alpha1.PhaseStatus, len(executedPlan.Phases))
	for i, phase := range executedPlan.Phases {
		//populate the Status elements in instance
		instance.Status.PlanStatus.Phases[i].Name = phase.Name
		instance.Status.PlanStatus.Phases[i].Strategy = phase.Strategy
		instance.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStatePending
		instance.Status.PlanStatus.Phases[i].Steps = make([]maestrov1alpha1.StepStatus, len(phase.Steps))
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
			instance.Status.PlanStatus.Phases[i].Steps[j].Name = step.Name
			instance.Status.PlanStatus.Phases[i].Steps[j].Objects = objs
		}
	}

	//Before returning from this function, update the status
	defer r.Update(context.Background(), instance)

	for i, phase := range instance.Status.PlanStatus.Phases {
		//If we still want to execute phases in this plan
		//check if phase is healthy
		for j, s := range phase.Steps {
			instance.Status.PlanStatus.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateComplete

			for _, obj := range s.Objects {
				//Make sure this objet is applied to the cluster.  Get back the instance from
				// the cluster so we can see if it's healthy or not
				obj, err = r.ApplyObject(obj, instance)
				if err != nil {
					log.Printf("Error applying Object: type: %v step:%v: %v\n", obj.GetObjectKind().GroupVersionKind(), s.Name, err)
					instance.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateError
					instance.Status.PlanStatus.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateError
					return reconcile.Result{}, err
				}
				err = health.IsHealthy(obj)
				if err != nil {
					fmt.Printf("Obj is NOT healthy: %v\n", obj)
					instance.Status.PlanStatus.Phases[i].Steps[j].State = maestrov1alpha1.PhaseStateInProgress
					instance.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateInProgress
				}
			}
			fmt.Printf("Step %v looked at\n", s.Name)
		}
		if health.IsPhaseHealthy(instance.Status.PlanStatus.Phases[i]) {
			fmt.Printf("Phase %v marked as healthy\n", phase.Name)
			instance.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateComplete
			continue
		}

		//This phase isn't quite ready yet.  Lets see what needs to be done
		instance.Status.PlanStatus.Phases[i].State = maestrov1alpha1.PhaseStateInProgress

		//Don't keep goign to other plans if we're flagged to perform the phases in serial
		if phase.Strategy == maestrov1alpha1.Serial {
			fmt.Printf("Phase %v not healthy, and plan marked as serial, so breaking.\n", phase.Name)
			break
		}
		fmt.Printf("Phase %v looked at\n", phase.Name)
	}

	if health.IsPlanHealthy(instance.Status.PlanStatus) {
		instance.Status.PlanStatus.State = maestrov1alpha1.PhaseStateComplete
	} else {
		instance.Status.PlanStatus.State = maestrov1alpha1.PhaseStateInProgress
	}

	//defer call from above should apply the status changes to the object
	return reconcile.Result{}, nil
}

//ApplyObject takes the object provided and either creates or updates it depending on whether the object
// exixts or not
func (r *ReconcileInstance) ApplyObject(obj runtime.Object, parent metav1.Object) (runtime.Object, error) {
	nnn, _ := client.ObjectKeyFromObject(obj)
	switch o := obj.(type) {
	//Service
	case *corev1.Service:
		svc := &corev1.Service{}
		err := r.Get(context.TODO(), nnn, svc)
		if err != nil && errors.IsNotFound(err) {
			svc = obj.(*corev1.Service)
			if err = controllerutil.SetControllerReference(parent, svc, r.scheme); err != nil {
				return nil, err
			}
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
			if err := controllerutil.SetControllerReference(parent, ss, r.scheme); err != nil {
				return nil, err
			}
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
			if err := controllerutil.SetControllerReference(parent, pdb, r.scheme); err != nil {
				return nil, err
			}
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
	default:
		return nil, fmt.Errorf("I dont know how to update types %v.  Please implement", o)

	}
}
