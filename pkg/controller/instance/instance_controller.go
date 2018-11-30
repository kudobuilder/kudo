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
	"github.com/kubernetes-sigs/kubebuilder-maestro/pkg/util/template"
	"gopkg.in/yaml.v2"
	"log"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"

	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by Instance - change this for objects you create
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &maestrov1alpha1.Instance{},
	})
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

	framework := &maestrov1alpha1.Framework{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Name:      frameworkVersion.Spec.Framework.Name,
		Namespace: "default", // Namespaced names on cluster resources are in the default namespace
	}, framework)

	if err != nil {
		log.Printf("InstanceController: Could not find Framework with name %v: %v\n", frameworkVersion.Spec.Framework.Name, err)
		return reconcile.Result{}, err
	}

	//Create configmap to hold all parameters for instantiation
	configs := make(map[string]string)
	//Default parameters from instance metadata
	//parameters from instance spec
	configs["FRAMEWORK_NAME"] = framework.Name
	configs["NAME"] = instance.Name
	configs["NAMESPACE"] = instance.Namespace

	for k, v := range instance.Spec.Parameters {
		configs[k] = v
	}

	for k, v := range frameworkVersion.Spec.Defaults {
		_, ok := configs[k]
		if !ok { //not specified in params
			configs[k] = v
		}
	}

	var resources []string
	fsys := fs.MakeFakeFS()
	for k, v := range frameworkVersion.Spec.Templates {
		templatedYaml, err := template.ExpandMustache(v, configs)
		if err != nil {
			log.Printf("InstanceController: Error expanding mustache: %v\n", err)
			return reconcile.Result{}, err
		}

		fsys.WriteFile(fmt.Sprintf("%s/%s", basePath, k), []byte(*templatedYaml))
		resources = append(resources, k)
	}

	// write kustomize.yaml...
	kustomization := &ktypes.Kustomization{
		NamePrefix: instance.Name + "-",
		Namespace:  instance.Namespace,
		CommonLabels: map[string]string{
			"heritage": "maestro",
			"app":      framework.Name,
			"instance": instance.Name,
		},
		Resources:             resources,
		PatchesStrategicMerge: []patch.StrategicMerge{},
	}
	yamlBytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return reconcile.Result{}, err
	}
	fsys.WriteFile(fmt.Sprintf("%s/kustomization.yaml", basePath), yamlBytes)

	ldr, err := loader.NewLoader("/kustomize", fsys)
	if err != nil {
		return reconcile.Result{}, err
	}

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()))
	kt, err := target.NewKustTarget(ldr, fsys, rf, transformer.NewFactoryImpl())
	if err != nil {
		return reconcile.Result{}, err
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return reconcile.Result{}, err
	}

	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return reconcile.Result{}, err
	}

	objs, err := template.ParseKubernetesObjects(string(res))
	if err != nil {

		log.Printf("InstanceController: Error parsing yaml into k8s objects: %v\n", err)
		return reconcile.Result{}, err
	}

	for _, obj := range objs {
		err = r.ApplyObject(obj, instance)
		if err != nil {
			log.Printf("InstanceController: Error applying object: %v\n", obj)
			log.Printf("InstanceController: Error: %v\n", err)
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

//ApplyObject takes the object provided and either creates or updates it depending on whether the object
// exixts or not
func (r *ReconcileInstance) ApplyObject(obj runtime.Object, parent metav1.Object) error {
	nnn, _ := client.ObjectKeyFromObject(obj)
	switch o := obj.(type) {
	//Service
	case *corev1.Service:
		svc := &corev1.Service{}
		err := r.Get(context.TODO(), nnn, svc)
		if err != nil && errors.IsNotFound(err) {
			svc = obj.(*corev1.Service)
			if err := controllerutil.SetControllerReference(parent, svc, r.scheme); err != nil {
				return err
			}
			err = r.Create(context.TODO(), svc)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
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
			return err
		}
	case *appsv1.StatefulSet:
		ss := &appsv1.StatefulSet{}
		err := r.Get(context.TODO(), nnn, ss)
		if err != nil && errors.IsNotFound(err) {
			ss = obj.(*appsv1.StatefulSet)
			if err := controllerutil.SetControllerReference(parent, ss, r.scheme); err != nil {
				return err
			}
			err = r.Create(context.TODO(), ss)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			ss.Spec = obj.(*appsv1.StatefulSet).Spec
			ss.Labels = obj.(*appsv1.StatefulSet).Labels
			ss.Annotations = obj.(*appsv1.StatefulSet).Annotations
			err = r.Update(context.TODO(), ss)
		}
		if err != nil {
			return err
		}
	case *policyv1beta1.PodDisruptionBudget:
		pdb := &policyv1beta1.PodDisruptionBudget{}
		err := r.Get(context.TODO(), nnn, pdb)
		if err != nil && errors.IsNotFound(err) {
			pdb = obj.(*policyv1beta1.PodDisruptionBudget)
			if err := controllerutil.SetControllerReference(parent, pdb, r.scheme); err != nil {
				return err
			}
			err = r.Create(context.TODO(), pdb)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			pdb.Spec = obj.(*policyv1beta1.PodDisruptionBudget).Spec
			pdb.Labels = obj.(*policyv1beta1.PodDisruptionBudget).Labels
			pdb.Annotations = obj.(*policyv1beta1.PodDisruptionBudget).Annotations
			err = r.Update(context.TODO(), pdb)
		}
		if err != nil {
			return err
		}
	case *corev1.ConfigMap:
		cm := &corev1.ConfigMap{}
		err := r.Get(context.TODO(), nnn, cm)
		if err != nil && errors.IsNotFound(err) {
			cm = obj.(*corev1.ConfigMap)
			if err := controllerutil.SetControllerReference(parent, cm, r.scheme); err != nil {
				return err
			}
			err = r.Create(context.TODO(), cm)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			cm.Data = obj.(*corev1.ConfigMap).Data
			cm.Labels = obj.(*corev1.ConfigMap).Labels
			cm.Annotations = obj.(*corev1.ConfigMap).Annotations
			err = r.Update(context.TODO(), cm)
		}
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("I dont know how to update types %v.  Please implement", o)

	}
	return nil
}
