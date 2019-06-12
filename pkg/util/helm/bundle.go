package helm

import (
	"fmt"
	"os"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/bundle"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/helm/helm/pkg/chartutil"
)

func LoadChart(path string) (*chart.Chart, error) {
	var chart *chart.Chart
	//check if this is a folder or a file
	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			return chartutil.Load(path)

		} else {
			//folder
			return chartutil.LoadDir(path)
		}
	} else {
		//not a folder, or a file, maybe a chart on a repo?
		chartPath, e := locateChartPath("", path, "")
		if e != nil {
			return chart, e
		}
		return chartutil.Load(chartPath)
	}
}

// ToBundle converts the helm chart on disk into a Kudo Bundle
func ToBundle(chart *chart.Chart) (bundle.Framework, error) {
	b := bundle.Framework{}

	meta := chart.Metadata

	b.Name = meta.GetName()
	b.Description = meta.GetDescription()
	b.Version = meta.GetVersion()
	b.KUDOVersion = MinKUDOVersionToSupportHelm
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
	tasks, err := loadTemplateFromChart(chart)
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
						Tasks: []string{"deploy"},
					},
				},
			},
		},
	}

	//parameters
	params, err := loadParametersFromChart(chart)
	if err != nil {
		return b, err
	}
	b.Parameters = params

	//dependencies
	//TODO(@runyontr)
	return b, nil
}
