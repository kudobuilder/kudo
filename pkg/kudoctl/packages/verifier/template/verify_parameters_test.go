package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTemplateParametersVerifier(t *testing.T) {
	params := []packages.Parameter{
		{Name: "Foo"},
		{Name: "NotUsed"},
		{Name: "UsedViaRoot"},
		{Name: "BROKER_COUNT"},
		{Name: "EXTERNAL_NODE_PORT"},
		{Name: "TRIGGER_ONLY", Trigger: "foo"},
		{Name: "ValidDefault", Type: kudoapi.IntegerValueType, Default: 42},
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
## 2 types of params Foo and Bar
{{.Params.Foo}}
{{.Params.Bar}}

## 2 implicits Bar and Name
{{.Bar}}
{{.Name}}

## All other implicits shouldn't fail
{{.Namespace}}
{{.OperatorName}}
{{.OperatorVersion}}
{{.PlanName}}
{{.PhaseName}}
{{.StepName}}
{{.AppVersion}}

## $ as a prefix should not cause issues
{{$.AppVersion}}
{{$.Params.UsedViaRoot}}

## param used in range (int) should be counted as param
{{ range $i, $v := until (int .Params.BROKER_COUNT) }}
{{ end }}

## range example should see EXTERNAL_NODE_PORT
{{ range $i, $v := until (int .Params.BROKER_COUNT) }}
- containerPort: {{ add (int $.Params.EXTERNAL_NODE_PORT) $v}}
  name: node-port-{{ $v }}
{{ end }}

{{.Params.ValidDefault}}
`
	operator := packages.OperatorFile{
		Tasks: []kudoapi.Task{
			{
				Name: "toggleTask",
				Kind: task.ToggleTaskKind,
				Spec: kudoapi.TaskSpec{
					ToggleTaskSpec: kudoapi.ToggleTaskSpec{Parameter: "Foo"},
				},
			},
			{
				Name: "toggleTaskNotDefinedParam",
				Kind: task.ToggleTaskKind,
				Spec: kudoapi.TaskSpec{
					ToggleTaskSpec: kudoapi.ToggleTaskSpec{Parameter: "NotDefined"},
				},
			},
		},
	}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := ParametersVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 1, len(res.Warnings))
	assert.Equal(t, `parameter "NotUsed" defined but not used.`, res.Warnings[0])
	assert.Equal(t, 3, len(res.Errors))
	assert.Equal(t, `parameter "Bar" in template foo.yaml is not defined`, res.Errors[0])
	assert.Equal(t, `parameter "NotDefined" in ToggleTask toggleTaskNotDefinedParam is not defined`, res.Errors[1])
	assert.Equal(t, `template foo.yaml defines an invalid implicit parameter "Bar"`, res.Errors[2])
}

func TestImmutableParams(t *testing.T) {
	trueFlag := true
	params := []packages.Parameter{
		{Name: "NoDefaultOrRequired", Immutable: &trueFlag},
		{Name: "IsRequired", Immutable: &trueFlag, Required: &trueFlag},
		{Name: "HasDefault", Immutable: &trueFlag, Default: "Yes"},
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)

	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := ParametersVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 3, len(res.Warnings)) // NotUsed Warnings
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, `parameter "NoDefaultOrRequired" is immutable but is not marked as required or has a default value`, res.Errors[0])
}

func TestEnumParams(t *testing.T) {

	params := []packages.Parameter{
		{Name: "NoDefaultNoEnum"},
		{Name: "EnumNoDefault", Enum: &[]interface{}{"someVal"}},
		{Name: "EnumWithDefault", Enum: &[]interface{}{"someVal"}, Default: "someOtherVal"},
		{Name: "EnumNoValues", Enum: &[]interface{}{}},
		{Name: "EnumWrongValues", Enum: &[]interface{}{"noint", "23", "42", "1.23"}, Type: kudoapi.IntegerValueType},
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)

	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := ParametersVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 5, len(res.Warnings)) // NotUsed Warnings
	assert.Equal(t, 4, len(res.Errors))
	assert.Equal(t, `parameter "EnumNoValues" is an enum but has no allowed values`, res.Errors[0])
	assert.Equal(t, `parameter "EnumWrongValues" has an invalid enum value: type is "integer" but format of "noint" is invalid: strconv.ParseInt: parsing "noint": invalid syntax`, res.Errors[1])
	assert.Equal(t, `parameter "EnumWrongValues" has an invalid enum value: type is "integer" but format of "1.23" is invalid: strconv.ParseInt: parsing "1.23": invalid syntax`, res.Errors[2])
	assert.Equal(t, `parameter "EnumWithDefault" has an invalid default value: value is "someOtherVal", but only allowed values are [someVal]`, res.Errors[3])
}

func TestMetadata(t *testing.T) {
	trueFlag := true
	params := []packages.Parameter{
		{Name: "InvalidGroup", Group: "some/group"},
		{Name: "InvalidAdvanced", Advanced: &trueFlag, Required: &trueFlag},
		{Name: "ValidAdvanced", Advanced: &trueFlag, Required: &trueFlag, Default: "someValue"},
		{Name: "AnotherValidAdvanced", Advanced: &trueFlag},
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)

	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := ParametersVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 4, len(res.Warnings)) // NotUsed Warnings
	assert.Equal(t, 3, len(res.Errors))
	assert.Equal(t, `parameter "InvalidGroup" has a group with invalid character '/'`, res.Errors[0])
	assert.Equal(t, `parameter "InvalidAdvanced" is marked as advanced, but also as required and has no default. An advanced parameter must either be optional or have a default value`, res.Errors[1])
	assert.Equal(t, `parameter "InvalidGroup" has a group "some/group" that is not defined in the group section`, res.Errors[2])
}
