package template

import (
	"fmt"

	engtask "github.com/kudobuilder/kudo/pkg/engine/task"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	packageconvert "github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
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

	params, err := collectParams(pf)
	if err != nil {
		res.AddErrors(err.Error())
		return res
	}
	pipes, err := collectPipes(pf)
	if err != nil {
		res.AddErrors(err.Error())
		return res
	}

	configs := renderer.NewVariableMap().
		WithDefaults().
		WithParameters(params).
		WithPipes(pipes)

	engine := renderer.New()
	for k, v := range pf.Templates {
		// Render the template
		s, err := engine.Render(k, v, configs)
		if err != nil {
			res.AddErrors(err.Error()) // err already mentions template name
		}

		// parameter file for KudoOperator task does not have to parse to kubernetes object
		if !isParameterFile(k, pf) {
			// Try to parse rendered template as valid Kubernetes objects
			_, err = renderer.YamlToObject(s)
			if err != nil {
				res.AddErrors(fmt.Sprintf("parsing rendered YAML from %s failed: %v", k, err))
			}
		}
	}

	return res
}

func isParameterFile(k string, pf *packages.Files) bool {
	for _, task := range pf.Operator.Tasks {
		switch task.Kind {
		case engtask.KudoOperatorTaskKind:
			if k == task.Spec.KudoOperatorTaskSpec.ParameterFile {
				return true
			}
		default:
		}
	}
	return false
}

func collectPipes(pf *packages.Files) (map[string]string, error) {
	pipes := make(map[string]string)
	for name, plan := range pf.Operator.Plans {
		plan := plan
		planPipes, err := instance.PipesMap(name, &plan, pf.Operator.Tasks, &engine.Metadata{})
		if err != nil {
			return nil, err
		}
		for key, value := range planPipes {
			pipes[key] = value
		}
	}
	return pipes, nil
}

func collectParams(pf *packages.Files) (map[string]interface{}, error) {
	parameters, err := packageconvert.ParametersToCRDType(pf.Params.Parameters)
	if err != nil {
		return nil, err
	}
	return instance.ParamsMap(&kudoapi.Instance{}, &kudoapi.OperatorVersion{Spec: kudoapi.OperatorVersionSpec{Parameters: parameters}})
}
