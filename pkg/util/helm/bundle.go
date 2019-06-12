package helm

import (
	"fmt"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/bundle"

	"github.com/helm/helm/pkg/chartutil"
)

type HelmChart struct {
	ChartFile []byte
	Templates map[string]string
	Values []byte
}

// ToBundle converts the helm chart on disk into a Kudo Bundle
func ToBundle(helmChart *HelmChart) (bundle.Framework, error) {
	b := bundle.Framework{}

	meta, err := chartutil.UnmarshalChartfile(helmChart.ChartFile)
	if err != nil {
		return b, err
	}
	b.Name = meta.GetName()
	b.Description = meta.GetDescription()
	b.Version = meta.GetVersion()
	b.KUDOVersion = MinHelmVersion
	b.KubernetesVersion = meta.GetKubeVersion()
	b.Maintainers = make([]kudo.Maintainer, 0)
	for _, m := range meta.Maintainers {
		b.Maintainers = append(b.Maintainers, kudo.Maintainer(fmt.Sprintf("%v <%v>", m.Name, m.Email)))
	}
	b.URL = meta.GetHome()

	b.Tasks = make(map[string]kudo.TaskSpec)
	b.Plans = make(map[string]kudo.Plan)
	b.Parameters = make(map[string]bundle.Parameter)

	//tasks
	resources := make([]string, 0)
	for k := range helmChart.Templates {
		resources = append(resources, k)
	}
	b.Tasks["deploy"] = kudo.TaskSpec{
		Resources: resources,
	}
	//plans
	b.Plans["deploy"] = kudo.Plan{
		Strategy: kudo.Parallel,
		Phases: []kudo.Phase{
			kudo.Phase{
				Name:     "deploy",
				Strategy: kudo.Parallel,
				Steps: []kudo.Step{
					kudo.Step{
						Name:  "deploy",
						Tasks: resources,
					},
				},
			},
		},
	}

	//parameters
	params, err := loadParameters(helmChart.Values)
	if err != nil {
		return b, err
	}
	mapParams := make(map[string]bundle.Parameter)
	for _, param := range params {
		mapParams[param.Name] = bundle.Parameter{
			Default:     param.Default,
			Description: param.Description,
			Trigger:     param.Trigger,
		}
	}
	b.Parameters = mapParams

	//dependencies
	//TODO(@runyontr)
	return b, nil
}
