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
func (ParametersVerifier) Verify(pf *packages.Files) (warnings verifier.Warnings, errors verifier.Errors) {
	errors = append(errors, paramsNotDefined(pf)...)
	warnings = append(warnings, paramsDefinedNotUsed(pf)...)

	nodes := getNodeMap(pf.Templates)
	// additional processing errors
	for fname, node := range nodes {
		if node.error != nil {
			errors = append(errors, verifier.Error(fmt.Sprintf(*node.error)))
			continue
		}
		for _, param := range node.implicitParams {
			if _, ok := implicits[param]; !ok {
				errors = append(errors, verifier.Error(fmt.Sprintf("template %v defines an invalid implicit parameter %q", fname, param)))
			}
		}
	}

	return warnings, errors
}

func paramsDefinedNotUsed(pf *packages.Files) (warnings verifier.Warnings) {
	tparams := make(map[string]bool)
	nodes := getNodeMap(pf.Templates)

	for _, nodes := range nodes {
		for _, tparam := range nodes.parameters {
			tparams[tparam] = true
		}
	}
	for _, value := range pf.Params.Parameters {
		if _, ok := tparams[value.Name]; !ok {
			warnings = append(warnings, verifier.CreateParamWarning(value, "defined but not used."))
		}
	}
	return warnings
}

func paramsNotDefined(pf *packages.Files) (errors verifier.Errors) {
	params := make(map[string]bool)
	for _, param := range pf.Params.Parameters {
		params[param.Name] = true
	}
	nodes := getNodeMap(pf.Templates)

	for fname, nodes := range nodes {
		for _, tparam := range nodes.parameters {
			if _, ok := params[tparam]; !ok {
				errors = append(errors, verifier.Error(fmt.Sprintf("parameter %q in template %v is not defined", tparam, fname)))
			}
		}
	}
	return errors
}
