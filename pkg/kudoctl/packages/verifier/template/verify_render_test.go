package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

func TestTemplateRenderVerifier(t *testing.T) {
	params := make([]v1beta1.Parameter, 0)
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

func TestTemplateRenderVerifier_InvalidYAML(t *testing.T) {
	params := make([]v1beta1.Parameter, 0)
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
		`decoding chunk "\napiVersion: batch/v1\nkind: Job\nmetadata:\n  name: backup\nspec:\n  template:\n    spec:\n      containers:\n        - name: backup\n`+
			`       restartPolicy: Never\n" failed: error converting YAML to JSON: yaml: line 10: did not find expected key`, res.Errors[0])
}

func TestTemplateRenderVerifierParameterTypes(t *testing.T) {
	params := []v1beta1.Parameter{
		v1beta1.Parameter{
			Name:    "labels",
			Default: convert.String("a: a\nb: b"),
			Type:    v1beta1.MapValueType,
		},
		v1beta1.Parameter{
			Name:    "containers",
			Default: convert.String("[a, b]"),
			Type:    v1beta1.ArrayValueType,
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
