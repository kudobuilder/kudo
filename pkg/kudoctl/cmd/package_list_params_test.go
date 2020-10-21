package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParamsList(t *testing.T) {
	file := "params-list"
	out := &bytes.Buffer{}
	cmd := newPackageListParamsCmd(fs, out)
	if err := cmd.RunE(cmd, []string{"../packages/testdata/zk.tgz"}); err != nil {
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

	assert.Equal(t, string(g), out.String(), "yaml does not match .golden file %s", gp)
}
