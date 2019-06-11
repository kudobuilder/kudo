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
