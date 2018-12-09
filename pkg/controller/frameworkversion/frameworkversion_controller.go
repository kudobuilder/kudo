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

package frameworkversion

import (
	"context"
	"log"

	maestrov1alpha1 "github.com/kubernetes-sigs/maestro/pkg/apis/maestro/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new FrameworkVersion Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this maestro.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileFrameworkVersion{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("frameworkversion-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to FrameworkVersion
	err = c.Watch(&source.Kind{Type: &maestrov1alpha1.FrameworkVersion{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileFrameworkVersion{}

// ReconcileFrameworkVersion reconciles a FrameworkVersion object
type ReconcileFrameworkVersion struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a FrameworkVersion object and makes changes based on the state read
// and what is in the FrameworkVersion.Spec

// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maestro.k8s.io,resources=frameworkversions,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileFrameworkVersion) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the FrameworkVersion instance
	instance := &maestrov1alpha1.FrameworkVersion{}
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

	log.Printf("FrameworkVersionController: Recieved Reconcile request for %v\n", request.Name)
	//When this is changed, we need to Reconcile all instances objects that reference this
	//object.

	//TODO Validate FrameworkVersion is appropriate
	return reconcile.Result{}, nil
}
