package renderer

import (
	"context"
	"crypto/md5" //nolint:gosec
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/runtime/schema"

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
	// The resources that are deployed in the task
	taskObjects []*unstructured.Unstructured
	// A simple cache that stores hashes of dependencies, in case they are used by multiple resources
	// The cache is only valid during one call to enhancer apply, i.e one task execution. The cache
	// is discarded after the task execution is completed
	cache map[resourceDependency]hashBytes
}

func newDependencyCalculator(client client.Client, taskObjects []*unstructured.Unstructured) dependencyCalculator {
	c := dependencyCalculator{
		Client:      client,
		taskObjects: taskObjects,
		cache:       map[resourceDependency]hashBytes{},
	}
	return c
}

var (
	// The types of dependencies we support
	typeSecret    = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	typeConfigMap = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
)

type resourceDependency struct {
	gvk       schema.GroupVersionKind
	name      string
	namespace string
}
type resourceDependencies []resourceDependency

func (rd resourceDependencies) empty() bool { return len(rd) == 0 }

// Len returns the number of dependencies
// This is needed to allow sorting.
func (rd resourceDependencies) Len() int { return len(rd) }

// Swap swaps the position of two items in the dependencies slice.
// This is needed to allow sorting.
func (rd resourceDependencies) Swap(i, j int) { rd[i], rd[j] = rd[j], rd[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// This is needed to allow sorting.
func (rd resourceDependencies) Less(x, y int) bool {
	if rd[x].gvk.Group != rd[y].gvk.Group {
		return rd[x].gvk.Group < rd[y].gvk.Group
	}
	if rd[x].gvk.Kind != rd[y].gvk.Kind {
		return rd[x].gvk.Kind < rd[y].gvk.Kind
	}
	if rd[x].gvk.Version != rd[y].gvk.Version {
		return rd[x].gvk.Version < rd[y].gvk.Version
	}
	if rd[x].namespace != rd[y].namespace {
		return rd[x].namespace < rd[y].namespace
	}
	return rd[x].name < rd[y].name
}

// addFromEmbeddedPodTemplateSpec adds all dependencies from a possible embedded pod template spec at the given path
// If the path does not exist in the unstructred object, no dependency is added and no error is returned
func (rd *resourceDependencies) addFromEmbeddedPodTemplateSpec(obj *unstructured.Unstructured, fields ...string) error {
	podTemplateData, ok, err := unstructured.NestedMap(obj.UnstructuredContent(), fields...)
	if err != nil {
		return fmt.Errorf("failed to get embedded pod template spec: %v", err)
	}
	if !ok {
		return nil
	}
	ns := obj.GetNamespace()
	podTemplate := &corev1.PodTemplateSpec{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(podTemplateData, podTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse pod template spec: %v", err)
	}

	for _, s := range podTemplate.Spec.ImagePullSecrets {
		*rd = append(*rd, resourceDependency{gvk: typeSecret, name: s.Name, namespace: ns})
	}
	for _, v := range podTemplate.Spec.Volumes {
		if v.ConfigMap != nil {
			*rd = append(*rd, resourceDependency{gvk: typeConfigMap, name: v.ConfigMap.Name, namespace: ns})
		}
		if v.Secret != nil {
			*rd = append(*rd, resourceDependency{gvk: typeSecret, name: v.Secret.SecretName, namespace: ns})
		}
	}
	return nil
}

// calculateAndSetHash adds a hash calculated from the dependencies to embedded pod template specs
func (de *dependencyCalculator) calculateAndSetHash(obj *unstructured.Unstructured, deps resourceDependencies) error {

	depHash := md5.New() //nolint:gosec
	sort.Sort(deps)
	for _, dep := range deps {
		hash, err := de.getHashForDependency(dep)
		if err != nil {
			return fmt.Errorf("error calculating hash for %s of type %s: %v", dep.name, dep.gvk, err)
		}
		_, _ = depHash.Write(hash[:]) // Hash.Write never returns an error
	}

	hashStr := fmt.Sprintf("%x", depHash.Sum([]byte{}))

	return setTemplateHash(obj, hashStr)
}

func (de *dependencyCalculator) getHashForDependency(d resourceDependency) (hashBytes, error) {
	if hash, ok := de.cache[d]; ok {
		return hash, nil
	}

	dep, err := de.resourceDependency(d)
	if err != nil {
		return hashBytes{}, fmt.Errorf("failed to get dependeny %s/%s: %v", d.namespace, d.name, err)
	}
	if _, ok := dep.GetAnnotations()[kudo.SkipHashCalculationAnnotation]; ok {
		de.cache[d] = hashBytes{}
	} else {
		yamlStr, err := sanitizeAndSerialize(dep)
		if err != nil {
			return hashBytes{}, fmt.Errorf("failed to serialize dependeny %s/%s: %v", d.namespace, d.name, err)
		}
		de.cache[d] = md5.Sum([]byte(yamlStr)) //nolint:gosec
	}

	return de.cache[d], nil
}

// sanitizeAndSerialize removes volatile parts of an object and returns the resulting object as serialized yaml
func sanitizeAndSerialize(origObj *unstructured.Unstructured) (string, error) {
	obj := origObj.DeepCopy()

	// Namespace is ignored mostly to allow easier integration tests (which use random namespaces)
	obj.SetNamespace("")

	// OwnerReferences need to be skipped as they contain a changing UID
	obj.SetOwnerReferences([]metav1.OwnerReference{})

	// Annotations are notorious for containing quickly changing strings: plan/phase/task names, uids, dates, etc.
	obj.SetAnnotations(map[string]string{})

	return ToYaml(obj)
}

// resourceDependency returns the resource of type t with the given namespace/name, either from the passed in list of objects or the last applied configuration from the API server
func (de *dependencyCalculator) resourceDependency(d resourceDependency) (*unstructured.Unstructured, error) {

	// First try to find the dependency in the local list, if it's deployed in the same task we'll find it here
	for _, obj := range de.taskObjects {
		if obj.GetObjectKind().GroupVersionKind() == d.gvk {
			if obj.GetName() == d.name && obj.GetNamespace() == d.namespace {
				return obj, nil
			}
		}
	}

	// We haven't found it, so we need to query the api server to get the current version
	//dep, _ := reflect.New(t).Interface().(metav1.Object)
	dep := &unstructured.Unstructured{}
	dep.SetGroupVersionKind(d.gvk)

	key := client.ObjectKey{
		Namespace: d.namespace,
		Name:      d.name,
	}

	err := de.Client.Get(context.TODO(), key, dep)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve object %s/%s: %v", d.namespace, d.name, err)
	}

	// We don't want the hash from the object itself, because of added metadata from the api-server
	// We use the LastAppliedConfigAnnotation that stores exactly what we applied last time
	lastConfiguration, ok := dep.GetAnnotations()[kudo.LastAppliedConfigAnnotation]
	if !ok {
		return nil, fmt.Errorf("LastAppliedConfigAnnotation is not available on %s/%s", d.namespace, d.name)
	}

	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, []byte(lastConfiguration))
	if err != nil {
		return nil, fmt.Errorf("failed to decode lastAppliedConfigAnnotation from %s/%s: %v", d.namespace, d.name, err)
	}

	return obj.(*unstructured.Unstructured), nil
}

// calculateResourceDependencies gets the resource dependencies of the passed in object
func calculateResourceDependencies(obj *unstructured.Unstructured) (resourceDependencies, error) {
	deps := resourceDependencies{}

	// We can't rely on actual types here, so we just look into each possible path and try to find a pod template spec

	if err := deps.addFromEmbeddedPodTemplateSpec(obj, "spec", "template"); err != nil {
		return nil, err
	}
	if err := deps.addFromEmbeddedPodTemplateSpec(obj, "spec", "jobTemplate", "spec", "template"); err != nil {
		return nil, err
	}

	return deps, nil
}

// setTemplateHash adds the given hash in the pod template spec of the obj
func setTemplateHash(obj *unstructured.Unstructured, hashStr string) error {
	fieldsToAdd := map[string]string{
		kudo.DependenciesHashAnnotation: hashStr,
	}

	gvk := obj.GroupVersionKind()
	for _, lp := range TemplateAnnotationPaths {
		if lp.matches(gvk) {
			if err := addMapValues(obj.UnstructuredContent(), fieldsToAdd, lp.pathFields()...); err != nil {
				return err
			}
		}
	}

	return nil
}
