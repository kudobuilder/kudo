package template

import (
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &ExtendedParametersVerifier{}

type ExtendedParametersVerifier struct{}

// Verify implements packages.Verifier for parameter verification
func (v *ExtendedParametersVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(v.VerifyDescription(pf))
	res.Merge(v.VerifyDisplayName(pf))
	res.Merge(v.VerifyHint(pf))
	res.Merge(v.VerifyGroup(pf))
	res.Merge(v.VerifyType(pf))
	return res
}

func (ExtendedParametersVerifier) VerifyDescription(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Description == "" {
			res.AddParamWarning(p.Name, "has no description")
		} else {
			if !strings.HasSuffix(p.Description, ".") {
				res.AddParamWarning(p.Name, " has a description not ending with a '.'")
			}
		}
	}
	return res
}

func (ExtendedParametersVerifier) VerifyDisplayName(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.DisplayName == "" {
			res.AddParamWarning(p.Name, "has no display name")
		}
	}
	return res
}

func (ExtendedParametersVerifier) VerifyHint(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Hint == "" {
			res.AddParamWarning(p.Name, "has no hint")
		} else {
			if !strings.HasSuffix(p.Hint, ".") {
				res.AddParamWarning(p.Name, " has a hint not ending with a '.'")
			}
		}
	}
	return res
}

func (ExtendedParametersVerifier) VerifyGroup(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Group == "" {
			res.AddParamWarning(p.Name, "has no group")
		}
	}
	for _, g := range pf.Params.Groups {
		if g.Description != "" {
			if !strings.HasSuffix(g.Description, ".") {
				res.AddGroupError(g.Name, " has a description not ending with a '.'")
			}
		}
	}
	return res
}

func (ExtendedParametersVerifier) VerifyType(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Type == "" {
			res.AddParamWarning(p.Name, "has no explicit type assigned")
		}
	}
	return res
}
