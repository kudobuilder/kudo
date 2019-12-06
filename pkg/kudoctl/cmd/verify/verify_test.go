package verify

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
	"github.com/stretchr/testify/assert"
)

func TestDuplicateVerifier(t *testing.T) {
	tests := []struct {
		name             string
		params           []v1beta1.Parameter
		expectedWarnings verifier.Warnings
		expectedErrors   verifier.Errors
	}{
		{"no warning or error", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, nil, nil},
		{"duplicate parameter", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Foo"},
		}, nil, []verifier.Error{verifier.Error(fmt.Sprintf("parameter \"Foo\" has a duplicate"))}},
		{"duplicate with different casing", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "foo"},
		}, nil, verifier.Errors{verifier.Error(fmt.Sprintf("parameter \"foo\" has a duplicate"))}},
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
		expectedWarnings verifier.Warnings
		expectedErrors   verifier.Errors
	}{
		{"no warning or error", []v1beta1.Parameter{
			{Name: "Foo"},
			{Name: "Fighters"},
		}, nil, nil},
		{"invalid character", []v1beta1.Parameter{
			{Name: "Foo:"},
			{Name: "Fighters,"},
		}, nil, []verifier.Error{verifier.Error("parameter \"Foo:\" contains invalid character ':'"), verifier.Error("parameter \"Fighters,\" contains invalid character ','")}},
	}

	verifier := InvalidCharVerifier{InvalidChars: ":,"}
	for _, tt := range tests {
		warnings, errors := verifier.Verify(packageFileForParams(tt.params))
		assert.Equal(t, tt.expectedWarnings, warnings, tt.name)
		assert.Equal(t, tt.expectedErrors, errors, tt.name)
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
