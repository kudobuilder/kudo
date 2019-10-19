package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine"
)

// render method takes resource names and Instance parameters and then renders passed templates using kudo engine.
func render(resourceNames []string, templates map[string]string, params map[string]string, meta ExecutionMetadata) (map[string]string, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.OperatorName
	configs["Name"] = meta.InstanceName
	configs["Namespace"] = meta.InstanceNamespace
	configs["Params"] = params
	configs["PlanName"] = meta.PlanName
	configs["PhaseName"] = meta.PhaseName
	configs["StepName"] = meta.StepName

	resources := map[string]string{}
	engine := engine.New()

	for _, rn := range resourceNames {
		resource, ok := templates[rn]

		if !ok {
			return nil, fmt.Errorf("error finding resource named %v for operator version %v", rn, meta.OperatorVersionName)
		}

		rendered, err := engine.Render(resource, configs)
		if err != nil {
			return nil, fmt.Errorf("error expanding template: %w", err)
		}

		resources[rn] = rendered
	}
	return resources, nil
}
