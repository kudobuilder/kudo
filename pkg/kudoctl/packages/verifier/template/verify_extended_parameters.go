package template

import (
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &ExtendedParametersVerifier{}

type ExtendedParametersVerifier struct {
	VerifyParamDescription bool
	VerifyParamHint        bool
	VerifyParamDisplayName bool
	VerifyParamType        bool
	VerifyParamGroup       bool
	VerifyGroups           bool
}

// Verify implements packages.Verifier for parameter verification
func (v *ExtendedParametersVerifier) Verify(pf *packages.Files) verifier.Result {

	res := verifier.NewResult()
	if v.VerifyParamDescription {
		res.Merge(v.verifyDescription(pf))
	}
	if v.VerifyParamHint {
		res.Merge(v.verifyHint(pf))
	}
	if v.VerifyParamDisplayName {
		res.Merge(v.verifyDisplayName(pf))
	}
	if v.VerifyParamGroup {
		res.Merge(v.verifyParamGroup(pf))
	}
	if v.VerifyParamType {
		res.Merge(v.verifyType(pf))
	}
	if v.VerifyGroups {
		res.Merge(v.verifyGroups(pf))
	}
	return res
}

func (ExtendedParametersVerifier) verifyDescription(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if ok, msg := validDescription(p.Description); !ok {
			res.AddParamWarning(p.Name, msg)
		}
	}
	return res
}

func (ExtendedParametersVerifier) verifyDisplayName(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if ok, msg := validDisplayName(p.DisplayName); !ok {
			res.AddParamWarning(p.Name, msg)
		}
	}
	return res
}

func (ExtendedParametersVerifier) verifyHint(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Hint == "" {
			res.AddParamWarning(p.Name, "has no hint")
		} else {
			if !strings.HasSuffix(p.Hint, ".") {
				res.AddParamWarning(p.Name, "has a hint not ending with a '.'")
			}
		}
	}
	return res
}

func (v ExtendedParametersVerifier) verifyParamGroup(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Group == "" {
			res.AddParamWarning(p.Name, "has no group")
		}
	}
	return res
}

func (v ExtendedParametersVerifier) verifyGroups(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, g := range pf.Params.Groups {
		if ok, msg := validDisplayName(g.DisplayName); !ok {
			res.AddGroupWarning(g.Name, msg)
		}
		if ok, msg := validDescription(g.Description); !ok {
			res.AddGroupWarning(g.Name, msg)
		}
	}
	return res
}

func (ExtendedParametersVerifier) verifyType(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.Type == "" {
			res.AddParamWarning(p.Name, "has no explicit type assigned")
		}
	}
	return res
}

func validDisplayName(displayName string) (bool, string) {
	if displayName == "" {
		return false, "has no displayName"
	}
	if strings.HasSuffix(displayName, ":") {
		return false, "has a displayName ending with ':'"
	}
	return true, ""
}

func validDescription(description string) (bool, string) {
	if description == "" {
		return false, "has no description"
	}
	lastChar := description[len(description)-1:]
	if !strings.Contains(".!?)", lastChar) {
		return false, "has a description not ending with one of '.!?)'"
	}
	return true, ""
}
