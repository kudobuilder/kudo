package verify

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/template"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
)

var verifiers = []verifier.PackageVerifier{
	DuplicateVerifier{},
	InvalidCharVerifier{";,"},
	template.ParametersVerifier{},
	template.ReferenceVerifier{},
}

// Operator verifies operator package files
func Operator(pf *packages.Files) (warnings verifier.ParamWarnings, errors verifier.ParamErrors) {
	for _, verifier := range verifiers {
		w, err := verifier.Verify(pf)
		warnings = append(warnings, w...)
		errors = append(errors, err...)
	}
	return warnings, errors
}

// DuplicateVerifier provides verification that there are no duplicates disallowing casing (Kudo and kudo are duplicates)
type DuplicateVerifier struct {
}

func (DuplicateVerifier) Verify(pf *packages.Files) (warnings verifier.ParamWarnings, errors verifier.ParamErrors) {
	names := map[string]bool{}
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		if names[name] {
			errors = append(errors, verifier.CreateParamError(param, "has a duplicate"))
		}
		names[name] = true
	}
	return warnings, errors
}

type InvalidCharVerifier struct {
	InvalidChars string
}

func (v InvalidCharVerifier) Verify(pf *packages.Files) (warnings verifier.ParamWarnings, errors verifier.ParamErrors) {
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		for _, char := range name {
			if strings.Contains(v.InvalidChars, strings.ToLower(string(char))) {
				errors = append(errors, verifier.CreateParamError(param, fmt.Sprintf("contains invalid character %q", char)))
			}
		}

	}

	return warnings, errors
}
