package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
)

func TestParamsList(t *testing.T) {
	tests := []struct {
		name       string
		outputType output.Type
		formatType string
		err        string
	}{
		{name: "list-output.txt"},
		{name: "list-output.yaml", outputType: output.TypeYAML, formatType: outputFormatList},
		{name: "list-output.json", outputType: output.TypeJSON, formatType: outputFormatList},
		{name: "schema-output.yaml", outputType: output.TypeYAML, formatType: outputFormatJSONSchema},
		{name: "schema-output.json", outputType: output.TypeJSON, formatType: outputFormatJSONSchema},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {

			file := tt.name
			out := &bytes.Buffer{}
			params := &packageListParamsCmd{fs: fs, out: out, Output: tt.outputType, Format: tt.formatType}

			cmd := newPackageListParamsCmd(params)

			if err := cmd.RunE(cmd, []string{"./testdata/listop"}); err != nil {
				t.Fatal(err)
			}

			gp := filepath.Join("testdata", file+".golden")

			if *updateGolden {
				t.Log("update golden file")

				//nolint:gosec
				if err := ioutil.WriteFile(gp, out.Bytes(), 0644); err != nil {
					t.Fatalf("failed to update golden file: %s", err)
				}
			}
			g, err := ioutil.ReadFile(gp)
			if err != nil {
				t.Fatalf("failed reading .golden: %s", err)
			}

			assert.Equal(t, string(g), out.String(), "output does not match .golden file %s", gp)

		})
	}

}
