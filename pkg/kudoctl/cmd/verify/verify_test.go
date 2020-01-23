package verify

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestDuplicateVerifier(t *testing.T) {
	tests := []struct {
		name             string
		params           []v1beta1.Parameter
		expectedWarnings []string
		expectedErrors   []string
	}{
		{"no warning or error", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, []string{}, []string{}},
		{"duplicate parameter", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Foo"},
		}, []string{}, []string{fmt.Sprintf("parameter \"Foo\" has a duplicate")}},
		{"duplicate with different casing", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "foo"},
		}, []string{}, []string{fmt.Sprintf("parameter \"foo\" has a duplicate")}},
	}

	verifier := DuplicateVerifier{}
	for _, tt := range tests {
		res := verifier.Verify(packageFileForParams(tt.params))
		assert.Equal(t, tt.expectedWarnings, res.Warnings)
		assert.Equal(t, tt.expectedErrors, res.Errors)
	}
}

func TestInvalidCharVerifier(t *testing.T) {
	tests := []struct {
		name             string
		params           []v1beta1.Parameter
		expectedWarnings []string
		expectedErrors   []string
	}{
		{"no warning or error", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, []string{}, []string{}},
		{"invalid character", []v1beta1.Parameter{
			{Name: "Foo:"},
			{Name: "Fighters,"},
		}, []string{}, []string{
			fmt.Sprintf("parameter %q %s", "Foo:", "contains invalid character ':'"),
			fmt.Sprintf("parameter %q %s", "Fighters,", "contains invalid character ','"),
		}},
	}

	verifier := InvalidCharVerifier{InvalidChars: ":,"}
	for _, tt := range tests {
		res := verifier.Verify(packageFileForParams(tt.params))
		assert.Equal(t, tt.expectedWarnings, res.Warnings, tt.name)
		assert.Equal(t, tt.expectedErrors, res.Errors, tt.name)
	}
}

func packageFileForParams(params []v1beta1.Parameter) *packages.Files {
	p := packages.ParamsFile{
		Parameters: params,
	}
	return &packages.Files{
		Params: &p,
	}
}

func TestK8sVersionVerifier(t *testing.T) {
	tests := []struct {
		name             string
		operatorFile     *packages.OperatorFile
		expectedWarnings []string
		expectedErrors   []string
	}{
		{"no warning or error", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.15",
		}, []string{}, []string{}},
		{"no warning or error", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "",
		}, []string{}, []string{"Unable to parse operators kubernetes version: Invalid Semantic Version"}},
	}

	verifier := K8sVersionVerifier{}
	for _, tt := range tests {
		res := verifier.Verify(packageFileForOperator(tt.operatorFile))
		assert.Equal(t, tt.expectedWarnings, res.Warnings, tt.name)
		assert.Equal(t, tt.expectedErrors, res.Errors, tt.name)
	}
}

func packageFileForOperator(op *packages.OperatorFile) *packages.Files {
	return &packages.Files{
		Operator: op,
	}
}
