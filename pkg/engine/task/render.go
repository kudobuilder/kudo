package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"k8s.io/apimachinery/pkg/runtime"
)

// render method takes resource names and Instance parameters and then renders passed templates using kudo engine.
func render(resourceNames []string, templates map[string]string, params map[string]string, meta renderer.Metadata) (map[string]string, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.OperatorName
	configs["Name"] = meta.InstanceName
	configs["Namespace"] = meta.InstanceNamespace
	configs["Params"] = params
	configs["PlanName"] = meta.PlanName
	configs["PhaseName"] = meta.PhaseName
	configs["StepName"] = meta.StepName
	configs["AppVersion"] = meta.AppVersion

	resources := map[string]string{}
	engine := renderer.New()

	for _, rn := range resourceNames {
		resource, ok := templates[rn]

		if !ok {
			return nil, fmt.Errorf("error finding resource named %s", rn)
		}

		rendered, err := engine.Render(resource, configs)
		if err != nil {
			return nil, fmt.Errorf("error expanding template %s: %w", rn, err)
		}

		resources[rn] = rendered
	}
	return resources, nil
}

// kustomize method takes a slice of rendered templates, applies conventions using Enhancer and
// returns a slice of k8s objects.
func kustomize(rendered map[string]string, meta renderer.Metadata, enhancer renderer.Enhancer) ([]runtime.Object, error) {
	enhanced, err := enhancer.Apply(rendered, meta)
	return enhanced, err
}
