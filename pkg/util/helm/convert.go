package helm

/*
  This package can be used to convert an existing helm style project into a Framework and FrameworkVersion for
  importing into KUDO

*/

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

	"github.com/helm/helm/pkg/chartutil"
)

// MinHelmVersion is the minimum KUDO version that supports helm templating
const MinHelmVersion = "0.2.0"

// Import convers the chart at the provided folder into a Framework and FrameworkVersion
func Import(folder string) (kudo.Framework, kudo.FrameworkVersion, error) {

	frameworkVersion := kudo.FrameworkVersion{}
	frameworkVersion.Kind = "FrameworkVersion"
	frameworkVersion.APIVersion = kudo.SchemeGroupVersion.String()

	framework, err := loadMetadata(folder)
	if err != nil {
		return framework, frameworkVersion, err
	}

	templates, err := loadTemplates(folder)
	if err != nil {
		return framework, frameworkVersion, err
	}
	//need to get Version from chart.yaml
	frameworkVersion.Name = framework.Name + "-stable"
	frameworkVersion.Spec.Templates = templates

	//Create the single task
	deployTask := kudo.TaskSpec{
		Resources: make([]string, 0),
	}
	for k := range templates {
		deployTask.Resources = append(deployTask.Resources, k)
	}
	frameworkVersion.Spec.Tasks = make(map[string]kudo.TaskSpec)
	frameworkVersion.Spec.Tasks["deploy"] = deployTask

	//create the single plan
	deployPlan := kudo.Plan{
		Strategy: kudo.Serial,
		Phases: []kudo.Phase{kudo.Phase{
			Name:     "deploy",
			Strategy: kudo.Serial,
			Steps: []kudo.Step{
				kudo.Step{
					Name:   "deploy",
					Tasks:  []string{"deploy"},
					Delete: false,
				},
			},
		}},
	}
	frameworkVersion.Spec.Plans = make(map[string]kudo.Plan)
	frameworkVersion.Spec.Plans["deploy"] = deployPlan

	//parameters
	params, err := loadParameters(folder)
	if err != nil {
		return framework, frameworkVersion, err
	}
	frameworkVersion.Spec.Parameters = params

	frameworkVersion.Spec.Framework.Name = framework.Name

	return framework, frameworkVersion, nil
}

func loadParameters(folder string) ([]kudo.Parameter, error) {
	params := make([]kudo.Parameter, 0)
	values, err := chartutil.ReadValuesFile(folder + "/values.yaml")
	if err != nil {
		return params, err
	}
	params, err = getParametersFromValues(values)
	if err != nil {
		return params, err
	}
	//all the parameters start with a period
	for i := range params {
		params[i].Name = ".Values." + params[i].Name
	}
	return params, err
}

func getParametersFromValues(value chartutil.Values) ([]kudo.Parameter, error) {
	params := make([]kudo.Parameter, 0)
	for k, v := range value {
		s, ok := v.(string)
		if ok {
			params = append(params,
				kudo.Parameter{
					Name:        k, // ???
					Default:     s,
					Description: "Auto import from helm chart", //Maybe look at comments above the line?
				})
			continue
		}
		b, ok := v.(bool)
		if ok {
			params = append(params,
				kudo.Parameter{
					Name:        k, // ???
					Default:     fmt.Sprintf("%v", b),
					Description: "Auto import from helm chart", //Maybe look at comments above the line?
				})
			continue
		}
		i, ok := v.(int)
		if ok {
			params = append(params,
				kudo.Parameter{
					Name:        k, // ???
					Default:     fmt.Sprintf("%v", i),
					Description: "Auto import from helm chart", //Maybe look at comments above the line?
				})
			continue
		}
		//See if table
		tab, e := value.Table(k)
		if e != nil {
			//TODO
			continue
		}
		p, e := getParametersFromValues(tab)
		if e != nil {
			return params, e
		}
		//prefix "k." in front of all the parameter names
		for _, param := range p {
			param.Name = k + "." + param.Name
			params = append(params, param)
		}
	}
	return params, nil
}

func loadMetadata(folder string) (kudo.Framework, error) {
	framework := kudo.Framework{}
	framework.Kind = "Framework"
	framework.APIVersion = kudo.SchemeGroupVersion.String()
	//TODO allow passing in of https:// paths.
	meta, err := chartutil.LoadChartfile(folder + "/Chart.yaml")
	if err != nil {
		meta, err = chartutil.LoadChartfile(folder + "/operator.yaml")
		if err != nil {
			return framework, err
		}
	}

	framework.ObjectMeta.Name = meta.Name
	framework.Spec.Description = meta.Description
	framework.Spec.KudoVersion = MinHelmVersion
	framework.Spec.KubernetesVersion = meta.KubeVersion
	framework.Spec.Maintainers = make([]kudo.Maintainer, 0)
	for _, m := range meta.Maintainers {
		framework.Spec.Maintainers = append(framework.Spec.Maintainers, kudo.Maintainer(fmt.Sprintf("%v <%v>", m.Name, m.Email)))
	}

	return framework, nil
}

func loadTemplates(folder string) (map[string]string, error) {
	//look in the templates folder
	templates := make(map[string]string)
	e := filepath.Walk(folder+"/templates", func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".yml" || filepath.Ext(path) == ".yaml" {
			b, e := ioutil.ReadFile(path)
			if e != nil {
				return e
			}
			templates[filepath.Base(path)] = string(b)
		}
		return nil
	})

	return templates, e
}
