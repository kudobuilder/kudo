package template

import (
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

var _ packages.Verifier = &RenderVerifier{}

// RenderVerifier checks that all templates are compilable and contain valid golang template syntax
type RenderVerifier struct{}

func (RenderVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(templateCompilable(pf))
	return res
}

func templateCompilable(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()

	params := make(map[string]interface{}, len(pf.Params.Parameters))

	for _, p := range pf.Params.Parameters {
		value, err := convert.ParamValue(p.Default, p.Type)
		if err != nil {
			res.AddErrors(err.Error())
			continue
		}

		params[p.Name] = value
	}

	configs := make(map[string]interface{})
	configs["OperatorName"] = "OperatorName"
	configs["Name"] = "Name"
	configs["Namespace"] = "Namespace"
	configs["Params"] = params
	configs["Pipes"] = make(map[string]string)
	configs["PlanName"] = "PlanName"
	configs["PhaseName"] = "PhaseName"
	configs["StepName"] = "StepName"
	configs["AppVersion"] = "AppVersion"

	engine := renderer.New()
	for k, v := range pf.Templates {
		// Render the template
		s, err := engine.Render(k, v, configs)
		if err != nil {
			res.AddErrors(err.Error())
		}

		// Try to parse rendered template as valid Kubernetes objects
		_, err = renderer.YamlToObject(s)
		if err != nil {
			res.AddErrors(err.Error())
		}
	}

	return res
}
