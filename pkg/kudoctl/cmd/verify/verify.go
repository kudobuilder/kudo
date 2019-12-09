package verify

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier/template"
)

var verifiers = []verifier.PackageVerifier{
	DuplicateVerifier{},
	InvalidCharVerifier{";,"},
	template.ParametersVerifier{},
	template.ReferenceVerifier{},
}

// PackageFiles verifies operator package files
func PackageFiles(pf *packages.Files) (warnings verifier.Warnings, errors verifier.Errors) {
	for _, verifier := range verifiers {
		ws, errs := verifier.Verify(pf)
		warnings = append(warnings, ws...)
		errors = append(errors, errs...)
	}
	return warnings, errors
}

// DuplicateVerifier provides verification that there are no duplicates disallowing casing (Kudo and kudo are duplicates)
type DuplicateVerifier struct{}

func (DuplicateVerifier) Verify(pf *packages.Files) (warnings verifier.Warnings, errors verifier.Errors) {
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

func (v InvalidCharVerifier) Verify(pf *packages.Files) (warnings verifier.Warnings, errors verifier.Errors) {
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
