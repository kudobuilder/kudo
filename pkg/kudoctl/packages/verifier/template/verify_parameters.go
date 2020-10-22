package template

import (
	"fmt"
	"strings"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &ParametersVerifier{}

// ParametersVerifier checks that all parameters used in templates are defined
// checks that all defined parameters are used in templates
type ParametersVerifier struct{}

// Verify implements packages.Verifier for parameter verification
func (ParametersVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(paramsNotDefined(pf))
	res.Merge(paramsDefinedNotUsed(pf))
	res.Merge(immutableParams(pf))
	res.Merge(enumParams(pf))
	res.Merge(paramDefaults(pf))
	res.Merge(metadata(pf))
	res.Merge(paramGroups(pf))

	implicits := renderer.NewVariableMap().WithDefaults()

	nodes := getNodeMap(pf.Templates)
	// additional processing errors
	for fname, node := range nodes {
		if node.error != nil {
			res.AddErrors(fmt.Sprintf(*node.error))
			continue
		}
		for _, param := range node.implicitParams {
			if _, ok := implicits[param]; !ok {
				res.AddErrors(fmt.Sprintf("template %v defines an invalid implicit parameter %q", fname, param))
			}
		}
	}

	return res
}

func immutableParams(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.IsImmutable() {
			if !p.HasDefault() && !p.IsRequired() {
				res.AddParamError(p.Name, "is immutable but is not marked as required or has a default value")
			}
		}
	}
	return res
}

func paramDefaults(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.HasDefault() {
			if err := p.ValidateDefault(); err != nil {
				res.AddErrors(err.Error())
			}
		}
	}
	return res
}

func enumParams(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {

		if p.IsEnum() {
			if len(p.EnumValues()) == 0 {
				res.AddParamError(p.Name, "is an enum but has no allowed values")
				continue
			}
			for _, enumVal := range p.EnumValues() {

				if err := kudoapi.ValidateParameterValueForType(p.Type, enumVal); err != nil {
					res.AddParamError(p.Name, fmt.Sprintf("has an invalid enum value: %v", err))
				}
			}
		}
	}
	return res
}

func metadata(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Group != "" {
			if strings.Contains(p.Group, "/") {
				res.AddParamError(p.Name, "has a group with invalid character '/'")
			}
		}
		if p.IsAdvanced() && !p.HasDefault() && p.IsRequired() {
			res.AddParamError(p.Name, "is marked as advanced, but also as required and has no default. An advanced parameter must either be optional or have a default value")
		}
	}
	return res
}

func paramGroups(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()

	groups := map[string]packages.Group{}
	for _, g := range pf.Params.Groups {
		if strings.Contains(g.Name, "/") {
			res.AddGroupError(g.Name, "contains invalid character '/'")
		}
		if _, ok := groups[g.Name]; ok {
			res.AddGroupError(g.Name, "is duplicated")
		}
		groups[g.Name] = g
	}

	for _, p := range pf.Params.Parameters {
		if p.Group != "" {
			if _, ok := groups[p.Group]; !ok {
				res.AddParamError(p.Name, fmt.Sprintf("has a group %q that is not defined in the group section", p.Group))
			}
		}
	}

	return res
}

func paramsDefinedNotUsed(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	tparams := make(map[string]bool)
	nodes := getNodeMap(pf.Templates)

	for _, node := range nodes {
		for _, tparam := range node.parameters {
			tparams[tparam] = true
		}
	}
	for _, opTask := range pf.Operator.Tasks {
		if opTask.Kind == task.ToggleTaskKind {
			tparams[opTask.Spec.Parameter] = true
		}
	}
	for _, value := range pf.Params.Parameters {
		if _, ok := tparams[value.Name]; !ok {
			// A parameter could be use to trigger a plan while not being used in templates.
			if value.Trigger == "" {
				res.AddParamWarning(value.Name, "defined but not used.")
			}
		}
	}
	return res
}

func paramsNotDefined(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	params := make(map[string]bool)
	for _, param := range pf.Params.Parameters {
		params[param.Name] = true
	}
	nodes := getNodeMap(pf.Templates)

	for fname, nodes := range nodes {
		for _, tparam := range nodes.parameters {
			if _, ok := params[tparam]; !ok {
				res.AddErrors(fmt.Sprintf("parameter %q in template %v is not defined", tparam, fname))
			}
		}
	}
	for _, opTask := range pf.Operator.Tasks {
		if opTask.Kind == task.ToggleTaskKind {
			// Only checking non-empty parameter prevents a double error in verification
			if len(opTask.Spec.Parameter) > 0 {
				if _, ok := params[opTask.Spec.Parameter]; !ok {
					res.AddErrors(fmt.Sprintf("parameter %q in ToggleTask %v is not defined", opTask.Spec.Parameter, opTask.Name))
				}
			}
		}
	}
	return res
}
