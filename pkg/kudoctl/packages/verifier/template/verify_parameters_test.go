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
