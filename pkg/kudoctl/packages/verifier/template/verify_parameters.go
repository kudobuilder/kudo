package template

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var (
	// implicits is a set of usable implicits defined in render.go
	implicits = map[string]bool{
		"Name":         true, // instance name
		"Namespace":    true,
		"OperatorName": true,
		"Params":       true,
		"PlanName":     true,
		"PhaseName":    true,
		"StepName":     true,
		"AppVersion":   true,
	}
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
	for _, value := range pf.Params.Parameters {
		if value.IsImmutable() {
			if !value.HasDefault() && !value.IsRequired() {
				res.AddParamError(value.Name, "is immutable but is not marked as required or has a default value")
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
			res.AddParamWarning(value.Name, "defined but not used.")
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
