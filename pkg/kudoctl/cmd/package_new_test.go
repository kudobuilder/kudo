package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

func TestPackageNew(t *testing.T) {
	gfolder := "newop"

	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}
	cmd := newPackageNewCmd(fs, out)
	err := cmd.RunE(cmd, []string{"newop"})
	if err != nil {
		t.Fatal("unable to run package new command", err)
	}
	operatorFile := filepath.Join("operator", reader.OperatorFileName)
	paramFile := filepath.Join("operator", reader.ParamsFileName)

	// comparison golden files
	gOperatorFile := filepath.Join("testdata", gfolder, reader.OperatorFileName+".golden")
	gParamFile := filepath.Join("testdata", gfolder, reader.ParamsFileName+".golden")

	operator, _ := afero.ReadFile(fs, operatorFile)
	param, _ := afero.ReadFile(fs, paramFile)

	if *updateGolden {
		t.Logf("updating golden file %s", gOperatorFile)
		if err := ioutil.WriteFile(gOperatorFile, operator, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
		t.Logf("updating golden file %s", gParamFile)
		if err := ioutil.WriteFile(gParamFile, param, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	gOperator, err := ioutil.ReadFile(gOperatorFile)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	gParam, err := ioutil.ReadFile(gParamFile)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, operator, gOperator, "for golden file: %s", gOperatorFile)
	assert.Equal(t, param, gParam, "for golden file: %s", gParamFile)

}

func TestPackageNew_validation(t *testing.T) {
	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}

	var tests = []struct {
		name         string
		args         []string
		errorMessage string
	}{
		{name: "0 argument", args: []string{}, errorMessage: "expecting exactly one argument - directory of the operator or name of package"},
		{name: "2 arguments", args: []string{"1", "2"}, errorMessage: "expecting exactly one argument - directory of the operator or name of package"},
	}

	for _, test := range tests {
		cmd := newPackageNewCmd(fs, out)
		err := cmd.RunE(cmd, test.args)
		assert.EqualError(t, err, test.errorMessage)
	}

}

func TestPackageNew_Overwrite(t *testing.T) {

	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}
	cmd := newPackageNewCmd(fs, out)
	err := cmd.RunE(cmd, []string{"newop"})
	if err != nil {
		t.Fatal("unable to run package new command", err)
	}

	// no overwrite
	err = cmd.RunE(cmd, []string{"newop"})
	assert.EqualError(t, err, `folder "operator" already exists`)

	// overwrite with flag
	_ = cmd.Flags().Set("overwrite", "true")
	err = cmd.RunE(cmd, []string{"newop"})
	assert.Nil(t, err)
}
