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
	"log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles an Instance object.
type Reconciler struct {
	client.Client
	recorder record.EventRecorder
}

// SetupWithManager registers this reconciler with the controller manager
func (r *Reconciler) SetupWithManager(
	mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kudov1alpha1.Instance{}).
		Owns(&kudov1alpha1.Instance{}).
		Owns(&appsv1.Deployment{}).
		Owns(&batchv1.Job{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

// Reconcile ...
//
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.dev,resources=instances,verbs=get;list;watch;create;update;patch;delete
func (r *Reconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// ---------- 1. Query the current state ----------

	log.Printf("InstanceController: Received Reconcile request for instance \"%+v\"", request.Name)
	instance, err := r.getInstance(request)
	if err != nil {
		return reconcile.Result{}, err
	}
	// query OV

	// ---------- 2. If we're currently running plan, continue with the execution ----------

	// ---------- 3. Update status of instance after the execution proceeded ----------

	return reconcile.Result{}, nil
}

// getInstance retrieves the instance by namespaced name
// returns nil, nil when instance is not found (not found is not considered an error)
func (r *Reconciler) getInstance(request ctrl.Request) (instance *kudov1alpha1.Instance, err error) {
	instance = &kudov1alpha1.Instance{}
	err = r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}
	return instance, nil
}