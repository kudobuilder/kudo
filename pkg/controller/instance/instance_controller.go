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

	"github.com/kudobuilder/kudo/pkg/util/kudo"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

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

// Add creates a new Instance Controller and adds it to the Manager with default RBAC.
//
// The Manager will set fields on the Controller and start it when the Manager is started.
func Add(mgr manager.Manager) error {
	log.Printf("InstanceController: Registering instance controller.")
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInstance{Client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetEventRecorderFor("instance-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("instance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Instance
	if err = c.Watch(&source.Kind{Type: &kudov1alpha1.Instance{}}, &handler.EnqueueRequestForObject{}, instanceEventFilter(mgr)); err != nil {
		return err
	}

	// Watch for changes to OperatorVersion. Since changes to OperatorVersion and Instance are often happening
	// concurrently there is an inherent race between both update events so that we might see a new Instance first
	// without the corresponding OperatorVersion. We additionally watch OperatorVersions and trigger
	// reconciliation for the corresponding instances.
	//
	// Define a mapping from the object in the event (OperatorVersion) to one or more objects to
	// reconcile (Instances). Specifically this calls for a reconciliation of any owned objects.
	ovEventHandler := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			requests := make([]reconcile.Request, 0)
			// We want to query and queue up operators Instances
			instances := &kudov1alpha1.InstanceList{}
			// we are listing all instances here, which could come with some performance penalty
			// a possible optimization is to introduce filtering based on operatorversion (or operator)
			err := mgr.GetClient().List(
				context.TODO(),
				instances,
			)

			if err != nil {
				log.Printf("InstanceController: Error fetching instances list for operator %v: %v", a.Meta.GetName(), err)
				return nil
			}

			for _, instance := range instances.Items {
				// Sanity check - lets make sure that this instance references the operatorVersion
				if instance.Spec.OperatorVersion.Name == a.Meta.GetName() &&
					instance.GetOperatorVersionNamespace() == a.Meta.GetNamespace() &&
					instance.Status.ActivePlan.Name == "" {

					log.Printf("InstanceController: Creating a deploy execution plan for the instance %v", instance.Name)
					err = createPlan(mgr, "deploy", &instance)
					if err != nil {
						log.Printf("InstanceController: Error creating \"%v\" object for \"deploy\": %v", instance.Name, err)
					}

					log.Printf("InstanceController: Queing instance %v for reconciliation", instance.Name)
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      instance.Name,
							Namespace: instance.Namespace,
						},
					})
				}
			}
			log.Printf("InstanceController: Found %v instances to reconcile for operator %v", len(requests), a.Meta.GetName())
			return requests
		})

	// This map function makes sure that we *ONLY* handle created operatorVersion
	ovEventFilter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf("InstanceController: Received create event for: %v", e.Meta.GetName())
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	if err = c.Watch(&source.Kind{Type: &kudov1alpha1.OperatorVersion{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: ovEventHandler}, ovEventFilter); err != nil {
		return err
	}

	return nil
}

