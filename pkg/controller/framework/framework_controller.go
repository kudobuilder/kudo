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

package framework

import (
	"context"
	"fmt"
	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Framework Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	log.Printf("Registering framework.\n")
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	apiextensionsClient, err := apiextensionsclientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}
	return &ReconcileFramework{Client: mgr.GetClient(), scheme: mgr.GetScheme(), apiextensionsClient: apiextensionsClient}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("framework-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Framework
	err = c.Watch(&source.Kind{Type: &maestrov1alpha1.Framework{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileFramework{}

// ReconcileFramework reconciles a Framework object
type ReconcileFramework struct {
	apiextensionsClient *apiextensionsclientset.Clientset
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Framework object and makes changes based on the state read
// and what is in the Framework.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maestro.k8s.io,resources=frameworks,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileFramework) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Framework instance
	instance := &maestrov1alpha1.Framework{}
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

	// Maybe we just make this a standard Instance CRD instead of dynamic CRDs
	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafkas.packages.maestro.k8s.io",
			Namespace: instance.Namespace,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "packages.maestro.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Plural:   "kafkas",
				Singular: "kafka",
				ListKind: "KafkaList",
				Kind:     "Kafka",
			},
			Scope: v1beta1.NamespaceScoped,
		},
	}

	if err := controllerutil.SetControllerReference(instance, crd, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found, err := r.apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Kafka",
			APIVersion: "v1alpha1",
		},
	})

	if err != nil && runtime.IsNotRegisteredError(err) && errors.IsNotFound(err) {
		log.Printf("Creating CRD %s/%s\n", crd.Namespace, crd.Name)
		_, err = r.apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if !reflect.DeepEqual(crd.Spec, found.Spec) {
		fmt.Printf("crd: %+v\n", crd.Spec)
		fmt.Printf("found: %+v\n", found.Spec)
		found.Spec = crd.Spec
		log.Printf("Updating CRD %s/%s\n", crd.Namespace, crd.Name)
		_, err = r.apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Update(crd)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
