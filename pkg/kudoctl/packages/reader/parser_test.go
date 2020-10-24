package reader

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		{filePath: "operator.yaml", fileContent: validOperator, isOperator: true},
		{filePath: "params.yaml", fileContent: validParams, isParam: true},
		{filePath: "templates/pod-params.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "templates/pod-operator.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "templates/some-template.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "templates/some/template.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "templates/some/nested/template2.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "./templates/some-template.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "./templates/with/subdirectory/some-template.yaml", isTemplate: true, fileContent: "not-empty"},
		{filePath: "operator.yaml", isOperator: true, expectedError: errors.New("failed to parse yaml into valid operator operator.yaml")},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.filePath, func(t *testing.T) {
			pf := newPackageFiles()

			err := parsePackageFile(tt.filePath, []byte(tt.fileContent), &pf)

			if tt.expectedError != nil {
				assert.NotNil(t, err, "Expected error but got none")
				if err != nil {
					assert.Equal(t, tt.expectedError.Error(), err.Error())
				}
				return
			}
			assert.Nil(t, err)

			if tt.isOperator {
				assert.NotNil(t, pf.Operator, "%v was not parsed as an operator file", tt.filePath)
			}
			if tt.isParam {
				assert.NotNil(t, pf.Params, "%v was not parsed as a param file", tt.filePath)
			}
			if tt.isTemplate {
				assert.Equal(t, 1, len(pf.Templates), "%v was not parsed as a template file", tt.filePath)

				fileName := filepath.Clean(tt.filePath)
				fileName = strings.TrimPrefix(fileName, "templates/")
				assert.NotEmpty(t, pf.Templates[fileName], "%v was not stored in template map", tt.filePath)
			}
		})
	}
}
