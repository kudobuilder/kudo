package renderer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"reflect"
	"strings"

	"k8s.io/api/batch/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/resource"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// Enhancer takes your kubernetes template and kudo related Metadata and applies them to all resources in form of labels
// and annotations
// it also takes care of setting an owner of all the resources to the provided object
type Enhancer interface {
	Apply(templates map[string]string, metadata Metadata) ([]runtime.Object, error)
}

// DefaultEnhancer is implementation of Enhancer that applies the defined conventions by directly editing runtime.Objects (Unstructured).
type DefaultEnhancer struct {
	Scheme    *runtime.Scheme
	Client    client.Client
	Discovery discovery.CachedDiscoveryInterface
}

type resourceDependencies struct {
	secrets    []string
	configMaps []string
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

// Apply accepts templates to be rendered in kubernetes and enhances them with our own KUDO conventions
// These include the way we name our objects and what labels we apply to them
func (de *DefaultEnhancer) Apply(templates map[string]string, metadata Metadata) (objsToAdd []runtime.Object, err error) {
	objs := make([]runtime.Object, 0, len(templates))

	for name, v := range templates {
		parsed, err := YamlToObject(v)
		if err != nil {
			return nil, fmt.Errorf("%wparsing YAML from %s: %v", engine.ErrFatalExecution, name, err)
		}
		for _, obj := range parsed {
			unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				return nil, fmt.Errorf("%wconverting to unstructured failed: %v", engine.ErrFatalExecution, err)
			}

			if err = addLabels(unstructMap, metadata); err != nil {
				return nil, fmt.Errorf("%wadding labels on parsed object: %v", engine.ErrFatalExecution, err)
			}
			if err = addAnnotations(unstructMap, metadata); err != nil {
				return nil, fmt.Errorf("%wadding annotations on parsed object %s: %v", engine.ErrFatalExecution, obj.GetObjectKind(), err)
			}

			objUnstructured := &unstructured.Unstructured{Object: unstructMap}

			isNamespaced, err := resource.IsNamespacedObject(obj, de.Discovery)
			if err != nil {
				return nil, fmt.Errorf("failed to determine if object %s is namespaced: %v", obj.GetObjectKind(), err)
			}

			// Note: Cross-namespace owner references are disallowed by design. This means:
			// 1) Namespace-scoped dependents can only specify owners in the same namespace, and owners that are cluster-scoped.
			// 2) Cluster-scoped dependents can only specify cluster-scoped owners, but not namespace-scoped owners.
			// More: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
			if isNamespaced {
				objUnstructured.SetNamespace(metadata.InstanceNamespace)
				if err = setControllerReference(metadata.ResourcesOwner, objUnstructured, de.Scheme); err != nil {
					return nil, fmt.Errorf("%wsetting controller reference on parsed object %s: %v", engine.ErrFatalExecution, obj.GetObjectKind(), err)
				}
			}

			// This is pretty important, if we don't convert it back to the original type everything will be Unstructured.
			// We depend on types later on in the processing e.g. when evaluating health.
			// Additionally, as we add annotations and labels to all possible paths, this step gets rid of anything
			// that doesn't belong to the specific object type.
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(objUnstructured.UnstructuredContent(), obj)
			if err != nil {
				return nil, fmt.Errorf("%wconverting from unstructured failed: %v", engine.ErrFatalExecution, err)
			}
			objs = append(objs, obj)
		}
	}

	cache := map[reflect.Type]map[string][32]byte{}

	for _, obj := range objs {
		typedObj, deps := calculateResourceDependencies(obj)
		if typedObj != nil {
			err = de.addDependenciesHash(typedObj, deps, metadata, objs, cache)
			if err != nil {
				return nil, fmt.Errorf("failed to add dependency hash")
			}
			log.Printf("Outside of add: %+v\n", typedObj)
		}
	}

	return objs, nil
}

