package renderer

import (
	"fmt"
	"log"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	Scheme *runtime.Scheme
}

// Apply accepts templates to be rendered in kubernetes and enhances them with our own KUDO conventions
// These include the way we name our objects and what labels we apply to them
func (k *DefaultEnhancer) Apply(templates map[string]string, metadata Metadata) (objsToAdd []runtime.Object, err error) {
	objs := make([]runtime.Object, 0, len(templates))

	for _, v := range templates {
		parsed, err := YamlToObject(v)
		if err != nil {
			return nil, err
		}
		for _, obj := range parsed {
			unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				return nil, err
			}

			objUnstructured := &unstructured.Unstructured{Object: unstructMap}
			objUnstructured.SetNamespace(metadata.InstanceNamespace)

			if err = addLabels(objUnstructured.Object, metadata); err != nil {
				return nil, fmt.Errorf("adding labels on parsed object: %v", err)
			}
			if err = addAnnotations(objUnstructured.Object, metadata); err != nil {
				return nil, fmt.Errorf("adding annotations on parsed object: %v", err)
			}
			if err = setControllerReference(metadata.ResourcesOwner, objUnstructured, k.Scheme); err != nil {
				return nil, fmt.Errorf("setting controller reference on parsed object: %v", err)
			}

			// this is pretty important, if we don't convert it back to the original type everything will be Unstructured
			// we depend on types later on in the processing e.g. when evaluating health
			// additionally, as we add annotations and labels to all possible paths, this step gets rid of anything
			// that doesn't belong to the specific object type
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(objUnstructured.UnstructuredContent(), obj)
			if err != nil {
				return nil, err
			}
			objs = append(objs, obj)
		}
	}

	return objs, nil
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
		if err := applyMapValues(obj, fieldsToAdd, path...); err != nil {
			return err
		}
	}

	return nil
}

func addAnnotations(obj map[string]interface{}, metadata Metadata) error {
	// List of pathsfor annotations from here:
	// https://github.com/kubernetes-sigs/kustomize/blob/master/api/konfig/builtinpluginconsts/commonannotations.go
	annotationPaths := [][]string{
		{"metadata", "annotations"},
		{"spec", "template", "metadata", "annotations"},
		{"spec", "jobTemplate", "metadata", "annotations"},
		{"spec", "jobTemplate", "spec", "template", "metadata", "annotations"},
	}

	fieldsToAdd := map[string]string{
		kudo.PlanAnnotation:            metadata.PlanName,
		kudo.PhaseAnnotation:           metadata.PhaseName,
		kudo.StepAnnotation:            metadata.StepName,
		kudo.OperatorVersionAnnotation: metadata.OperatorVersion,
		kudo.PlanUIDAnnotation:         string(metadata.PlanUID),
	}

	for _, path := range annotationPaths {
		if err := applyMapValues(obj, fieldsToAdd, path...); err != nil {
			return err
		}
	}

	return nil
}

func applyMapValues(obj map[string]interface{}, fieldsToAdd map[string]string, path ...string) error {
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
					if err = applyMapValues(sliceMap, fieldsToAdd, remainingPath...); err != nil {
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

func setControllerReference(owner v1.Object, object v1.Object, scheme *runtime.Scheme) error {
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
