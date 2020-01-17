package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTemplateParametersVerifier(t *testing.T) {
	params := []v1beta1.Parameter{
		{Name: "Foo"},
		{Name: "NotUsed"},
		{Name: "UsedViaRoot"},
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
{{.Params.Foo}}
{{.Params.Bar}}
{{.Bar}}
{{.Name}}
{{$.AppVersion}}
{{$.Params.UsedViaRoot}}
`
	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := ParametersVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 1, len(res.Warnings))
	assert.Equal(t, `parameter "NotUsed" defined but not used.`, res.Warnings[0])
	assert.Equal(t, 2, len(res.Errors))
	assert.Equal(t, `parameter "Bar" in template foo.yaml is not defined`, res.Errors[0])
	assert.Equal(t, `template foo.yaml defines an invalid implicit parameter "Bar"`, res.Errors[1])
}
