package template

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &ExtendedParametersVerifier{}

type ParamVerifyArgumentType string

const (
	ParamVerifyDisplayName ParamVerifyArgumentType = "display"
	ParamVerifyHint        ParamVerifyArgumentType = "hint"
	ParamVerifyDescription ParamVerifyArgumentType = "desc"
	ParamVerifyType        ParamVerifyArgumentType = "type"
	ParamVerifyHasGroup    ParamVerifyArgumentType = "hasgroup"
	ParamVerifyGroups      ParamVerifyArgumentType = "groups"
	ParamVerifyAll         ParamVerifyArgumentType = "all"
)

var (
	ParamVerifyArguments = []ParamVerifyArgumentType{
		ParamVerifyDisplayName,
		ParamVerifyHint,
		ParamVerifyDescription,
		ParamVerifyType,
		ParamVerifyHasGroup,
		ParamVerifyGroups,
		ParamVerifyAll,
	}
)

type ExtendedParametersVerifier struct {
	VerifyParamDescription bool
	VerifyParamHint        bool
	VerifyParamDisplayName bool
	VerifyParamType        bool
	VerifyParamGroup       bool
	VerifyGroups           bool
}

func (v *ExtendedParametersVerifier) SetFromArguments(args []string) error {
	for _, c := range args {

		switch ParamVerifyArgumentType(c) {
		case ParamVerifyDisplayName:
			v.VerifyParamDisplayName = true
		case ParamVerifyHint:
			v.VerifyParamHint = true
		case ParamVerifyDescription:
			v.VerifyParamDescription = true
		case ParamVerifyType:
			v.VerifyParamType = true
		case ParamVerifyHasGroup:
			v.VerifyParamGroup = true
		case ParamVerifyGroups:
			v.VerifyGroups = true
		case ParamVerifyAll:
			v.VerifyParamDisplayName = true
			v.VerifyParamHint = true
			v.VerifyParamDescription = true
			v.VerifyParamType = true
			v.VerifyParamGroup = true
			v.VerifyGroups = true
		default:
			return fmt.Errorf("unknown parameter check: %s, must be one of %v", c, ParamVerifyArguments)
		}
	}
	return nil
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
		if ok, msg := validHint(p.Hint); !ok {
			res.AddParamWarning(p.Name, msg)
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
		// The check if a specified group is defined in the groups section is in verify_parameters.go, as it
		// is a required check and not an optional one.
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
	if !IsUpper(displayName[0:1]) {
		return false, "has a displayName that does not start with a capital letter"
	}
	return true, ""
}

func validDescription(description string) (bool, string) {
	if description == "" {
		return false, "has no description"
	}
	lastChar := description[len(description)-1:]
	if !strings.Contains(".!?)", lastChar) { //nolint:gocritic
		return false, "has a description not ending with one of '.!?)'"
	}
	if !IsUpper(description[0:1]) {
		return false, "has a description that does not start with a capital letter"
	}
	return true, ""
}

func validHint(hint string) (bool, string) {
	if hint == "" {
		return false, "has no hint"
	}
	if !strings.HasSuffix(hint, ".") {
		return false, "has a hint not ending with a '.'"
	}
	return true, ""
}

func IsUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
