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
	}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = `
{{.Params.Foo}}
{{.Params.Bar}}
{{.Bar}}
{{.Name}}
`
	operator := packages.OperatorFile{}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	verifier := ParametersVerifier{}
	warnings, errors := verifier.Verify(&pf)

	assert.Equal(t, 1, len(warnings))
	assert.Equal(t, `parameter "NotUsed" defined but not used.`, string(warnings[0]))
	assert.Equal(t, 2, len(errors))
	assert.Equal(t, `parameter "Bar" in template foo.yaml is not defined`, string(errors[0]))
	assert.Equal(t, `template foo.yaml defines an invalid implicit parameter "Bar"`, string(errors[1]))
}
