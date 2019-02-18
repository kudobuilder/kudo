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
	"fmt"
	"reflect"

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

	// Have the ComponentVersion trigger the Component it references
	//Watch for Deployments, Jobs and StatefulSets
	// Define a mapping from the object in the event to one or more
	// objects to Reconcile.  Specifically this calls for
	// a reconsiliation of any objects "Owner".
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			requests := make([]reconcile.Request, 0)
			cv, ok := a.Object.(*frameworksv1beta1.ComponentVersion)
			if ok {
				compName := cv.Spec.Component
				comp := &frameworksv1beta1.Component{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{
					Name: compName,
				}, comp)
				if err != nil {
					fmt.Printf("Error: ComponentVersion %v references unknown Component %v\n", cv.Name, compName)
					return nil
				} else {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name: compName,
						},
					})
				}
			}
			return requests
		})

	// Watch for changes to ComponentVersion and trigger a Component reconcile
	err = c.Watch(&source.Kind{Type: &frameworksv1beta1.ComponentVersion{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		})
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
			fmt.Printf("Could not find Component %v\n", request.NamespacedName)
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//List all the ComponentVersions and find the ones referencing this component
	cvs := frameworksv1beta1.ComponentVersionList{}
	err = r.List(context.TODO(), &client.ListOptions{}, &cvs)

	dynamicFinalizer := "dynamic.finalizers.kudo.sh"
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		//if the component doesn't have the required finalizer, but isn't flagged for deletion
		//add it
		if !containsString(instance.ObjectMeta.Finalizers, dynamicFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, dynamicFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		//if its marked for deletion and has the finalizer, do some cleanup
		if containsString(instance.ObjectMeta.Finalizers, dynamicFinalizer) {
			for _, cv := range cvs.Items {
				if cv.Spec.Component == instance.Name {
					u := unstructured.Unstructured{}
					u.SetGroupVersionKind(schema.GroupVersionKind{
						Kind:    instance.Spec.Spec.Names.Kind,
						Group:   instance.Spec.Spec.Group,
						Version: cv.Spec.Version,
					})
					err = r.cr.Stop(u)
					if err != nil {
						return reconcile.Result{}, err
					}
				}
			}
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, dynamicFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}

		return reconcile.Result{}, nil
	}

	myCvs := make([]frameworksv1beta1.ComponentVersion, 0)
	for _, cv := range cvs.Items {
		if cv.Spec.Component == instance.Name {
			myCvs = append(myCvs, cv)
		}
	}
	if len(myCvs) == 0 {
		fmt.Printf("Error:  Component %v Created but not ComponentVersion pointing at it\n", instance.Name)
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
	crd.Spec.Versions = make([]apiextv1beta1.CustomResourceDefinitionVersion, 0)
	for i, cv := range myCvs {
		crd.Spec.Versions = append(crd.Spec.Versions,
			apiextv1beta1.CustomResourceDefinitionVersion{
				Name:    cv.Spec.Version,
				Served:  true,
				Storage: i == 0, //Not sure what the right version should be here
				//TODO add these to part of the CV schema
				//Schema *CustomResourceValidation `json:"schema,omitempty" protobuf:"bytes,4,opt,name=schema"`
				//Subresources *CustomResourceSubresources `json:"subresources,omitempty" protobuf:"bytes,5,opt,name=subresources"`
				//AdditionalPrinterColumns []CustomResourceColumnDefinition `json:"additionalPrinterColumns,omitempty" protobuf:"bytes,6,rep,name=additionalPrinterColumns"`
			},
		)
	}

	if err := controllerutil.SetControllerReference(instance, crd, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &apiextv1beta1.CustomResourceDefinition{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: crd.Name, Namespace: crd.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// we need to create the crd.
		err = r.Create(context.TODO(), crd)
		if err != nil {
			log.Error(err, "Error creating crd %v: %+v\n", crd.Name, err)
			return reconcile.Result{}, err
		}
		time.Sleep(1 * time.Second)
		//We only need to watch this once.  If we watch once per version, we end up with multiple events to handle:

		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    crd.Spec.Names.Kind,
			Group:   crd.Spec.Group,
			Version: crd.Spec.Versions[0].Name,
		})
		err = r.cr.Register(u)
		if err != nil {
			fmt.Printf("Error registering GVK (%v,%v,%v): %v\n", crd.Spec.Group, crd.Spec.Names.Kind, crd.Spec.Versions[0].Name, err)
		}
		return reconcile.Result{}, err
	} else if err != nil {
		return reconcile.Result{}, err
	}
	//Now we already have one, so we need to update it with a new FV if applicable
	if !reflect.DeepEqual(found.Spec.Versions, crd.Spec.Versions) {
		found.Spec.Versions = crd.Spec.Versions
		err = r.Update(context.TODO(), found)
		if err != nil {
			fmt.Printf("Error updating CRD %v: %v\n", crd.Name, err)
			return reconcile.Result{}, err
		}
		for _, v := range found.Spec.Versions {
			u := unstructured.Unstructured{}
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Kind:    found.Spec.Names.Kind,
				Group:   found.Spec.Group,
				Version: v.Name,
			})
			err = r.cr.Register(u)
			if err != nil {
				fmt.Printf("Error registering GVK (%v,%v,%v): %v\n", crd.Spec.Group, crd.Spec.Names.Kind, v.Name, err)
			}
		}
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