func instanceEventFilter(mgr manager.Manager) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*kudov1alpha1.Instance)
			new := e.ObjectNew.(*kudov1alpha1.Instance)

			// Get the OperatorVersion that corresponds to the new instance.
			ov := &kudov1alpha1.OperatorVersion{}
			err := mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      new.Spec.OperatorVersion.Name,
					Namespace: new.GetOperatorVersionNamespace(),
				},
				ov)
			if err != nil {
				log.Printf("InstanceController: Error getting operatorversion \"%v\" for instance \"%v\": %v",
					new.Spec.OperatorVersion.Name,
					new.Name,
					err)
				// TODO: We probably want to handle this differently and mark this instance as unhealthy
				// since it's linking to a bad OV.
				return false
			}

			// Identify plan to be executed by this change.
			var planName string
			var planFound bool

			if old.Spec.OperatorVersion != new.Spec.OperatorVersion {
				// It's an upgrade!
				names := []string{"upgrade", "update", "deploy"}
				for _, n := range names {
					if _, planFound = ov.Spec.Plans[n]; planFound {
						planName = n
						break
					}
				}

				if !planFound {
					log.Printf("InstanceController: Could not find any plan to use to upgrade instance %v", new.Name)
					return false
				}
			} else if !reflect.DeepEqual(old.Spec, new.Spec) {
				for k := range parameterDifference(old.Spec.Parameters, new.Spec.Parameters) {
					// Find the spec of the updated parameter.
					paramFound := false
					for _, param := range ov.Spec.Parameters {
						if param.Name == k {
							paramFound = true

							if param.Trigger != "" {
								planName = param.Trigger
								planFound = true
							}

							break
						}
					}

					if paramFound {
						if !planFound {
							// The parameter doesn't have a trigger, try to find the corresponding default plan.
							names := []string{"update", "deploy"}
							for _, n := range names {
								if _, planFound = ov.Spec.Plans[n]; planFound {
									planName = n
									planFound = true
									break
								}
							}

							if planFound {
								log.Printf("InstanceController: Instance %v updated parameter %v, but it is not associated to a trigger. Using default plan %v\n", new.Name, k, planName)
							}
						}

						if !planFound {
							log.Printf("InstanceController: Could not find any plan to use to update instance %v", new.Name)
						}
					} else {
						log.Printf("InstanceController: Instance %v updated parameter %v, but parameter not found in operatorversion %v\n", new.Name, k, ov.Name)
					}
				}
			} else {
				// FIXME: reading the status here feels very wrong, and so does the fact
				// that this predicate funcion has side effects. We could probably move
				// all this logic to the reconciler if we stored the parameters in the
				// `PlanExecution`.
				// See https://github.com/kudobuilder/kudo/issues/422
				if new.Status.ActivePlan.Name == "" {
					log.Printf("InstanceController: Old and new spec matched...\n %+v ?= %+v\n", old.Spec, new.Spec)
					planName = "deploy"
					planFound = true
				}
			}

			if planFound {
				log.Printf("InstanceController: Going to run plan \"%v\" for instance %v", planName, new.Name)
				// Suspend the the current plan.
				current := &kudov1alpha1.PlanExecution{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{Name: new.Status.ActivePlan.Name, Namespace: new.Status.ActivePlan.Namespace}, current)
				if err != nil {
					log.Printf("InstanceController: Ignoring error when getting plan for new instance: %v", err)
				} else {
					if current.Status.State == kudov1alpha1.PhaseStateComplete {
						log.Printf("InstanceController: Current plan for instance %v is already done, won't change the Suspend flag.", new.Name)
					} else {
						log.Printf("InstanceController: Suspending the PlanExecution for instance %v", new.Name)
						t := true
						current.Spec.Suspend = &t
						did, err := controllerutil.CreateOrUpdate(context.TODO(), mgr.GetClient(), current, func() error {
							t := true
							current.Spec.Suspend = &t
							return nil
						})
						if err != nil {
							log.Printf("InstanceController: Error suspending PlanExecution for instance %v: %v", new.Name, err)
						} else {
							log.Printf("InstanceController: Successfully suspended PlanExecution for instance %v. Returned: %v", new.Name, did)
						}
					}
				}

				if err = createPlan(mgr, planName, new); err != nil {
					log.Printf("InstanceController: Error creating PlanExecution \"%v\" for instance \"%v\": %v", planName, new.Name, err)
				}
			}

			// See if there's a current plan being run, if so "cancel" the plan run.
			return e.ObjectOld != e.ObjectNew
		},
		// New Instances should have Deploy called
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf("InstanceController: Received create event for instance \"%v\"", e.Meta.GetName())
			instance := e.Object.(*kudov1alpha1.Instance)

			// Get the instance OperatorVersion object
			ov := &kudov1alpha1.OperatorVersion{}
			err := mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      instance.Spec.OperatorVersion.Name,
					Namespace: instance.GetOperatorVersionNamespace(),
				},
				ov)
			if err != nil {
				log.Printf("InstanceController: Error getting operatorversion \"%v\" for instance \"%v\": %v",
					instance.Spec.OperatorVersion.Name,
					instance.Name,
					err)
				// TODO: We probably want to handle this differently and mark this instance as unhealthy
				// since its linking to a bad OV.
				return false
			}

			planName := "deploy"

			if _, ok := ov.Spec.Plans[planName]; !ok {
				log.Printf("InstanceController: Could not find deploy plan \"%v\" for instance \"%v\"", planName, instance.Name)
				return false
			}

			err = createPlan(mgr, planName, instance)
			if err != nil {
				log.Printf("InstanceController: Error creating PlanExecution \"%v\" for instance \"%v\": %v", planName, instance.Name, err)
			}
			return err == nil
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			log.Printf("InstanceController: Received delete event for instance \"%v\"", e.Meta.GetName())
			return true
		},
	}
}

func createPlan(mgr manager.Manager, planName string, instance *kudov1alpha1.Instance) error {
	gvk, _ := apiutil.GVKForObject(instance, mgr.GetScheme())
	recorder := mgr.GetEventRecorderFor("instance-controller")
	log.Printf("Creating PlanExecution of plan %s for instance %s", planName, instance.Name)
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
			// TODO: Should also add one for Operator in here as well.
			Labels: map[string]string{
				kudo.OperatorVersionAnnotation: instance.Spec.OperatorVersion.Name,
				kudo.InstanceLabel:             instance.Name,
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
	log.Printf("Created PlanExecution of plan %s for instance %s", planName, instance.Name)
	recorder.Event(instance, "Normal", "PlanCreated", fmt.Sprintf("PlanExecution \"%v\" created", planExecution.Name))
	return nil
}

var _ reconcile.Reconciler = &ReconcileInstance{}

// ReconcileInstance reconciles an Instance object.
type ReconcileInstance struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Instance object and makes changes based on the state read
// and what is in the Instance.Spec.
//
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.dev,resources=instances,verbs=get;list;watch;create;update;patch;delete
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

	log.Printf("InstanceController: Received Reconcile request for instance \"%+v\"", request.Name)

	// Make sure the OperatorVersion is present
	ov := &kudov1alpha1.OperatorVersion{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.OperatorVersion.Name,
			Namespace: instance.GetOperatorVersionNamespace(),
		},
		ov)
	if err != nil {
		log.Printf("InstanceController: Error getting operatorversion \"%v\" for instance \"%v\": %v",
			instance.Spec.OperatorVersion.Name,
			instance.Name,
			err)
		r.recorder.Event(instance, "Warning", "InvalidOperatorVersion", fmt.Sprintf("Error getting operatorversion \"%v\": %v", ov.Name, err))
		return reconcile.Result{}, err
	}

	// Make sure all the required parameters in the operatorVersion are present
	for _, param := range ov.Spec.Parameters {
		if param.Required && param.Default == nil {
			if _, ok := instance.Spec.Parameters[param.Name]; !ok {
				r.recorder.Event(instance, "Warning", "MissingParameter", fmt.Sprintf("Missing parameter \"%v\" required by operatorversion \"%v\"", param.Name, ov.Name))
			}
		}
	}

	// Defer call from above should apply the status changes to the object
	return reconcile.Result{}, nil
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
