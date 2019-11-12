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

var verifiers = []ParameterVerifier{
	DuplicateVerifier{},
	InvalidCharVerifier{";,"},
}

// Parameters verifies parameters
func Parameters(params packages.Params) (warnings ParamWarnings, errors ParamErrors) {
	for _, verifier := range verifiers {
		w, err := verifier.Verify(params)
		warnings = append(warnings, w...)
		errors = append(errors, err...)
	}
	return warnings, errors
}

// ParameterVerifier defines the interface for all parameter verifiers
type ParameterVerifier interface {
	Verify(params packages.Params) (ParamWarnings, ParamErrors)
}

func CreateParamError(param v1beta1.Parameter, reason string) ParamError {
	return ParamError(fmt.Sprintf("parameter %q %s", param.Name, reason))
}

// DuplicateVerifier provides verification that there are no duplicates disallowing casing (Kudo and kudo are duplicates)
type DuplicateVerifier struct {
}

func (DuplicateVerifier) Verify(params packages.Params) (warnings ParamWarnings, errors ParamErrors) {
	names := map[string]bool{}
	for _, param := range params {
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

func (v InvalidCharVerifier) Verify(params packages.Params) (warnings ParamWarnings, errors ParamErrors) {
	for _, param := range params {
		name := strings.ToLower(param.Name)
		for _, char := range name {
			if strings.Contains(v.InvalidChars, strings.ToLower(string(char))) {
				errors = append(errors, CreateParamError(param, fmt.Sprintf("contains invalid character %q", char)))
			}
		}

	}

	return warnings, errors
}
