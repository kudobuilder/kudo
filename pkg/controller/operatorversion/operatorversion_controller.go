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

package operatorversion

import (
	"context"
	"log"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new OperatorVersion Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	log.Printf("OperatorVersionController: Registering operatorversion controller.")
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileOperatorVersion{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("operatorversion-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to OperatorVersion
	err = c.Watch(&source.Kind{Type: &kudov1alpha1.OperatorVersion{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileOperatorVersion{}

// ReconcileOperatorVersion reconciles a OperatorVersion object
type ReconcileOperatorVersion struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a OperatorVersion object and makes changes based on the state read
// and what is in the OperatorVersion.Spec.
//
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.k8s.io,resources=operatorversions,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileOperatorVersion) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the operator version
	operatorVersion := &kudov1alpha1.OperatorVersion{}
	err := r.Get(context.TODO(), request.NamespacedName, operatorVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log.Printf("OperatorVersionController: Received Reconcile request for a operatorVersion named: %v", request.Name)

	// TODO: Validate OperatorVersion is appropriate.
	return reconcile.Result{}, nil
}
