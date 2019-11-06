package verify

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestDuplicateVerifier_NoErrorVerify(t *testing.T) {

	goodParams := []v1beta1.Parameter{
		{Name: "Foo"},
		{Name: "Fighters"},
	}

	verifier := DuplicateVerifier{}

	warnings, errors := verifier.Verify(goodParams)
	assert.Nil(t, warnings)
	assert.Nil(t, errors)
}

func TestDuplicateVerifier_ErrorVerify(t *testing.T) {

	dupParams := []v1beta1.Parameter{
		{Name: "Foo"},
		{Name: "Foo"},
	}

	verifier := DuplicateVerifier{}

	warnings, errors := verifier.Verify(dupParams)
	assert.Nil(t, warnings)
	assert.Equal(t, ParamError(fmt.Sprintf("parameter \"Foo\" has a duplicate")), errors[0])
}

func TestDuplicateVerifier_CaseErrorVerify(t *testing.T) {

	dupParams := []v1beta1.Parameter{
		{Name: "Foo"},
		{Name: "foo"},
	}

	verifier := DuplicateVerifier{}

	warnings, errors := verifier.Verify(dupParams)
	assert.Nil(t, warnings)
	assert.Equal(t, ParamError(fmt.Sprintf("parameter \"foo\" has a duplicate")), errors[0])
}

func TestInvalidCharVerifier_GoodVerify(t *testing.T) {
	params := []v1beta1.Parameter{
		{Name: "Foo"},
		{Name: "Fighters"},
	}

	verifier := InvalidCharVerifier{InvalidChars: ":,"}
	warnings, errors := verifier.Verify(params)
	assert.Nil(t, warnings)
	assert.Nil(t, errors)
}

func TestInvalidCharVerifier_BadVerify(t *testing.T) {
	params := []v1beta1.Parameter{
		{Name: "Foo:"},
		{Name: "Fighters,"},
	}

	verifier := InvalidCharVerifier{InvalidChars: ":,"}
	warnings, errors := verifier.Verify(params)
	assert.Nil(t, warnings)
	assert.Equal(t, 2, len(errors))
	assert.Equal(t, ParamError("parameter \"Foo:\" has a the invalid char ':'"), errors[0])
}
