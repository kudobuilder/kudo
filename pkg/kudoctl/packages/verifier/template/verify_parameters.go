package template

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
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

var _ verifier.PackageVerifier = &ParametersVerifier{}

// ParametersVerifier checks that all parameters used in templates are defined
// checks that all defined parameters are used in templates
type ParametersVerifier struct{}

// Verify implements verifier.PackageVerifier for parameter verification
func (ParametersVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(paramsNotDefined(pf))
	res.Merge(paramsDefinedNotUsed(pf))

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

func paramsDefinedNotUsed(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	tparams := make(map[string]bool)
	nodes := getNodeMap(pf.Templates)

	for _, nodes := range nodes {
		for _, tparam := range nodes.parameters {
			tparams[tparam] = true
		}
	}
	for _, value := range pf.Params.Parameters {
		if _, ok := tparams[value.Name]; !ok {
			res.AddParamWarning(value, "defined but not used.")
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
	return res
}
