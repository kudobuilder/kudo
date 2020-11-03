package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTemplateRenderVerifier(t *testing.T) {
	params := make([]packages.Parameter, 0)
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
{{ if eq }}
{{ end }}
`
	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := RenderVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, `error rendering template: template: foo.yaml:2:6: executing "foo.yaml" at <eq>: wrong number of args for eq: want at least 1 got 0`, res.Errors[0])
}

func TestTemplateRenderVerifier_Pipes(t *testing.T) {
	params := make([]packages.Parameter, 0)
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
correct: {{ .Pipes.existing }}
wrong: {{ .Pipes.inexistent }}
`
	operator := packages.OperatorFile{
		Tasks: []kudoapi.Task{
			{
				Name: "a-task",
				Kind: "Pipe",
				Spec: kudoapi.TaskSpec{
					PipeTaskSpec: kudoapi.PipeTaskSpec{
						Pipe: []kudoapi.PipeSpec{
							{
								Key: "existing",
							},
						},
					},
				},
			},
		},
		Plans: map[string]kudoapi.Plan{
			"a-plan": {
				Phases: []kudoapi.Phase{
					{
						Steps: []kudoapi.Step{
							{Tasks: []string{"a-task"}},
						},
					},
				},
			},
		},
	}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := RenderVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, `error rendering template: template: foo.yaml:3:16: executing "foo.yaml" at <.Pipes.inexistent>: `+
		`map has no entry for key "inexistent"`, res.Errors[0])
}

func TestTemplateRenderVerifier_InvalidYAML(t *testing.T) {
	params := make([]packages.Parameter, 0)
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
apiVersion: batch/v1
kind: Job
metadata:
  name: backup
spec:
  template:
    spec:
      containers:
        - name: backup
       restartPolicy: Never
`
	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := RenderVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t,
		`parsing rendered YAML from foo.yaml failed: decoding chunk "\napiVersion: batch/v1\nkind: Job\nmetadata:\n  name: backup\nspec:\n  template:\n    spec:\n      containers:\n        - name: backup\n`+
			`       restartPolicy: Never\n" failed: error converting YAML to JSON: yaml: line 10: did not find expected key`, res.Errors[0])
}

func TestTemplateRenderVerifierParameterTypes(t *testing.T) {
	params := []packages.Parameter{
		{
			Name:    "labels",
			Default: map[string]string{"a": "a", "b": "b"},
			Type:    kudoapi.MapValueType,
		},
		{
			Name:    "containers",
			Default: []string{"a", "b"},
			Type:    kudoapi.ArrayValueType,
		},
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
apiVersion: batch/v1
kind: Job
metadata:
  name: backup
  labels:
  {{ range $key, $value := .Params.labels }}
    {{ $key }}: {{ $value }}
  {{ end }}
spec:
  template:
    spec:
      containers:
        {{ range .Params.containers }}
        - name: {{ . }}
          restartPolicy: Never
        {{ end }}
`
	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := RenderVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 0, len(res.Errors))
}
