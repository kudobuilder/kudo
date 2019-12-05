package template

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
)

// ParametersVerifier checks that all parameters used in templates are defined
// checks that all defined parameters are used in templates
type ParametersVerifier struct {
}

func (ParametersVerifier) Verify(pf *packages.Files) (warnings verifier.ParamWarnings, errors verifier.ParamErrors) {

	errors = append(errors, paramsNotDefined(pf)...)
	warnings = append(warnings, paramsDefinedNotUsed(pf)...)

	nodes := getNodeMap(pf.Templates)
	// additional processing errors
	for fname, node := range nodes {
		if node.error != nil {
			errors = append(errors, verifier.ParamError(fmt.Sprintf(*node.error)))
			continue
		}
		for _, param := range node.implicitParams {
			if _, ok := packages.Implicits[param]; !ok {
				errors = append(errors, verifier.ParamError(fmt.Sprintf("template %v defines an invalid implicit parameter %q", fname, param)))
			}
		}
	}

	return warnings, errors
}

func paramsDefinedNotUsed(pf *packages.Files) (warnings verifier.ParamWarnings) {
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

func paramsNotDefined(pf *packages.Files) (errors verifier.ParamErrors) {
	params := make(map[string]bool)
	for _, param := range pf.Params.Parameters {
		params[param.Name] = true
	}
	nodes := getNodeMap(pf.Templates)

	for fname, nodes := range nodes {
		for _, tparam := range nodes.parameters {
			if _, ok := params[tparam]; !ok {
				errors = append(errors, verifier.ParamError(fmt.Sprintf("parameter %q in template %v is not defined", tparam, fname)))
			}
		}
	}
	return errors
}

// ReferenceVerifier checks that all referenced templates exists (without errors)
// and warns if a template exists but isn't referenced in a plan
type ReferenceVerifier struct {
}

func (ReferenceVerifier) Verify(pf *packages.Files) (warnings verifier.ParamWarnings, errors verifier.ParamErrors) {
	templates := make(map[string]bool)
	for template := range pf.Templates {
		templates[template] = true
	}

	// conflated a bit...  the loop 1) confirms that all resources are defined templates, and 2) creates a map of all resources for next verification
	requiredTemplates := make(map[string]bool)
	for _, task := range pf.Operator.Tasks {
		for _, resource := range task.Spec.Resources {
			requiredTemplates[resource] = true
			if _, ok := templates[resource]; !ok {
				errors = append(errors, verifier.ParamError(fmt.Sprintf("template %q required by %v but not defined", resource, task.Name)))
			}
		}
	}

	for template := range templates {
		if _, ok := requiredTemplates[template]; !ok {
			warnings = append(warnings, verifier.ParamWarning(fmt.Sprintf("template %q is not referenced from any task", template)))
		}
	}

	return warnings, errors
}
