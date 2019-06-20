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
	"fmt"
	"log"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Instance Controller and adds it to the Manager with default RBAC.
//
// The Manager will set fields on the Controller and start it when the Manager is started.
func Add(mgr manager.Manager) error {
	log.Printf("InstanceController: Registering instance controller.")
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInstance{Client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetRecorder("instance-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("instance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Instance
	err = c.Watch(&source.Kind{Type: &kudov1alpha1.Instance{}}, &handler.EnqueueRequestForObject{}, instanceEventPredicateFunc(mgr))
	if err != nil {
		return err
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcileInstance{}

// ReconcileInstance reconciles a Instance object
type ReconcileInstance struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Instance object and makes changes based on the state read
// and what is in the Instance.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.k8s.io,resources=instances,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Instance instance
	instance := &kudov1alpha1.Instance{}
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

	log.Printf("InstanceController: Received Reconcile request for \"%+v\"", request.Name)

	//Make sure the FrameworkVersion is present
	fv := &kudov1alpha1.FrameworkVersion{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.FrameworkVersion.Name,
			Namespace: instance.Spec.FrameworkVersion.Namespace,
		},
		fv)
	if err != nil {
		log.Printf("InstanceController: Error getting frameworkversion \"%v\" for instance \"%v\": %v",
			instance.Spec.FrameworkVersion.Name,
			instance.Name,
			err)
		r.recorder.Event(instance, "Warning", "InvalidFrameworkVersion", fmt.Sprintf("Error getting frameworkversion \"%v\": %v", fv.Name, err))
		return reconcile.Result{}, err
	}

	//make sure all the required parameters in the frameworkversion are present
	for _, param := range fv.Spec.Parameters {
		if param.Required {
			if _, ok := instance.Spec.Parameters[param.Name]; !ok {
				r.recorder.Event(instance, "Warning", "MissingParameter", fmt.Sprintf("Missing parameter \"%v\" required by frameworkversion \"%v\"", param.Name, fv.Name))
			}
		}
	}
	return reconcile.Result{}, nil
}

func instanceEventPredicateFunc(mgr manager.Manager) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			old := e.ObjectOld.(*kudov1alpha1.Instance)
			new := e.ObjectNew.(*kudov1alpha1.Instance)

			// Haven't done anything yet
			if new.Status.ActivePlan.Name == "" {
				err := createPlan(mgr, "deploy", new)
				if err != nil {
					log.Printf("InstanceEventPredicate: Error creating \"%v\" object for \"%v\": %v", "deploy", new.Name, err)
				}
				return true
			}

			// Get the new FrameworkVersion object
			fv := &kudov1alpha1.FrameworkVersion{}
			err := mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      new.Spec.FrameworkVersion.Name,
					Namespace: new.Spec.FrameworkVersion.Namespace,
				},
				fv)
			if err != nil {
				log.Printf("InstanceEventPredicate: Error getting frameworkversion \"%v\" for instance \"%v\": %v",
					new.Spec.FrameworkVersion.Name,
					new.Name,
					err)
				// TODO: We probably want to handle this differently and mark this instance as unhealthy
				// since its linking to a bad FV.
				return false
			}
			// Identify plan to be executed by this change
			var planName string
			var ok bool
			if old.Spec.FrameworkVersion != new.Spec.FrameworkVersion {
				// Its an Upgrade!
				_, ok = fv.Spec.Plans["upgrade"]
				if !ok {
					_, ok = fv.Spec.Plans["update"]
					if !ok {
						_, ok = fv.Spec.Plans["deploy"]
						if !ok {
							log.Println("InstanceEventPredicate: Could not find any plan to use for upgrade")
							return false
						}
						ok = true // TODO: Do we need this here?
						planName = "deploy"
					} else {
						planName = "update"
					}
				} else {
					planName = "upgrade"
				}
			} else if !reflect.DeepEqual(old.Spec, new.Spec) {
				for k := range parameterDifference(old.Spec.Parameters, new.Spec.Parameters) {
					// Find the right parameter in the FV
					for _, param := range fv.Spec.Parameters {
						if param.Name == k {
							planName = param.Trigger
							ok = true
						}
					}
					if !ok {
						log.Printf("InstanceController: Instance %v updated parameter %v, but parameter not found in FrameworkVersion %v\n", new.Name, k, fv.Name)
					} else if planName == "" {
						_, ok = fv.Spec.Plans["update"]
						if !ok {
							_, ok = fv.Spec.Plans["deploy"]
							if !ok {
								log.Println("InstanceController: Could not find any plan to use for update")
							} else {
								planName = "deploy"
							}
						} else {
							planName = "update"
						}
						log.Printf("InstanceController: Instance %v updated parameter %v, but no specified trigger.  Using default plan %v\n", new.Name, k, planName)
					}
				}
				// Not currently doing anything for Dependency changes.
			} else {
				log.Println("InstanceEventPredicate: Old and new spec matched...")
				planName = "deploy"
			}
			log.Printf("InstanceEventPredicate: Going to call plan \"%v\"", planName)

			// we found something
			if ok {

				// Mark the current plan as Suspend
				current := &kudov1alpha1.PlanExecution{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{Name: new.Status.ActivePlan.Name, Namespace: new.Status.ActivePlan.Namespace}, current)
				if err != nil {
					log.Printf("InstanceEventPredicate: Ignoring error when getting plan for new instance: %v", err)
				} else {
					if current.Status.State == kudov1alpha1.PhaseStateComplete {
						log.Println("InstanceEventPredicate: Current Plan for Instance is already done, won't change the Suspend flag.")
					} else {
						log.Println("InstanceEventPredicate: Setting PlanExecution to Suspend")
						t := true
						current.Spec.Suspend = &t
						did, err := controllerutil.CreateOrUpdate(context.TODO(), mgr.GetClient(), current, func(o runtime.Object) error {
							t := true
							o.(*kudov1alpha1.PlanExecution).Spec.Suspend = &t
							return nil
						})
						if err != nil {
							log.Printf("InstanceEventPredicate: Error changing the current PlanExecution to Suspend: %v", err)
						} else {
							log.Printf("InstanceEventPredicate: No error in setting PlanExecution.Suspend to true. Returned %v", did)
						}
					}
				}

				err = createPlan(mgr, planName, new)
				if err != nil {
					log.Printf("InstanceEventPredicate: Error creating \"%v\" object for \"%v\": %v", planName, new.Name, err)
				}
			}

			// See if there's a current plan being run, if so "cancel" the plan run
			return e.ObjectOld != e.ObjectNew
		},
		// New Instances should have Deploy called
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf("InstanceEventPredicate: Received create event for an instance named: %v", e.Meta.GetName())
			instance := e.Object.(*kudov1alpha1.Instance)

			// Get the instance FrameworkVersion object
			fv := &kudov1alpha1.FrameworkVersion{}
			err := mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      instance.Spec.FrameworkVersion.Name,
					Namespace: instance.Spec.FrameworkVersion.Namespace,
				},
				fv)
			if err != nil {
				log.Printf("InstanceEventPredicate: Error getting frameworkversion \"%v\" for instance \"%v\": %v",
					instance.Spec.FrameworkVersion.Name,
					instance.Name,
					err)
				// TODO: We probably want to handle this differently and mark this instance as unhealthy
				// since its linking to a bad FV.
				return false
			}
			planName := "deploy"

			if _, ok := fv.Spec.Plans[planName]; !ok {
				log.Println("InstanceEventPredicate: Could not find deploy plan")
				return false
			}

			err = createPlan(mgr, planName, instance)
			if err != nil {
				log.Printf("InstanceEventPredicate: Error creating \"%v\" object for \"%v\": %v", planName, instance.Name, err)
			}
			return err == nil
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			log.Printf("InstanceEventPredicate: Received delete event for an instance named: %v", e.Meta.GetName())
			return true
		},
	}
}

