package renderer

import (
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		parsed, err := YamlToObject(string(v))
		if err != nil {
			return nil, err
		}
		for _, obj := range parsed {
			unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				return nil, err
			}
			objUnstructured := &unstructured.Unstructured{Object: unstructMap}

			labels := objUnstructured.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			annotations := objUnstructured.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}

			labels[kudo.HeritageLabel] = "kudo"
			labels[kudo.OperatorLabel] = metadata.OperatorName
			labels[kudo.InstanceLabel] = metadata.InstanceName

			annotations[kudo.PlanAnnotation] = metadata.PlanName
			annotations[kudo.PhaseAnnotation] = metadata.PhaseName
			annotations[kudo.StepAnnotation] = metadata.StepName
			annotations[kudo.OperatorVersionAnnotation] = metadata.OperatorVersion
			annotations[kudo.PlanUIDAnnotation] = string(metadata.PlanUID)

			objUnstructured.SetNamespace(metadata.InstanceNamespace)
			objUnstructured.SetLabels(labels)
			objUnstructured.SetAnnotations(annotations)

			err = setControllerReference(metadata.ResourcesOwner, objUnstructured, k.Scheme)
			if err != nil {
				return nil, fmt.Errorf("setting controller reference on parsed object: %v", err)
			}

			// this is pretty important, if we don't convert it back to the original type everything will be Unstructured
			// we depend on types later on in the processing e.g. when evaluating health
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(objUnstructured.UnstructuredContent(), obj)
			if err != nil {
				return nil, err
			}
			objs = append(objs, obj)
		}
	}

	return objs, nil
}

func setControllerReference(owner v1.Object, object *unstructured.Unstructured, scheme *runtime.Scheme) error {
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
