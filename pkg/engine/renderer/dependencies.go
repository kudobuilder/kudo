package renderer

import (
	"context"
	"crypto/md5" //nolint:gosec
	"fmt"
	"log"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

type hashBytes [16]byte

type dependencyCalculator struct {
	// Used to retrieve the current version of dependencies if they are not in the taskObjects list
	Client client.Client
	// The namespace to which the current task deploys
	namespace string
	// The resources that are deployed in the task
	taskObjects []runtime.Object
	// A simple cache that stores hashes of dependencies, in case they are used by multiple resources
	cache map[reflect.Type]map[string]hashBytes
}

func newDependencyCalculator(client client.Client, namespace string, taskObjects []runtime.Object) dependencyCalculator {
	c := dependencyCalculator{
		Client:      client,
		namespace:   namespace,
		taskObjects: taskObjects,
		cache:       map[reflect.Type]map[string]hashBytes{},
	}
	c.cache[typeSecret] = map[string]hashBytes{}
	c.cache[typeConfigMap] = map[string]hashBytes{}
	return c
}

var (
	// The types of dependencies we support - need to be pointer to obj type
	typeSecret    = reflect.TypeOf(&corev1.Secret{})
	typeConfigMap = reflect.TypeOf(&corev1.ConfigMap{})

	// We need a list of types to ensure correct order of hash calculation
	dependencyTypes = []reflect.Type{
		typeSecret,
		typeConfigMap,
	}
)

type resourceDependencies map[reflect.Type][]string

func newDependencies() resourceDependencies {
	deps := map[reflect.Type][]string{}

	for _, t := range dependencyTypes {
		deps[t] = []string{}
	}

	return deps
}

// Adds all dependencies from a pod template spec
func (rd resourceDependencies) addFromPodTemplateSpec(SpecTemplate corev1.PodTemplateSpec) {
	for _, s := range SpecTemplate.Spec.ImagePullSecrets {
		rd[typeSecret] = append(rd[typeSecret], s.Name)
	}
	for _, v := range SpecTemplate.Spec.Volumes {
		if v.ConfigMap != nil {
			rd[typeConfigMap] = append(rd[typeConfigMap], v.ConfigMap.Name)
		}
		if v.Secret != nil {
			rd[typeSecret] = append(rd[typeSecret], v.Secret.SecretName)
		}
	}
}

// Adds a hash calculated from the dependencies to embedded pod template specs
func (de *dependencyCalculator) calculateAndSetHash(obj metav1.Object, deps resourceDependencies) error {
	log.Printf("Enhancer: Add dependencies hash for %s/%s: %+v\n", obj.GetNamespace(), obj.GetName(), deps)

	depHash := md5.New() //nolint:gosec
	for _, depType := range dependencyTypes {
		for _, name := range deps[depType] {
			hash, err := de.getHashForDependency(name, depType)
			if err != nil {
				return fmt.Errorf("error calculating hash for %s of type %s: %v", name, depType, err)
			}
			_, _ = depHash.Write(hash[:])
		}
	}

	hashStr := fmt.Sprintf("%x", depHash.Sum([]byte{}))

	return setTemplateHash(obj, hashStr)
}

func (de *dependencyCalculator) getHashForDependency(name string, t reflect.Type) (hashBytes, error) {
	cache := de.cache[t]

	if hash, ok := cache[name]; ok {
		return hash, nil
	}

	dep, err := de.resourceDependency(name, t)
	if err != nil {
		return hashBytes{}, fmt.Errorf("failed to get dependeny %s/%s: %v", de.namespace, name, err)
	}
	if _, ok := dep.GetAnnotations()[kudo.SkipHashCalculationAnnotation]; ok {
		cache[name] = hashBytes{}
	} else {
		yamlStr, err := calculateResourceHash(dep)
		if err != nil {
			return hashBytes{}, fmt.Errorf("failed to serialize dependeny %s/%s: %v", de.namespace, name, err)
		}
		cache[name] = md5.Sum([]byte(yamlStr)) //nolint:gosec
	}

	return cache[name], nil
}

// Calculates a stable hash from a resource
func calculateResourceHash(obj metav1.Object) (string, error) {
	// Namespace is ignored mostly to allow easier integration tests (which use random namespaces)
	ns := obj.GetNamespace()

	// OwnerReferences need to be skipped as they contain a changing UID
	or := obj.GetOwnerReferences()

	// Annotations are notorious for containing quickly changing strings: plan/phase/task names, uids, dates, etc.
	ann := obj.GetAnnotations()

	obj.SetNamespace("")
	obj.SetOwnerReferences([]metav1.OwnerReference{})
	obj.SetAnnotations(map[string]string{})
	yamlStr, err := ToYaml(obj)
	obj.SetNamespace(ns)
	obj.SetOwnerReferences(or)
	obj.SetAnnotations(ann)
	return yamlStr, err
}

// resourceDependency returns the resource of type t with the given namespace/name, either from the passed in list of objects or the last applied configuration from the API server
func (de *dependencyCalculator) resourceDependency(name string, t reflect.Type) (metav1.Object, error) {

	// First try to find the dependency in the local list, if it's deployed in the same task we'll find it here
	for _, obj := range de.taskObjects {
		log.Printf("Check Task Object %s - %s: %v", reflect.TypeOf(obj), t, obj)
		if reflect.TypeOf(obj) == t {
			obj, _ := obj.(metav1.Object)
			if obj.GetName() == name {
				return obj, nil
			}
		}
	}

	// We haven't found it, so we need to query the api server to get the current version
	dep, _ := reflect.New(t).Elem().Interface().(metav1.Object)
	key := client.ObjectKey{
		Namespace: de.namespace,
		Name:      name,
	}

	err := de.Client.Get(context.TODO(), key, dep.(runtime.Object))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve object %s/%s: %v", de.namespace, name, err)
	}

	// We don't want the hash from the object itself, because of added metadata from the api-server
	// We use the LastAppliedConfigAnnotation that stores exactly what we applied last time
	lastConfiguration, ok := dep.GetAnnotations()[kudo.LastAppliedConfigAnnotation]
	if !ok {
		return nil, fmt.Errorf("LastAppliedConfigAnnotation is not available on %s/%s", de.namespace, name)
	}

	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, []byte(lastConfiguration))
	if err != nil {
		return nil, fmt.Errorf("failed to decode lastAppliedConfigAnnotation from %s/%s: %v", de.namespace, name, err)
	}
	return obj.(metav1.Object), nil
}

