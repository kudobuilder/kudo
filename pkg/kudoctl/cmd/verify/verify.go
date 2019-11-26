package verify

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

type ParamWarning string
type ParamWarnings []ParamWarning
type ParamError string
type ParamErrors []ParamError

var verifiers = []PackageVerifier{
	DuplicateVerifier{},
	InvalidCharVerifier{";,"},
	TemplateParametersVerifier{},
}

// Parameters verifies parameters
func Parameters(pf *packages.Files) (warnings ParamWarnings, errors ParamErrors) {
	for _, verifier := range verifiers {
		w, err := verifier.Verify(pf)
		warnings = append(warnings, w...)
		errors = append(errors, err...)
	}
	return warnings, errors
}

// PackageVerifier defines the interface for all parameter verifiers
type PackageVerifier interface {
	Verify(pf *packages.Files) (ParamWarnings, ParamErrors)
}

func CreateParamError(param v1beta1.Parameter, reason string) ParamError {
	return ParamError(fmt.Sprintf("parameter %q %s", param.Name, reason))
}

func CreateParamWarning(param v1beta1.Parameter, reason string) ParamWarning {
	return ParamWarning(fmt.Sprintf("parameter %q %s", param.Name, reason))
}

// DuplicateVerifier provides verification that there are no duplicates disallowing casing (Kudo and kudo are duplicates)
type DuplicateVerifier struct {
}

func (DuplicateVerifier) Verify(pf *packages.Files) (warnings ParamWarnings, errors ParamErrors) {
	names := map[string]bool{}
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		if names[name] {
			errors = append(errors, CreateParamError(param, "has a duplicate"))
		}
		names[name] = true
	}
	return warnings, errors
}

type InvalidCharVerifier struct {
	InvalidChars string
}

func (v InvalidCharVerifier) Verify(pf *packages.Files) (warnings ParamWarnings, errors ParamErrors) {
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		for _, char := range name {
			if strings.Contains(v.InvalidChars, strings.ToLower(string(char))) {
				errors = append(errors, CreateParamError(param, fmt.Sprintf("contains invalid character %q", char)))
			}
		}

	}

	return warnings, errors
}

// TemplateParametersDefinedVerifier checks that all parameters used in templates are defined
// checks that all defined parameters are used in templates
type TemplateParametersVerifier struct {
}

func (TemplateParametersVerifier) Verify(pf *packages.Files) (warnings ParamWarnings, errors ParamErrors) {

	errors = append(errors, paramsNotDefined(pf)...)
	warnings = append(warnings, paramsDefinedNotUsed(pf)...)

	// additional processing errors
	for fname, node := range pf.Templates.Nodes() {
		if node.Error != nil {
			errors = append(errors, ParamError(fmt.Sprintf("template %v has error %v", fname, node.Error)))
		}
	}

	return warnings, errors
}

func paramsDefinedNotUsed(pf *packages.Files) (warnings ParamWarnings) {
	tparams := make(map[string]bool)
	for _, nodes := range pf.Templates.Nodes() {
		for _, tparam := range nodes.Parameters {
			tparams[tparam] = true
		}
	}
	for _, value := range pf.Params.Parameters {
		if _, ok := tparams[value.Name]; ok {
			warnings = append(warnings, CreateParamWarning(value, "defined but not used."))
		}
	}
	return warnings
}

func paramsNotDefined(pf *packages.Files) (errors ParamErrors) {
	params := make(map[string]bool)
	for _, param := range pf.Params.Parameters {
		params[param.Name] = true
	}
	for fname, nodes := range pf.Templates.Nodes() {
		for _, tparam := range nodes.Parameters {
			if _, ok := params[tparam]; !ok {
				errors = append(errors, ParamError(fmt.Sprintf("parameter %q in template %v is not defined", tparam, fname)))
			}
		}
	}
	return errors
}

// TemplateVerifier checks that all referenced templates exists (without errors)
// and warns if a template exists but isn't referenced in a plan
type TemplateVerifier struct {
}

func (TemplateVerifier) Verify(pf *packages.Files) (warnings ParamWarnings, errors ParamErrors) {

	return warnings, errors
}
