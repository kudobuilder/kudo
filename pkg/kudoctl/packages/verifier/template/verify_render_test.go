package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
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
