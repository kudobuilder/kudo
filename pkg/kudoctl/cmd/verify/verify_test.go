package verify

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
		warnings, errors := verifier.Verify(tt.params)
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
		warnings, errors := verifier.Verify(tt.params)
		assert.Equal(t, tt.expectedWarnings, warnings, tt.name)
		assert.Equal(t, tt.expectedErrors, errors, tt.name)
	}
}
