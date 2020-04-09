package verify

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestDuplicateVerifier(t *testing.T) {
	tests := []struct {
		name             string
		params           []packages.Parameter
		expectedWarnings []string
		expectedErrors   []string
	}{
		{"no warning or error", []packages.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, []string{}, []string{}},
		{"duplicate parameter", []packages.Parameter{
			{Name: "Foo"},
			{Name: "Foo"},
		}, []string{}, []string{"parameter \"Foo\" has a duplicate"}},
		{"duplicate with different casing", []packages.Parameter{
			{Name: "Foo"},
			{Name: "foo"},
		}, []string{}, []string{"parameter \"foo\" has a duplicate"}},
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
		params           []packages.Parameter
		expectedWarnings []string
		expectedErrors   []string
	}{
		{"no warning or error", []packages.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, []string{}, []string{}},
		{"invalid character", []packages.Parameter{
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

func packageFileForParams(params []packages.Parameter) *packages.Files {
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
		{"no warning or error with all versions", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.15",
			KUDOVersion:       "0.12.0",
			OperatorVersion:   "0.1.0",
		}, []string{}, []string{}},
		{"no warning or error without kudo version", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.15",
			OperatorVersion:   "0.1.0",
		}, []string{}, []string{}},
		{"kubernetesVersion required", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "",
			KUDOVersion:       "0.12.0",
			OperatorVersion:   "0.1.0",
		}, []string{}, []string{"\"kubernetesVersion\" is required and must be semver"}},
		{"kubernetesVersion must be semver", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.",
			KUDOVersion:       "0.12.0",
			OperatorVersion:   "0.1.0",
		}, []string{}, []string{"unable to parse \"kubernetesVersion\": Invalid Semantic Version"}},
		{"kubernetesVersion required", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.15",
			KUDOVersion:       "0.12.0",
			OperatorVersion:   "",
		}, []string{}, []string{"\"operatorVersion\" is required and must be semver"}},
		{"kubernetesVersion must be semver", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.15",
			KUDOVersion:       "0.12.0",
			OperatorVersion:   "0.1.",
		}, []string{}, []string{"unable to parse \"operatorVersion\": Invalid Semantic Version"}},
		{"kudoVersion must be semver", &packages.OperatorFile{
			APIVersion:        packages.APIVersion,
			Name:              "kafka",
			KubernetesVersion: "1.15",
			KUDOVersion:       "0.12.",
			OperatorVersion:   "0.1.0",
		}, []string{}, []string{"unable to parse \"kudoVersion\": Invalid Semantic Version"}},
		{"kubernetesVersion and OperatorVersion missing", &packages.OperatorFile{
			APIVersion:  packages.APIVersion,
			Name:        "kafka",
			KUDOVersion: "0.12.0",
		}, []string{}, []string{"\"operatorVersion\" is required and must be semver", "\"kubernetesVersion\" is required and must be semver"}},
		{"kubernetesVersion missing and OperatorVersion not semver", &packages.OperatorFile{
			APIVersion:      packages.APIVersion,
			Name:            "kafka",
			KUDOVersion:     "0.12.0",
			OperatorVersion: "0.1.",
		}, []string{}, []string{"unable to parse \"operatorVersion\": Invalid Semantic Version", "\"kubernetesVersion\" is required and must be semver"}},
	}

	verifier := VersionVerifier{}
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