func (de *DefaultEnhancer) addDependenciesHash(obj metav1.Object, deps resourceDependencies, metadata Metadata, objs []runtime.Object, cache map[reflect.Type]map[string][32]byte) error {
	log.Printf("Enhancer: Add dependencies hash for %+v: %+v\n", obj, deps)
	depHash := sha256.New()

	secretType := reflect.TypeOf(&corev1.Secret{})
	if _, ok := cache[secretType]; !ok {
		cache[secretType] = map[string][32]byte{}
	}
	for _, secretName := range deps.secrets {
		hash, err := de.calcDependencyHash(secretName, metadata.InstanceNamespace, objs, cache[secretType], secretType)
		if err != nil {
			return fmt.Errorf("error calculating hash for secret: %v", err)
		}
		_, _ = depHash.Write(hash[:])
	}

	configMapType := reflect.TypeOf(&corev1.ConfigMap{})
	if _, ok := cache[configMapType]; !ok {
		cache[configMapType] = map[string][32]byte{}
	}
	for _, cmName := range deps.configMaps {
		hash, err := de.calcDependencyHash(cmName, metadata.InstanceNamespace, objs, cache[configMapType], configMapType)
		if err != nil {
			return fmt.Errorf("error calculating hash for configMap: %v", err)
		}
		_, _ = depHash.Write(hash[:])
	}

	hashStr := fmt.Sprintf("%x", depHash.Sum([]byte{}))

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

	//obj.GetAnnotations()[kudo.DependenciesHashAnnotation] = hashStr
	log.Printf("Enhancer: Added hash: %s\n", obj.GetAnnotations()[kudo.DependenciesHashAnnotation])
	return nil
}

func (de *DefaultEnhancer) calcDependencyHash(name string, namespace string, objs []runtime.Object, cache map[string][32]byte, t reflect.Type) ([32]byte, error) {
	hash, ok := cache[name]
	if !ok {
		dep, err := de.resourceDependency(name, namespace, t, objs)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to get dependeny %s/%s: %v", namespace, name, err)
		}
		if _, ok := dep.GetAnnotations()[kudo.SkipHashCalculationAnnotation]; ok {
			cache[name] = [32]byte{}
		} else {
			yamlStr, err := ToYaml(dep)
			if err != nil {
				return [32]byte{}, fmt.Errorf("failed to serialize dependeny %s/%s: %v", namespace, name, err)
			}
			hash = sha256.Sum256([]byte(yamlStr))
			cache[name] = hash
		}
	}
	return hash, nil
}

// resourceDependency returns the resource of type t with the given namespace/name, either from the passed in list of objects or the last applied configuration from the API server
func (de *DefaultEnhancer) resourceDependency(name string, namespace string, t reflect.Type, objs []runtime.Object) (metav1.Object, error) {
	fmt.Printf("get resource Dependency %s/%s with type %s\n", namespace, name, t)
	for _, obj := range objs {
		fmt.Printf("Check existing object type: %s vs %s\n", reflect.TypeOf(obj), t)
		if reflect.TypeOf(obj) == t {
			obj, _ := obj.(metav1.Object)
			fmt.Printf("Check existing object name: %s vs %s\n", obj.GetName(), name)
			if obj.GetName() == name {
				return obj, nil
			}
		}
	}
	dep, _ := reflect.New(t).Elem().Interface().(metav1.Object)
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	err := de.Client.Get(context.TODO(), key, dep.(runtime.Object))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve TODO")
	}
	lastConfiguration, ok := dep.GetAnnotations()[kudo.LastAppliedConfigAnnotation]
	if !ok {
		return nil, fmt.Errorf("LastAppliedConfigAnnotation is not available")
	}

	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, []byte(lastConfiguration))
	if err != nil {
		return nil, fmt.Errorf("failed to decode lastAppliedConfigAnnotation")
	}
	return obj.(metav1.Object), err
}

// Calculates the resource dependencies of the passed in object. This are currently config maps and secrets
func calculateResourceDependencies(obj runtime.Object) (metav1.Object, resourceDependencies) {
	deps := resourceDependencies{}
	switch obj := (obj).(type) {
	case *appsv1.StatefulSet:
		deps.addSecrets(obj.Spec.Template.Spec.ImagePullSecrets)
		for _, v := range obj.Spec.Template.Spec.Volumes {
			deps.addConfigMapVolumeSource(v.ConfigMap)
			deps.addSecretVolumeSource(v.Secret)
		}
		return obj, deps
	case *appsv1.Deployment:
		deps.addSecrets(obj.Spec.Template.Spec.ImagePullSecrets)
		for _, v := range obj.Spec.Template.Spec.Volumes {
			deps.addConfigMapVolumeSource(v.ConfigMap)
			deps.addSecretVolumeSource(v.Secret)
		}
		return obj, deps
	}
	return nil, resourceDependencies{}
}

