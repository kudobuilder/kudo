package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestExtendedParameters(t *testing.T) {
	tests := []struct {
		param    packages.Parameter
		group    packages.Group
		verifier ExtendedParametersVerifier
		errs     []string
		warnings []string
	}{
		{
			param:    packages.Parameter{Name: "MissingHint"},
			verifier: ExtendedParametersVerifier{VerifyParamHint: true},
			warnings: []string{`parameter "MissingHint" has no hint`},
		},
		{
			param:    packages.Parameter{Name: "MissingDotInHint", Hint: "Hint without a final dot"},
			verifier: ExtendedParametersVerifier{VerifyParamHint: true},
			warnings: []string{`parameter "MissingDotInHint" has a hint not ending with a '.'`},
		},
		{
			param:    packages.Parameter{Name: "ValidHint", Hint: "Hint with a final dot."},
			verifier: ExtendedParametersVerifier{VerifyParamHint: true},
			warnings: []string{},
		},
		{
			param:    packages.Parameter{Name: "MissingDescription"},
			verifier: ExtendedParametersVerifier{VerifyParamDescription: true},
			warnings: []string{`parameter "MissingDescription" has no description`},
		},
		{
			param:    packages.Parameter{Name: "MissingDotInDescription", Description: "Description without a final dot"},
			verifier: ExtendedParametersVerifier{VerifyParamDescription: true},
			warnings: []string{`parameter "MissingDotInDescription" has a description not ending with one of '.!?)'`},
		},
		{
			param:    packages.Parameter{Name: "ValidDescription", Description: "Description with a final dot."},
			verifier: ExtendedParametersVerifier{VerifyParamDescription: true},
			warnings: []string{},
		},
		{
			param:    packages.Parameter{Name: "MissingDisplayName"},
			verifier: ExtendedParametersVerifier{VerifyParamDisplayName: true},
			warnings: []string{`parameter "MissingDisplayName" has no displayName`},
		},
		{
			param:    packages.Parameter{Name: "DisplayNameWithColon", DisplayName: "MyParameter:"},
			verifier: ExtendedParametersVerifier{VerifyParamDisplayName: true},
			warnings: []string{`parameter "DisplayNameWithColon" has a displayName ending with ':'`},
		},
		{
			param:    packages.Parameter{Name: "ParamMissingGroup"},
			verifier: ExtendedParametersVerifier{VerifyParamGroup: true},
			warnings: []string{`parameter "ParamMissingGroup" has no group`},
		},
		{
			param:    packages.Parameter{Name: "ParamValidGroup", Group: "somegroup"},
			verifier: ExtendedParametersVerifier{VerifyParamGroup: true},
			warnings: []string{},
		},
		{
			group:    packages.Group{Name: "GroupMissingBoth"},
			verifier: ExtendedParametersVerifier{VerifyGroups: true},
			warnings: []string{
				`parameter group "GroupMissingBoth" has no displayName`,
				`parameter group "GroupMissingBoth" has no description`,
			},
		},
		{
			group:    packages.Group{Name: "GroupWithInvalidFields", DisplayName: "MyGroup:", Description: "Description without dot"},
			verifier: ExtendedParametersVerifier{VerifyGroups: true},
			warnings: []string{
				`parameter group "GroupWithInvalidFields" has a displayName ending with ':'`,
				`parameter group "GroupWithInvalidFields" has a description not ending with one of '.!?)'`,
			},
		},
		{
			group:    packages.Group{Name: "ValidGroup", DisplayName: "MyGroup", Description: "My Group Description."},
			verifier: ExtendedParametersVerifier{VerifyGroups: true},
			warnings: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		var name string
		if tt.param.Name != "" {
			name = tt.param.Name
		} else {
			name = tt.group.Name
		}

		t.Run(name, func(t *testing.T) {
			paramFile := packages.ParamsFile{Parameters: []packages.Parameter{tt.param}, Groups: []packages.Group{tt.group}}
			templates := make(map[string]string)

			operator := packages.OperatorFile{}
			pf := packages.Files{
				Templates: templates,
				Operator:  &operator,
				Params:    &paramFile,
			}

			verifier := tt.verifier
			res := verifier.Verify(&pf)

			assert.Equal(t, len(tt.warnings), len(res.Warnings))
			assert.Equal(t, len(tt.errs), len(res.Errors))
			if len(tt.errs) == len(res.Errors) {
				for i, e := range tt.errs {
					assert.Equal(t, e, res.Errors[i], "Error at position %d did not match", i)
				}
			}
			if len(tt.warnings) == len(res.Warnings) {
				for i, w := range tt.warnings {
					assert.Equal(t, w, res.Warnings[i], "Warning at position %d did not match", i)
				}
			}
		})
	}
}
