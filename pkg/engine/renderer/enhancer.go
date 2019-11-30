package renderer

import (
	"fmt"
	"log"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	apipatch "sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
)

const basePath = "/kustomize"

// Enhancer takes your kubernetes template and kudo related Metadata and applies them to all resources in form of labels
// and annotations
// it also takes care of setting an owner of all the resources to the provided object
type Enhancer interface {
	Apply(templates map[string]string, metadata Metadata) ([]runtime.Object, error)
}

// KustomizeEnhancer is implementation of Enhancer that uses kustomize to apply the defined conventions
type KustomizeEnhancer struct {
	Scheme *runtime.Scheme
}

// Apply accepts templates to be rendered in kubernetes and enhances them with our own KUDO conventions
// These include the way we name our objects and what labels we apply to them
func (k *KustomizeEnhancer) Apply(templates map[string]string, metadata Metadata) (objsToAdd []runtime.Object, err error) {
	fsys := fs.MakeFakeFS()

	templateNames := make([]string, 0, len(templates))

	for k, v := range templates {
		templateNames = append(templateNames, k)
		err := fsys.WriteFile(fmt.Sprintf("%s/%s", basePath, k), []byte(v))
		if err != nil {
			return nil, fmt.Errorf("error when writing templates to filesystem before applying kustomize: %w", err)
		}
	}

	kustomization := &ktypes.Kustomization{
		Namespace: metadata.InstanceNamespace,
		CommonLabels: map[string]string{
			kudo.HeritageLabel: "kudo",
			kudo.OperatorLabel: metadata.OperatorName,
			kudo.InstanceLabel: metadata.InstanceName,
		},
		CommonAnnotations: map[string]string{
			kudo.PlanAnnotation:            metadata.PlanName,
			kudo.PhaseAnnotation:           metadata.PhaseName,
			kudo.StepAnnotation:            metadata.StepName,
			kudo.OperatorVersionAnnotation: metadata.OperatorVersion,
			kudo.PlanUIDAnnotation:         string(metadata.PlanUID),
		},
		GeneratorOptions: &ktypes.GeneratorOptions{
			DisableNameSuffixHash: true,
		},
		Resources:             templateNames,
		PatchesStrategicMerge: []apipatch.StrategicMerge{},
	}

	yamlBytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, fmt.Errorf("error marshalling kustomize yaml: %w", err)
	}

	err = fsys.WriteFile(fmt.Sprintf("%s/kustomization.yaml", basePath), yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("error writing kustomization.yaml file: %w", err)
	}

	ldr, err := loader.NewLoader(basePath, fsys)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ferr := ldr.Cleanup(); ferr != nil {
			err = ferr
		}
	}()

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()))
	kt, err := target.NewKustTarget(ldr, rf, transformer.NewFactoryImpl())
	if err != nil {
		return nil, fmt.Errorf("error creating kustomize target: %w", err)
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return nil, fmt.Errorf("error creating customized resource map for kustomize: %w", err)
	}

	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return nil, fmt.Errorf("error encoding kustomized files into yaml: %w", err)
	}

	objsToAdd, err = YamlToObject(string(res))
	if err != nil {
		return nil, fmt.Errorf("error parsing kubernetes objects after applying kustomize: %w", err)
	}

	for _, o := range objsToAdd {
		err = setControllerReference(metadata.ResourcesOwner, o, k.Scheme)
		if err != nil {
			return nil, fmt.Errorf("setting controller reference on parsed object: %w", err)
		}
	}

	return objsToAdd, nil
}

func setControllerReference(owner v1.Object, obj runtime.Object, scheme *runtime.Scheme) error {
	object := obj.(v1.Object)
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
