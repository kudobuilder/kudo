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
		if p.Description == "" {
			res.AddParamWarning(p.Name, "has no description")
		} else {
			if !strings.HasSuffix(p.Description, ".") {
				res.AddParamWarning(p.Name, "has a description not ending with a '.'")
			}
		}
	}
	return res
}

func (ExtendedParametersVerifier) verifyDisplayName(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, p := range pf.Params.Parameters {
		if p.DisplayName == "" {
			res.AddParamWarning(p.Name, "has no displayName")
		}
		if strings.HasSuffix(p.DisplayName, ":") {
			res.AddParamWarning(p.Name, "has a displayName ending with ':'")
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
	for _, g := range pf.Params.Groups {
		if v.VerifyParamDescription && g.Description != "" {
			if !strings.HasSuffix(g.Description, ".") {
				res.AddGroupWarning(g.Name, "has a description not ending with a '.'")
			}
		}
		if v.VerifyParamDisplayName && g.DisplayName != "" {
			if strings.HasSuffix(g.DisplayName, ":") {
				res.AddGroupWarning(g.Name, "has a displayName ending with ':'")
			}
		}
	}
	return res
}

func (v ExtendedParametersVerifier) verifyGroups(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, g := range pf.Params.Groups {
		if g.DisplayName == "" {
			res.AddGroupWarning(g.Name, "has no displayName")
		} else {
			if strings.HasSuffix(g.DisplayName, ":") {
				res.AddGroupWarning(g.Name, "has a displayName ending with ':'")
			}
		}
		if g.Description == "" {
			res.AddGroupWarning(g.Name, "has no description")
		} else {
			if !strings.HasSuffix(g.Description, ".") {
				res.AddGroupWarning(g.Name, "has a description not ending with a '.'")
			}
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
