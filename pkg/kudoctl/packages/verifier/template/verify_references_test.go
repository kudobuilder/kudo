package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
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
		Kind: "Apply",
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
	res := verifier.Verify(&pf)

	assert.Equal(t, 1, len(res.Warnings))
	assert.Equal(t, `template "baz.yaml" is not referenced from any task`, res.Warnings[0])
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, `template "bar.yaml" required by foo but is not defined`, res.Errors[0])
}
