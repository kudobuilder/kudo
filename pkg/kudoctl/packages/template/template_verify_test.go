package template

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/stretchr/testify/assert"
)

func TestTemplateReferenceVerifier(t *testing.T) {
	params := []v1beta1.Parameter{}
	paramFile := packages.ParamsFile{Parameters: params}
	templates := make(map[string]string)
	templates["foo.yaml"] = "does not matter"
	templates["baz.yaml"] = "does not matter"

	resources := []string{"foo.yaml", "bar.yaml"}
	tasks := []v1beta1.Task{{
		Name: "foo",
		Kind: "",
		Spec: v1beta1.TaskSpec{
			ResourceTaskSpec: v1beta1.ResourceTaskSpec{Resources: resources},
		},
	}}
	operator := packages.OperatorFile{
		Tasks: tasks,
	}
	pf := packages.Files{
		Templates: templates,
		Operator:  &operator,
		Params:    &paramFile,
	}
	pf.Operator.Tasks = tasks
	verifier := ReferenceVerifier{}
	warnings, errors := verifier.Verify(&pf)

	assert.Equal(t, 1, len(warnings))
	assert.Equal(t, `template "baz.yaml" is not referenced from any task`, string(warnings[0]))
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, `template "bar.yaml" required by foo but not defined`, string(errors[0]))
}

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
