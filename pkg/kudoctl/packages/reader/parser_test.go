package reader

import (
	"errors"
	"testing"

	"gotest.tools/assert"
)

const (
	validOperator = `
apiVersion: kudo.dev/v1beta1
name: "first-operator"
operatorVersion: "0.1.0"
`
	validParams = `
apiVersion: kudo.dev/v1beta1
parameters:
  - name: param
    default: "value"
`
)

func TestParsePackageFile(t *testing.T) {
	tests := []struct {
		filePath    string
		fileContent string

		isOperator bool
		isParam    bool
		isTemplate bool

		expectedError error
	}{
		{"operator.yaml", validOperator, true, false, false, nil},
		{"params.yaml", validParams, false, true, false, nil},
		{"templates/pod-params.yaml", "", false, false, true, nil},
		{"templates/pod-operator.yaml", "", false, false, true, nil},
		{"templates/some-template.yaml", "", false, false, true, nil},
		{"operator.yaml", "", true, false, false, errors.New("failed to parse yaml into valid operator operator.yaml")},
	}

	for _, tt := range tests {
		tt := tt

		pf := newPackageFiles()

		err := parsePackageFile(tt.filePath, []byte(tt.fileContent), &pf)

		if tt.expectedError != nil {
			assert.Equal(t, tt.expectedError.Error(), err.Error())
			continue
		} else {
			assert.NilError(t, err)
		}

		if tt.isOperator {
			assert.Check(t, pf.Operator != nil, "%v was not parsed as an operator file", tt.filePath)
		}
		if tt.isParam {
			assert.Check(t, pf.Params != nil, "%v was not parsed as a param file", tt.filePath)
		}
		if tt.isTemplate {
			assert.Check(t, len(pf.Templates) > 0, "%v was not parsed as a template file", tt.filePath)

			//assert.Equal(t, pf.Templates[tt])
		}

	}
}
