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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"log"
	"reflect"
	"time"

	maestrov1alpha1 "github.com/maestrosdk/maestro/pkg/apis/maestro/v1alpha1"

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

const basePath = "/kustomize"

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
	return &ReconcileInstance{Client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetRecorder("instance-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("instance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			old := e.ObjectOld.(*maestrov1alpha1.Instance)
			new := e.ObjectNew.(*maestrov1alpha1.Instance)

			//Haven't done anything yet
			if new.Status.ActivePlan.Name == "" {
				err = createPlan(mgr, "deploy", new)
				if err != nil {
					log.Printf("Error creating %v object for %v: %v\n", "deploy", new.Name, err)
				}
				return true
			}

			//get the new FrameworkVersion object
			fv := &maestrov1alpha1.FrameworkVersion{}
			err = mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      new.Spec.FrameworkVersion.Name,
					Namespace: new.Spec.FrameworkVersion.Namespace,
				},
				fv)
			if err != nil {
				log.Printf("Error getting FrameworkVersion %v for instance %v: %v\n",
					new.Spec.FrameworkVersion.Name,
					new.Name,
					err)
				//TODO
				//We probably want to handle this differently and mark this instance as unhealthy
				//since its linking to a bad FV
				return false
			}
			var planName string
			var ok bool
			if old.Spec.FrameworkVersion != new.Spec.FrameworkVersion {
				//Its an Upgrade!
				_, ok = fv.Spec.Plans["upgrade"]
				if !ok {
					_, ok = fv.Spec.Plans["update"]
					if !ok {
						_, ok = fv.Spec.Plans["deploy"]
						if !ok {
							log.Println("Could not find any plan to use for upgrade")
							return false
						}
						ok = true
						planName = "deploy"
					} else {
						planName = "update"
					}
				} else {
					planName = "upgrade"
				}
			} else if !reflect.DeepEqual(old.Spec, new.Spec) {
				_, ok = fv.Spec.Plans["update"]
				if !ok {
					_, ok = fv.Spec.Plans["deploy"]
					if !ok {
						log.Println("could not find any plan to use for update")
					} else {
						planName = "deploy"
					}
				} else {
					planName = "update"
				}
			} else {
				log.Println("Old and new spec matched...")
			}

			log.Printf("InstanceController: UpdateInstance: Going to call plan %v\n", planName)
			//we found something
			if ok {

				//mark the current plan as Suspend,
				current := &maestrov1alpha1.PlanExecution{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{Name: new.Status.ActivePlan.Name, Namespace: new.Status.ActivePlan.Namespace}, current)
				if err != nil {
					log.Printf("Error getting plan for newInstance... Ignoring")
				} else {
					if current.Status.State == maestrov1alpha1.PhaseStateComplete {
						log.Println("Current Plan for Instance is already done, wont change the Suspend flag")
					} else {
						log.Println("Setting PlanExecution to Suspend")
						t := true
						current.Spec.Suspend = &t
						did, err := controllerutil.CreateOrUpdate(context.TODO(), mgr.GetClient(), current, func(o runtime.Object) error {
							t := true
							o.(*maestrov1alpha1.PlanExecution).Spec.Suspend = &t
							return nil
						})
						if err != nil {
							log.Printf("Error changing the current PlanExecution to Suspend: %v\n", err)
						} else {
							log.Printf("No error in setting PlanExecution.Suspend to true.  Returned %v\n", did)
						}
					}
				}

				err = createPlan(mgr, planName, new)
				if err != nil {
					log.Printf("Error creating %v object for %v: %v\n", planName, new.Name, err)
				}
			}

			//status change?  Sent it along

			//See if there's a current plan being run.
			//if so "cancel" the plan run
			//create a new plan
			return e.ObjectOld != e.ObjectNew
		},
		//New Instances should have Deploy called
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf("Recieved create event for %v\n", e.Meta)
			new := e.Object.(*maestrov1alpha1.Instance)

			//get the new FrameworkVersion object
			fv := &maestrov1alpha1.FrameworkVersion{}
			err = mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      new.Spec.FrameworkVersion.Name,
					Namespace: new.Spec.FrameworkVersion.Namespace,
				},
				fv)
			if err != nil {
				log.Printf("Error getting FrameworkVersion %v for instance %v: %v\n",
					new.Spec.FrameworkVersion.Name,
					new.Name,
					err)
				//TODO
				//We probably want to handle this differently and mark this instance as unhealthy
				//since its linking to a bad FV
				return false
			}
			planName := "deploy"
			ok := false
			_, ok = fv.Spec.Plans[planName]
			if !ok {
				log.Println("Could not find deploy plan")
				return false
			}

			err = createPlan(mgr, planName, new)
			if err != nil {
				log.Printf("Error creating %v object for %v: %v\n", planName, new.Name, err)
			}
			return err == nil
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
	}

	// Watch for changes to Instance
	err = c.Watch(&source.Kind{Type: &maestrov1alpha1.Instance{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return err
	}
	return nil
}

func createPlan(mgr manager.Manager, planName string, instance *maestrov1alpha1.Instance) error {
	gvk, _ := apiutil.GVKForObject(instance, mgr.GetScheme())
	recorder := mgr.GetRecorder("instance-controller")
	recorder.Event(instance, "Normal", "CreatePlanExecution", fmt.Sprintf("Creating %v plan execution", planName))

	// Create a new ref
	ref := corev1.ObjectReference{
		Kind:      gvk.Kind,
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}

	planExecution := maestrov1alpha1.PlanExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v-%v", instance.Name, planName, time.Now().Nanosecond()),
			Namespace: instance.GetNamespace(),
			//Should also add one for Framework in here as well.
			Labels: map[string]string{
				"framework-version": instance.Spec.FrameworkVersion.Name,
				"instance":          instance.Name,
			},
		},
		Spec: maestrov1alpha1.PlanExecutionSpec{
			Instance: ref,
			PlanName: planName,
		},
	}
	//Make this instance the owner of the PlanExecution
	controllerutil.SetControllerReference(instance, &planExecution, mgr.GetScheme())
	//new!
	if err := mgr.GetClient().Create(context.TODO(), &planExecution); err != nil {
		recorder.Event(instance, "Warning", "CreatePlanExecution", "Error creating plan execution: %v")
		return err
	}
	recorder.Event(instance, "Normal", "PlanCreated", fmt.Sprintf("PlanExecution %v created", planExecution.Name))
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

	//Make sure the FrameworkVersion is present
	fv := &maestrov1alpha1.FrameworkVersion{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.FrameworkVersion.Name,
			Namespace: instance.Spec.FrameworkVersion.Namespace,
		},
		fv)
	if err != nil {
		log.Printf("Error getting FrameworkVersion %v for instance %v: %v\n",
			instance.Spec.FrameworkVersion.Name,
			instance.Name,
			err)
		r.recorder.Event(instance, "Warning", "InvalidFrameworkVersion", fmt.Sprintf("Error getting FrameworkVersion %v: %v", fv.Name, err))
		return reconcile.Result{}, err
	}

	//make sure all the required parameters in the frameworkversion are present
	for _, param := range fv.Spec.Parameters {
		if param.Required {
			if _, ok := instance.Spec.Parameters[param.Name]; !ok {
				r.recorder.Event(instance, "Warning", "MissingParameter", fmt.Sprintf("Missing parameter %v required by FrameworkVersion %v", param.Name, fv.Name))
			}
		}
	}

	//defer call from above should apply the status changes to the object
	return reconcile.Result{}, nil
}
