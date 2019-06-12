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
	"strings"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/helm/helm/pkg/chartutil"
)

// MinKUDOVersionToSupportHelm is the minimum KUDO version that supports helm templating
const MinKUDOVersionToSupportHelm = "0.2.0"

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
	return params, err
}

func loadParametersFromChart(chart *chart.Chart) ([]kudo.Parameter, error) {
	params := make([]kudo.Parameter, 0)

	if settings.Debug {
		fmt.Printf("Raw: %v\n", chart.GetValues().GetRaw())

		fmt.Printf("Values:\n")
	}
	cparams := chart.GetValues().GetValues()

	for k, v := range cparams {
		if settings.Debug {
			fmt.Printf("%v -> %v\n", k, v.GetValue())
		}
		params = append(params,
			kudo.Parameter{
				Name:        k, // ???
				Default:     v.GetValue(),
				Description: "Auto import from helm chart", //Maybe look at comments above the line?
			})
	}
	return params, nil
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

func loadTemplateFromChart(chart *chart.Chart) (map[string]string, error) {
	templates := chart.GetTemplates()
	out := make(map[string]string)
	for _, template := range templates {
		name := template.GetName()
		name = strings.Replace(name, "templates/", "", -1)
		if strings.HasPrefix(name, "tests/") {
			//part of testing
			continue
		}
		out[name] = string(template.GetData())
	}
	return out, nil
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
