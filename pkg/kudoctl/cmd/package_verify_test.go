package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
)

func TestOperatorVerify(t *testing.T) {
	tests := []struct {
		name          string
		goldenFile    string
		output        output.Type
		expectedError string
	}{
		{name: "human readable output", goldenFile: "invalid-params.txt", output: ""},
		{name: "yaml output", goldenFile: "invalid-params.yaml", output: output.TypeYAML},
		{name: "json output", goldenFile: "invalid-params.json", output: output.TypeJSON},
		{name: "invalid output", expectedError: output.InvalidOutputError, output: "invalid"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			cmd := packageVerifyCmd{fs: fs, out: out, output: tt.output}

			if err := cmd.run("./testdata/invalidzk"); err != nil {
				if tt.expectedError != "" {
					assert.Equal(t, tt.expectedError, err.Error())
				}
				// We expect an error in all other cases, because we verify an invalid operator here
			}

			if tt.goldenFile != "" {
				gp := filepath.Join("testdata", tt.goldenFile+".golden")

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
			}
		})
	}
}
