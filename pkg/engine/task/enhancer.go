package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"

	"github.com/kudobuilder/kudo/pkg/util/template"
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

// KubernetesObjectEnhancer takes your kubernetes template and kudo related Metadata and applies them to all resources in form of labels
// and annotations
// it also takes care of setting an owner of all the resources to the provided object
type KubernetesObjectEnhancer interface {
	ApplyConventionsToTemplates(templates map[string]string, metadata Metadata) ([]runtime.Object, error)
}

// KustomizeEnhancer is implementation of KubernetesObjectEnhancer that uses kustomize to apply the defined conventions
type KustomizeEnhancer struct {
	Scheme *runtime.Scheme
}

// ApplyConventionsToTemplates accepts templates to be rendered in kubernetes and enhances them with our own KUDO conventions
// These include the way we name our objects and what labels we apply to them
func (k *KustomizeEnhancer) ApplyConventionsToTemplates(templates map[string]string, metadata Metadata) (objsToAdd []runtime.Object, err error) {
	fsys := fs.MakeFakeFS()

	templateNames := make([]string, 0, len(templates))

	for k, v := range templates {
		templateNames = append(templateNames, k)
		err := fsys.WriteFile(fmt.Sprintf("%s/%s", basePath, k), []byte(v))
		if err != nil {
			return nil, errors.Wrapf(err, "error when writing templates to filesystem before applying kustomize")
		}
	}

	kustomization := &ktypes.Kustomization{
		NamePrefix: metadata.InstanceName + "-",
		Namespace:  metadata.InstanceNamespace,
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
		},
		GeneratorOptions: &ktypes.GeneratorOptions{
			DisableNameSuffixHash: true,
		},
		Resources:             templateNames,
		PatchesStrategicMerge: []apipatch.StrategicMerge{},
	}

	yamlBytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshalling kustomize yaml")
	}

	err = fsys.WriteFile(fmt.Sprintf("%s/kustomization.yaml", basePath), yamlBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "error writing kustomization.yaml file")
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
		return nil, errors.Wrapf(err, "error creating kustomize target")
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return nil, errors.Wrapf(err, "error creating customized resource map for kustomize")
	}

	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return nil, errors.Wrapf(err, "error encoding kustomized files into yaml")
	}

	objsToAdd, err = template.ParseKubernetesObjects(string(res))
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing kubernetes objects after applying kustomize")
	}

	for _, o := range objsToAdd {
		err = setControllerReference(metadata.ResourcesOwner, o, k.Scheme)
		if err != nil {
			return nil, errors.Wrapf(err, "setting controller reference on parsed object")
		}
	}

	return objsToAdd, nil
}

func setControllerReference(owner v1.Object, obj runtime.Object, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(owner, obj.(v1.Object), scheme); err != nil {
		return err
	}
	return nil
}