func createPlan(mgr manager.Manager, planName string, instance *kudov1alpha1.Instance) error {
	gvk, _ := apiutil.GVKForObject(instance, mgr.GetScheme())
	recorder := mgr.GetRecorder("instance-controller")
	recorder.Event(instance, "Normal", "CreatePlanExecution", fmt.Sprintf("Creating \"%v\" plan execution", planName))

	ref := corev1.ObjectReference{
		Kind:      gvk.Kind,
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}

	planExecution := kudov1alpha1.PlanExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v-%v", instance.Name, planName, time.Now().Nanosecond()),
			Namespace: instance.GetNamespace(),
			// TODO: Should also add one for Framework in here as well.
			Labels: map[string]string{
				"framework-version": instance.Spec.FrameworkVersion.Name,
				"instance":          instance.Name,
			},
		},
		Spec: kudov1alpha1.PlanExecutionSpec{
			Instance: ref,
			PlanName: planName,
		},
	}

	// Make this instance the owner of the PlanExecution
	if err := controllerutil.SetControllerReference(instance, &planExecution, mgr.GetScheme()); err != nil {
		log.Printf("InstanceController: Error setting ControllerReference")
		return err
	}

	if err := mgr.GetClient().Create(context.TODO(), &planExecution); err != nil {
		log.Printf("InstanceController: Error creating planexecution \"%v\": %v", planExecution.Name, err)
		recorder.Event(instance, "Warning", "CreatePlanExecution", fmt.Sprintf("Error creating planexecution \"%v\": %v", planExecution.Name, err))
		return err
	}
	recorder.Event(instance, "Normal", "PlanCreated", fmt.Sprintf("PlanExecution \"%v\" created", planExecution.Name))
	return nil
}

func parameterDifference(old, new map[string]string) map[string]string {
	diff := make(map[string]string)

	for key, val := range old {
		// If a parameter was removed in the new spec
		if _, ok := new[key]; !ok {
			diff[key] = val
		}
	}

	for key, val := range new {
		// If new spec parameter was added or changed
		if v, ok := old[key]; !ok || v != val {
			diff[key] = val
		}
	}

	return diff
}
