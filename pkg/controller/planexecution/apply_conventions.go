package planexecution

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/kudobuilder/kudo/pkg/util/template"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
)

const basePath = "/kustomize"

// metadata contains metadata associated with current PlanExecution
type metadata struct {
	InstanceName    string
	Namespace       string
	OperatorName    string
	OperatorVersion string
	PlanExecution   string
	PlanName        string
	PhaseName       string
	StepName        string
}

// ApplyConventions accepts templates to be rendered in kubernetes and enhances them with our own KUDO conventions
// These include the way we name our objects and what labels we apply to them
func applyConventionsToTemplates(templates map[string]string, metadata metadata) ([]runtime.Object, error) {
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
		Namespace:  metadata.Namespace,
		CommonLabels: map[string]string{
			"heritage": "kudo",
			"app":      metadata.OperatorName,
			"version":  metadata.OperatorVersion,
			"instance": metadata.InstanceName,
		},
		CommonAnnotations: map[string]string{
			"planexecution": metadata.PlanExecution,
			"plan":          metadata.PlanName,
			"phase":         metadata.PhaseName,
			"step":          metadata.StepName,
		},
		GeneratorOptions: &ktypes.GeneratorOptions{
			DisableNameSuffixHash: true,
		},
		Resources:             templateNames,
		PatchesStrategicMerge: []patch.StrategicMerge{},
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
	defer ldr.Cleanup()

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

	objsToAdd, err := template.ParseKubernetesObjects(string(res))
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing kubernetes objects after applying kustomize")
	}

	return objsToAdd, nil
}
