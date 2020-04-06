package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
		Tasks: []v1beta1.Task{
			{
				Name: "toggleTask",
				Kind: task.ToggleTaskKind,
				Spec: v1beta1.TaskSpec{
					ToggleTaskSpec: v1beta1.ToggleTaskSpec{Parameter: "Foo"},
				},
			},
			{
				Name: "toggleTaskNotDefinedParam",
				Kind: task.ToggleTaskKind,
				Spec: v1beta1.TaskSpec{
					ToggleTaskSpec: v1beta1.ToggleTaskSpec{Parameter: "NotDefined"},
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