// Calculates the resource dependencies of the passed in object
func calculateResourceDependencies(obj runtime.Object) (metav1.Object, resourceDependencies) {
	deps := newDependencies()

	switch obj := (obj).(type) {
	case *appsv1.StatefulSet:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
		return obj, deps
	case *appsv1.Deployment:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
		return obj, deps
	case *appsv1.DaemonSet:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
		return obj, deps
	case *appsv1.ReplicaSet:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
		return obj, deps
	case *corev1.ReplicationController:
		deps.addFromPodTemplateSpec(*obj.Spec.Template)
		return obj, deps
	case *batchv1.Job:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
		return obj, deps
	case *v1beta1.CronJob:
		deps.addFromPodTemplateSpec(obj.Spec.JobTemplate.Spec.Template)
		return obj, deps
	}
	return nil, resourceDependencies{}
}

// Sets the given hash in the pod template spec of the obj
func setTemplateHash(obj metav1.Object, hashStr string) error {
	switch obj := obj.(type) {
	case *appsv1.StatefulSet:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *appsv1.Deployment:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *appsv1.DaemonSet:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *appsv1.ReplicaSet:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *corev1.ReplicationController:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *batchv1.Job:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *v1beta1.CronJob:
		obj.Spec.JobTemplate.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	default:
		return fmt.Errorf("unknown object type to set dependencies hash: %s", reflect.TypeOf(obj))
	}
	return nil
}
