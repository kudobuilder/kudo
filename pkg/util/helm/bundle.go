package helm

import (
	"fmt"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/bundle"

	"github.com/helm/helm/pkg/chartutil"
)

func ToBundle(folder string) (bundle.Framework, error) {
	b := bundle.Framework{}

	meta, err := chartutil.LoadChartfile(folder + "/Chart.yaml")
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
	b.Parameters = make([]kudo.Parameter, 0)

	//tasks
	tasks, err := loadTemplates(folder)
	if err != nil {
		return b, err
	}

	resources := make([]string, 0)
	for k := range tasks {
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
	params, err := loadParameters(folder)
	if err != nil {
		return b, err
	}
	b.Parameters = params

	//dependencies
	//TODO(@runyontr)
	return b, nil
}
