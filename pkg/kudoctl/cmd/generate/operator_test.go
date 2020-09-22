package generate

import (
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

func TestOperatorGenSafe(t *testing.T) {

	//	empty fs should be fine
	fs := afero.NewMemMapFs()
	err := CanGenerateOperator(fs, "operator", false)
	assert.Nil(t, err)

	// folder that doesn't exist should be fine
	err = CanGenerateOperator(fs, "operator", false)
	assert.Nil(t, err)

	_ = fs.Mkdir("operator", 0755)
	// folder that exist should fail
	err = CanGenerateOperator(fs, "operator", false)
	assert.NotNil(t, err)

	// folder that exist should not fail if overwrite
	err = CanGenerateOperator(fs, "operator", true)
	assert.Nil(t, err)
}

var (
	op1 = packages.OperatorFile{
		Name:            "foo",
		APIVersion:      reader.APIVersion,
		OperatorVersion: "0.1.0",
	}
	opFilename    = path.Join("operator", "operator.yaml")
	paramFilename = path.Join("operator", "params.yaml")
)

func TestOperator_Write(t *testing.T) {

	fs := afero.NewMemMapFs()

	err := Operator(fs, "operator", &op1, false)
	// no error on create
	assert.Nil(t, err)

	// results in operator file
	exists, _ := afero.Exists(fs, opFilename)
	assert.True(t, exists)
	// results in params file
	exists, _ = afero.Exists(fs, paramFilename)
	assert.True(t, exists)

	// test fail on existing
	err = Operator(fs, "operator", &op1, false)
	assert.Errorf(t, err, "folder 'operator' already exists")

	// test overwriting with no error
	err = Operator(fs, "operator", &op1, true)
	// no error on overwrite
	assert.Nil(t, err)

	// updating params file and testing params are not overwritten
	pf := packages.ParamsFile{
		APIVersion: "FOO",
		Parameters: []packages.Parameter{},
	}
	// replace param file with a marker "FOO" to test that we do NOT overwrite it
	err = writeParameters(fs, "operator", pf)
	assert.Nil(t, err)
	// test overwriting with no error
	err = Operator(fs, "operator", &op1, true)
	// no error on overwrite
	assert.Nil(t, err)
	parmfile, _ := afero.ReadFile(fs, paramFilename)
	assert.Contains(t, string(parmfile), "FOO")
}
