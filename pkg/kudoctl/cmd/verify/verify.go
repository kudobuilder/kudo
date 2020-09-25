package verify

import (
	"fmt"
	"io"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier/plan"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier/template"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
	"github.com/kudobuilder/kudo/pkg/version"
)

var verifiers = []packages.Verifier{
	DuplicateVerifier{},
	InvalidCharVerifier{";,"},
	VersionVerifier{},
	task.BuildVerifier{},
	task.ReferenceVerifier{},
	plan.ReferenceVerifier{},
	template.ParametersVerifier{},
	template.ReferenceVerifier{},
	template.RenderVerifier{},
	template.NamespaceVerifier{},
}

// PackageFiles verifies operator package files
func PackageFiles(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, vv := range verifiers {
		res.Merge(vv.Verify(pf))
	}
	return res
}

func PrintResult(res verifier.Result, out io.Writer) {
	res.PrintWarnings(out)
	res.PrintErrors(out)

	if res.IsValid() {
		fmt.Fprintf(out, "package is valid\n")
	}
}

// DuplicateVerifier provides verification that there are no duplicates disallowing casing (Kudo and kudo are duplicates)
type DuplicateVerifier struct{}

func (DuplicateVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	names := map[string]bool{}
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		if names[name] {
			res.AddParamError(param.Name, "has a duplicate")
		}
		names[name] = true
	}
	return res
}

type InvalidCharVerifier struct {
	InvalidChars string
}

func (v InvalidCharVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		for _, char := range name {
			if strings.Contains(v.InvalidChars, strings.ToLower(string(char))) {
				res.AddParamError(param.Name, fmt.Sprintf("contains invalid character %q", char))
			}
		}

	}

	return res
}

// VersionVerifier verifies the version in operator.yaml, kubernetesVersion, operatorVersion and kudoVersion
type VersionVerifier struct{}

func (VersionVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	if pf.Operator == nil {
		res.AddErrors("operator not defined.")
		return res
	}
	verifySemVer(pf.Operator.OperatorVersion, "operatorVersion", &res, true)
	verifySemVer(pf.Operator.AppVersion, "appVersion", &res, false)
	verifySemVer(pf.Operator.KubernetesVersion, "kubernetesVersion", &res, true)
	verifySemVer(pf.Operator.KUDOVersion, "kudoVersion", &res, false)
	return res
}

func verifySemVer(ver string, name string, res *verifier.Result, required bool) {
	v := strings.TrimSpace(ver)
	if !required && v == "" {
		return
	}

	if required && v == "" {
		res.AddErrors(fmt.Sprintf("%q is required and must be semver", name))
		return
	}

	_, err := version.New(ver)
	if err != nil {
		res.AddErrors(fmt.Sprintf("unable to parse %q: %v", name, err))
	}
}
