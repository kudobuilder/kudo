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

type dependencyCalculator struct {
	Client    client.Client
	namespace string
	objs      []runtime.Object
	cache     map[reflect.Type]map[string]hashBytes
}

type resourceDependencies struct {
	secrets    []string
	configMaps []string
}

type hashBytes [16]byte

func (rd *resourceDependencies) addFromPodTemplateSpec(SpecTemplate corev1.PodTemplateSpec) {
	rd.addSecrets(SpecTemplate.Spec.ImagePullSecrets)
	for _, v := range SpecTemplate.Spec.Volumes {
		rd.addConfigMapVolumeSource(v.ConfigMap)
		rd.addSecretVolumeSource(v.Secret)
	}
}

func (rd *resourceDependencies) addSecrets(secrets []corev1.LocalObjectReference) {
	for _, s := range secrets {
		rd.secrets = append(rd.secrets, s.Name)
	}
}

func (rd *resourceDependencies) addSecretVolumeSource(svs *corev1.SecretVolumeSource) {
	if svs != nil {
		rd.secrets = append(rd.secrets, svs.SecretName)
	}
}

func (rd *resourceDependencies) addConfigMapVolumeSource(cmvs *corev1.ConfigMapVolumeSource) {
	if cmvs != nil {
		rd.configMaps = append(rd.configMaps, cmvs.Name)
	}
}

func (de *dependencyCalculator) addDependenciesHash(obj metav1.Object, deps resourceDependencies) error {
	log.Printf("Enhancer: Add dependencies hash for %s/%s: %+v\n", obj.GetNamespace(), obj.GetName(), deps)
	depHash := md5.New() //nolint:gosec

	for _, name := range deps.secrets {
		hash, err := de.calcDependencyHash(name, reflect.TypeOf(&corev1.Secret{}))
		if err != nil {
			return fmt.Errorf("error calculating hash for secret: %v", err)
		}
		log.Printf("Calculated hash %x for secret %s\n", hash, name)
		_, _ = depHash.Write(hash[:])
	}

	for _, name := range deps.configMaps {
		hash, err := de.calcDependencyHash(name, reflect.TypeOf(&corev1.ConfigMap{}))
		if err != nil {
			return fmt.Errorf("error calculating hash for configMap: %v", err)
		}
		log.Printf("Calculated hash %x for config map %s\n", hash, name)
		_, _ = depHash.Write(hash[:])
	}

	hashStr := fmt.Sprintf("%x", depHash.Sum([]byte{}))

	setTemplateHash(obj, hashStr)

	//obj.GetAnnotations()[kudo.DependenciesHashAnnotation] = hashStr
	log.Printf("Enhancer: Added hash: %s\n", obj.GetAnnotations()[kudo.DependenciesHashAnnotation])
	return nil
}

func (de *dependencyCalculator) calcDependencyHash(name string, t reflect.Type) (hashBytes, error) {
	cache, ok := de.cache[t]
	if !ok {
		cache = map[string]hashBytes{}
		de.cache[t] = cache
	}

	hash, ok := cache[name]
	if !ok {
		dep, err := de.resourceDependency(name, t)
		if err != nil {
			return hashBytes{}, fmt.Errorf("failed to get dependeny %s/%s: %v", de.namespace, name, err)
		}
		if _, ok := dep.GetAnnotations()[kudo.SkipHashCalculationAnnotation]; ok {
			de.cache[t][name] = hashBytes{}
		} else {
			ns := dep.GetNamespace()
			or := dep.GetOwnerReferences()
			dep.SetNamespace("")
			dep.SetOwnerReferences([]metav1.OwnerReference{})
			yamlStr, err := ToYaml(dep)
			dep.SetNamespace(ns)
			dep.SetOwnerReferences(or)
			if err != nil {
				return hashBytes{}, fmt.Errorf("failed to serialize dependeny %s/%s: %v", de.namespace, name, err)
			}
			hash = md5.Sum([]byte(yamlStr)) //nolint:gosec
			log.Printf("Calculated Hash %x from %s\n", hash, yamlStr)
			cache[name] = hash
		}
	}
	return hash, nil
}

// resourceDependency returns the resource of type t with the given namespace/name, either from the passed in list of objects or the last applied configuration from the API server
func (de *dependencyCalculator) resourceDependency(name string, t reflect.Type) (metav1.Object, error) {

	// First try to find the dependency in the local list, if it's deployed in the same task we'll find it here
	for _, obj := range de.objs {
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
		return nil, fmt.Errorf("failed to decode lastAppliedConfigAnnotation from %s/%s", de.namespace, name)
	}
	return obj.(metav1.Object), err
}

// Calculates the resource dependencies of the passed in object. This are currently config maps and secrets
func calculateResourceDependencies(obj runtime.Object) (metav1.Object, resourceDependencies) {
	deps := resourceDependencies{}
	switch obj := (obj).(type) {
	case *appsv1.StatefulSet:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
		return obj, deps
	case *appsv1.Deployment:
		deps.addFromPodTemplateSpec(obj.Spec.Template)
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

func setTemplateHash(obj metav1.Object, hashStr string) {
	switch obj := obj.(type) {
	case *appsv1.StatefulSet:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *appsv1.Deployment:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *batchv1.Job:
		obj.Spec.Template.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *corev1.Pod:
		obj.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	case *v1beta1.CronJob:
		obj.Spec.JobTemplate.Annotations[kudo.DependenciesHashAnnotation] = hashStr
	}
}
