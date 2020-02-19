package task

import (
	"fmt"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"
)

// render method takes resource names and Instance parameters and then renders passed templates using kudo engine.
func render(resourceNames []string, ctx Context) (map[string]string, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = ctx.Meta.OperatorName
	configs["Name"] = ctx.Meta.InstanceName
	configs["Namespace"] = ctx.Meta.InstanceNamespace
	configs["Pipes"] = ctx.Pipes
	configs["PlanName"] = ctx.Meta.PlanName
	configs["PhaseName"] = ctx.Meta.PhaseName
	configs["StepName"] = ctx.Meta.StepName
	configs["AppVersion"] = ctx.Meta.AppVersion

	params, err := convertYAMLParameters(ctx.Parameters)
	if err != nil {
		return nil, err
	}

	configs["Params"] = params

	resources := map[string]string{}
	engine := renderer.New()

	for _, rn := range resourceNames {
		resource, ok := ctx.Templates[rn]

		if !ok {
			return nil, fmt.Errorf("error finding resource named %s", rn)
		}

		rendered, err := engine.Render(rn, resource, configs)
		if err != nil {
			return nil, fmt.Errorf("error expanding template %s: %w", rn, err)
		}

		resources[rn] = rendered
	}
	return resources, nil
}

// enhance method takes a slice of rendered templates, applies conventions using Enhancer and
// returns a slice of k8s objects.
func enhance(rendered map[string]string, meta renderer.Metadata, enhancer renderer.Enhancer) ([]runtime.Object, error) {
	enhanced, err := enhancer.Apply(rendered, meta)
	return enhanced, err
}

func convertYAMLParameters(parameters map[string]string) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	for k, v := range parameters {
		var unmarshalled interface{}

		if err := yaml.Unmarshal([]byte(v), &unmarshalled); err != nil {
			return nil, fmt.Errorf("error converting value '%s' of parameter '%s': %w", v, k, err)
		}

		out[k] = unmarshalled
	}

	return out, nil
}
