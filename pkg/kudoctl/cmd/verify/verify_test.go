package verify

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/stretchr/testify/assert"
)

func TestDuplicateVerifier(t *testing.T) {
	tests := []struct {
		name             string
		params           []v1beta1.Parameter
		expectedWarnings ParamWarnings
		expectedErrors   ParamErrors
	}{
		{"no warning or error", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, nil, nil},
		{"duplicate parameter", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Foo"},
		}, nil, []ParamError{ParamError(fmt.Sprintf("parameter \"Foo\" has a duplicate"))}},
		{"duplicate with different casing", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "foo"},
		}, nil, ParamErrors{ParamError(fmt.Sprintf("parameter \"foo\" has a duplicate"))}},
	}

	verifier := DuplicateVerifier{}
	for _, tt := range tests {
		warnings, errors := verifier.Verify(packageFileForParams(tt.params))
		assert.Equal(t, tt.expectedWarnings, warnings)
		assert.Equal(t, tt.expectedErrors, errors)
	}
}

func TestInvalidCharVerifier(t *testing.T) {
	tests := []struct {
		name             string
		params           []v1beta1.Parameter
		expectedWarnings ParamWarnings
		expectedErrors   ParamErrors
	}{
		{"no warning or error", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, nil, nil},
		{"invalid character", []v1beta1.Parameter{
			{Name: "Foo:"},
			{Name: "Fighters,"},
		}, nil, []ParamError{ParamError("parameter \"Foo:\" contains invalid character ':'"), ParamError("parameter \"Fighters,\" contains invalid character ','")}},
	}

	verifier := InvalidCharVerifier{InvalidChars: ":,"}
	for _, tt := range tests {
		warnings, errors := verifier.Verify(packageFileForParams(tt.params))
		assert.Equal(t, tt.expectedWarnings, warnings, tt.name)
		assert.Equal(t, tt.expectedErrors, errors, tt.name)
	}
}

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
	verifier := TemplateReferenceVerifier{}
	warnings, errors := verifier.Verify(&pf)

	assert.Equal(t, 1, len(warnings))
	assert.Equal(t, `template "baz.yaml" is not used as a resource`, string(warnings[0]))
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
	verifier := TemplateParametersVerifier{}
	warnings, errors := verifier.Verify(&pf)

	assert.Equal(t, 1, len(warnings))
	assert.Equal(t, `parameter "NotUsed" defined but not used.`, string(warnings[0]))
	assert.Equal(t, 2, len(errors))
	assert.Equal(t, `parameter "Bar" in template foo.yaml is not defined`, string(errors[0]))
	assert.Equal(t, `template foo.yaml defines an invalid implicit parameter "Bar"`, string(errors[1]))
}

func packageFileForParams(params []v1beta1.Parameter) *packages.Files {
	p := packages.ParamsFile{
		Parameters: params,
	}
	return &packages.Files{
		Params: &p,
	}
}
