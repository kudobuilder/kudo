package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageListTasks(t *testing.T) {
	file := "tasks-list"
	out := &bytes.Buffer{}
	cmd := newPackageListTasksCmd(fs, out)
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

	assert.Equal(t, out.String(), string(g), "does not match .golden file %s", gp)
}
