package generate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestAddParameter(t *testing.T) {
	goldenFile := "parameter"
	path := "/opt/zk" //nolint:goconst
	fs := afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../../packages/testdata/zk", "/opt")

	bar := "Bar"
	p := packages.Parameter{
		Name:    "Foo",
		Default: &bar,
	}

	err := AddParameter(fs, path, &p)
	assert.NoError(t, err)

	params, err := afero.ReadFile(fs, "/opt/zk/params.yaml")
	assert.NoError(t, err)

	gp := filepath.Join("testdata", goldenFile+".golden")

	if *updateGolden {
		t.Logf("updating golden file %s", goldenFile)

		//nolint:gosec
		if err := ioutil.WriteFile(gp, params, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	golden, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, golden, params, "for golden file: %s", gp)
}

func TestAddParameter_bad_path(t *testing.T) {
	path, _ := os.Getwd()
	fs := afero.OsFs{}

	bar := "Bar"
	p := packages.Parameter{
		Name:    "Foo",
		Default: &bar,
	}

	err := AddParameter(fs, path, &p)
	assert.Error(t, err)
}

func TestListParams(t *testing.T) {
	fs := afero.OsFs{}
	ps, err := ParameterNameList(fs, "../../packages/testdata/zk")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(ps))
	assert.Equal(t, "memory", ps[0])
}
