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

package component

import (
	"context"
	frameworksv1beta1 "github.com/kudobuilder/kudo/pkg/apis/frameworks/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/dynamic"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller")

// Add creates a new Component Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	rc := &ReconcileComponent{Client: mgr.GetClient(), scheme: mgr.GetScheme(), cr: dynamic.ControllerRegistry{}}

	c := extclient.NewForConfigOrDie(mgr.GetConfig())
	crdList, _ := c.CustomResourceDefinitions().List(metav1.ListOptions{LabelSelector: "heritage=kudo"})

	for _, crd := range crdList.Items {
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    crd.Spec.Names.Kind,
			Group:   crd.Spec.Group,
			Version: crd.Spec.Versions[0].Name,
		})

		rc.cr.Register(u)
	}

	return rc
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("component-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Component
	err = c.Watch(&source.Kind{Type: &frameworksv1beta1.Component{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by Component - change this for objects you create
	err = c.Watch(&source.Kind{Type: &apiextv1beta1.CustomResourceDefinition{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &frameworksv1beta1.Component{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileComponent{}

// ReconcileComponent reconciles a Component object
type ReconcileComponent struct {
	client.Client
	scheme *runtime.Scheme
	cr     dynamic.ControllerRegistry
}

// Reconcile reads that state of the cluster for a Component object and makes changes based on the state read
// and what is in the Component.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frameworks.kudo.sh,resources=components,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frameworks.kudo.sh,resources=components/status,verbs=get;update;patch
func (r *ReconcileComponent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Component instance
	instance := &frameworksv1beta1.Component{}
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

	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: instance.Spec.ObjectMeta.Name,
			Labels: map[string]string{
				"heritage": "kudo",
			},
		},
		Spec: instance.Spec.Spec,
	}

	if err := controllerutil.SetControllerReference(instance, crd, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &apiextv1beta1.CustomResourceDefinition{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: crd.Name, Namespace: crd.Namespace}, found)
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    crd.Spec.Names.Kind,
		Group:   crd.Spec.Group,
		Version: crd.Spec.Versions[0].Name,
	})

	dynamicFinalizer := "dynamic.finalizers.kudo.sh"
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(instance.ObjectMeta.Finalizers, dynamicFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, dynamicFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		if containsString(instance.ObjectMeta.Finalizers, dynamicFinalizer) {
			r.cr.Stop(u)
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, dynamicFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}

		return reconcile.Result{}, nil
	}

	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), crd)

		time.Sleep(1 * time.Second)
		r.cr.Register(u)
		return reconcile.Result{}, err
	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
