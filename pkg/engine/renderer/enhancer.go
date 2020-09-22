package renderer

import (
	"fmt"
	"strings"

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
	Apply(objs []runtime.Object, metadata Metadata) ([]runtime.Object, error)
}

// DefaultEnhancer is implementation of Enhancer that applies the defined conventions by directly editing runtime.Objects (Unstructured).
type DefaultEnhancer struct {
	Scheme    *runtime.Scheme
	Client    client.Client
	Discovery discovery.CachedDiscoveryInterface
}

// Apply accepts templates to be rendered in kubernetes and enhances them with our own KUDO conventions
// These include the way we name our objects and what labels we apply to them
func (de *DefaultEnhancer) Apply(sourceObjs []runtime.Object, metadata Metadata) ([]runtime.Object, error) {
	unstructuredObjs := make([]*unstructured.Unstructured, 0, len(sourceObjs))

	for _, obj := range sourceObjs {
		unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("%wconverting to unstructured failed: %v", engine.ErrFatalExecution, err)
		}

		objUnstructured := &unstructured.Unstructured{Object: unstructMap}

		if err = addLabels(objUnstructured, metadata); err != nil {
			return nil, fmt.Errorf("%wadding labels on parsed object: %v", engine.ErrFatalExecution, err)
		}
		if err = addAnnotations(objUnstructured, metadata); err != nil {
			return nil, fmt.Errorf("%wadding annotations on parsed object %s: %v", engine.ErrFatalExecution, obj.GetObjectKind(), err)
		}

		isNamespaced, err := resource.IsNamespacedObject(obj, de.Discovery)
		if err != nil {
			return nil, fmt.Errorf("%wfailed to determine if object %s is namespaced: %v", engine.ErrFatalExecution, obj.GetObjectKind(), err)
		}

		// Note: Cross-namespace owner references are disallowed by design. This means:
		// 1) Namespace-scoped dependents can only specify owners in the same namespace, and owners that are cluster-scoped.
		// 2) Cluster-scoped dependents can only specify cluster-scoped owners, but not namespace-scoped owners.
		// More: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
		if isNamespaced {
			objUnstructured.SetNamespace(metadata.InstanceNamespace)
			if err := controllerutil.SetControllerReference(metadata.ResourcesOwner, objUnstructured, de.Scheme); err != nil {
				return nil, fmt.Errorf("%wsetting controller reference on parsed object %s: %v", engine.ErrFatalExecution, obj.GetObjectKind(), err)
			}
		}

		unstructuredObjs = append(unstructuredObjs, objUnstructured)
	}

	if err := de.addDependenciesHashes(unstructuredObjs); err != nil {
		return nil, fmt.Errorf("failed to add dependencies hash: %v", err)
	}

	// This is pretty important, if we don't convert it to the actual type everything will be Unstructured.
	// We depend on types later on in the processing e.g. when evaluating health.
	objs, err := de.convertToTyped(unstructuredObjs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert objects to typed: %v", err)
	}

	return objs, nil
}

func (de *DefaultEnhancer) addDependenciesHashes(unstructuredObjs []*unstructured.Unstructured) error {
	dc := newDependencyCalculator(de.Client, unstructuredObjs)
	for _, uo := range unstructuredObjs {
		deps, err := calculateResourceDependencies(uo)
		if err != nil {
			return fmt.Errorf("failed to calculate resource dependencies for %s/%s: %v", uo.GetNamespace(), uo.GetName(), err)
		}
		if !deps.empty() {
			err = dc.calculateAndSetHash(uo, deps)
			if err != nil {
				return fmt.Errorf("failed to add dependency hash to %s/%s: %v", uo.GetNamespace(), uo.GetName(), err)
			}
		}
	}
	return nil
}

func (de *DefaultEnhancer) convertToTyped(unstructuredObjs []*unstructured.Unstructured) ([]runtime.Object, error) {
	objs := make([]runtime.Object, 0, len(unstructuredObjs))
	for _, uo := range unstructuredObjs {
		obj, err := de.Scheme.New(uo.GroupVersionKind())
		if err != nil {
			objs = append(objs, uo)
			continue
		}

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(uo.UnstructuredContent(), obj)
		if err != nil {
			return nil, fmt.Errorf("%wconverting from unstructured failed: %v", engine.ErrFatalExecution, err)
		}

		objs = append(objs, obj)
	}
	return objs, nil
}

func addLabels(obj *unstructured.Unstructured, metadata Metadata) error {
	fieldsToAdd := map[string]string{
		kudo.HeritageLabel: "kudo",
		kudo.OperatorLabel: metadata.OperatorName,
		kudo.InstanceLabel: metadata.InstanceName,
	}

	gvk := obj.GroupVersionKind()
	for _, lp := range CommonLabelPaths {
		if lp.matches(gvk) {
			if err := addMapValues(obj.UnstructuredContent(), fieldsToAdd, lp.pathFields()...); err != nil {
				return err
			}
		}
	}

	return nil
}

func addAnnotations(obj *unstructured.Unstructured, metadata Metadata) error {
	// For all pod template specs, we only add the operator version annotation. It is pretty stable
	// and shouldn't change often, therefore not trigger an unwanted restart of the created pod
	fieldsToAdd := map[string]string{
		kudo.OperatorVersionAnnotation: metadata.OperatorVersion,
	}
	gvk := obj.GroupVersionKind()
	for _, lp := range TemplateAnnotationPaths {
		if lp.matches(gvk) {
			if err := addMapValues(obj.UnstructuredContent(), fieldsToAdd, lp.pathFields()...); err != nil {
				return err
			}
		}
	}

	// The plan, phase and step annotations are only added to the top level resources, not any pod template specs, as
	// that may lead to unwanted restarts of the pods
	topLevelFieldsToAdd := map[string]string{
		kudo.PlanAnnotation:    metadata.PlanName,
		kudo.PhaseAnnotation:   metadata.PhaseName,
		kudo.StepAnnotation:    metadata.StepName,
		kudo.PlanUIDAnnotation: string(metadata.PlanUID),
	}
	for _, lp := range CommonAnnotationPaths {
		if lp.matches(gvk) {
			if err := addMapValues(obj.UnstructuredContent(), topLevelFieldsToAdd, lp.pathFields()...); err != nil {
				return err
			}
		}
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