func addLabels(obj map[string]interface{}, metadata Metadata) error {
	// List of paths for labels from here:
	// https://github.com/kubernetes-sigs/kustomize/blob/master/api/konfig/builtinpluginconsts/commonlabels.go
	labelPaths := [][]string{
		{"metadata", "labels"},
		{"spec", "template", "metadata", "labels"},
		{"spec", "volumeClaimTemplates[]", "metadata", "labels"},
		{"spec", "jobTemplate", "metadata", "labels"},
		{"spec", "jobTemplate", "spec", "template", "metadata", "labels"},
	}

	fieldsToAdd := map[string]string{
		kudo.HeritageLabel: "kudo",
		kudo.OperatorLabel: metadata.OperatorName,
		kudo.InstanceLabel: metadata.InstanceName,
	}

	for _, path := range labelPaths {
		if err := addMapValues(obj, fieldsToAdd, path...); err != nil {
			return err
		}
	}

	return nil
}

func addAnnotations(obj map[string]interface{}, metadata Metadata) error {
	// List of paths for annotations from here:
	// https://github.com/kubernetes-sigs/kustomize/blob/master/api/konfig/builtinpluginconsts/commonannotations.go
	annotationPaths := [][]string{
		{"metadata", "annotations"},
		{"spec", "template", "metadata", "annotations"},
		{"spec", "jobTemplate", "metadata", "annotations"},
		{"spec", "jobTemplate", "spec", "template", "metadata", "annotations"},
	}

	fieldsToAdd := map[string]string{
		kudo.OperatorVersionAnnotation: metadata.OperatorVersion,
	}

	for _, path := range annotationPaths {
		if err := addMapValues(obj, fieldsToAdd, path...); err != nil {
			return err
		}
	}

	topLevelFieldsToAdd := map[string]string{
		kudo.PlanAnnotation:  metadata.PlanName,
		kudo.PhaseAnnotation: metadata.PhaseName,
		kudo.StepAnnotation:  metadata.StepName,
	}
	if err := addMapValues(obj, topLevelFieldsToAdd, "metadata", "annotations"); err != nil {
		return err
	}

	return nil
}

func addMapValues(obj map[string]interface{}, fieldsToAdd map[string]string, path ...string) error {
	for i, p := range path {
		// If we have an element with a slice in the path, apply the fields to all elements of the
		// slice with the remaining path
		if strings.HasSuffix(p, "[]") {
			sliceField := strings.TrimSuffix(p, "[]")

			subPath := append(path[0:i], sliceField)
			remainingPath := path[i+1:]

			unstructuredSlice, found, err := unstructured.NestedSlice(obj, subPath...)
			if !found || err != nil {
				// We don't return err here, as it just means that path is invalid for this object.
				// This is ok and does not indicate an error
				return nil
			}
			for _, s := range unstructuredSlice {
				if sliceMap, ok := s.(map[string]interface{}); ok {
					if err = addMapValues(sliceMap, fieldsToAdd, remainingPath...); err != nil {
						return err
					}
				}
			}
			if err = unstructured.SetNestedSlice(obj, unstructuredSlice, subPath...); err != nil {
				return err
			}

			return nil
		}
	}

	// Merge added fields to map at specified path
	stringMap, _, err := unstructured.NestedStringMap(obj, path...)
	if err != nil {
		// We don't return err here, as it just means that path is invalid for this object.
		// This is ok and does not indicate an error
		return nil
	}
	if stringMap == nil {
		stringMap = make(map[string]string)
	}
	for k, v := range fieldsToAdd {
		stringMap[k] = v
	}
	return unstructured.SetNestedStringMap(obj, stringMap, path...)
}

func setControllerReference(owner metav1.Object, object metav1.Object, scheme *runtime.Scheme) error {
	ownerNs := owner.GetNamespace()
	if ownerNs != "" {
		objNs := object.GetNamespace()
		if objNs == "" {
			// we're trying to create cluster-scoped resource from and bind Instance as owner of that
			// that is disallowed by design, see https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents
			// for now solve by not adding the owner
			log.Printf("Not adding owner to resource %s because it's cluster-scoped and cannot be owned by namespace-scoped instance %s/%s", object.GetName(), owner.GetNamespace(), owner.GetName())
			return nil
		}
		if ownerNs != objNs {
			// we're trying to create resource in another namespace as is Instance's namespace, Instance cannot be owner of such resource
			// that is disallowed by design, see https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents
			// for now solve by not adding the owner
			log.Printf("Not adding owner to resource %s/%s because it's in different namespace than instance %s/%s and thus cannot be owned by that instance", object.GetNamespace(), object.GetName(), owner.GetNamespace(), owner.GetName())
			return nil
		}
	}
	if err := controllerutil.SetControllerReference(owner, object, scheme); err != nil {
		return err
	}
	return nil
}
